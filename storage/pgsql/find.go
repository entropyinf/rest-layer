package pgsql

import (
	"bytes"
	"context"
	"encoding/json"
	. "github.com/doug-martin/goqu/v9"
	"github.com/rs/rest-layer/resource"
	"github.com/rs/rest-layer/schema"
	"github.com/rs/rest-layer/schema/query"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

func (s store) Find(ctx context.Context, q *query.Query) (*resource.ItemList, error) {
	builder := From(s.table)
	buildSelects(q, builder)
	buildWheres(q, builder)
	buildSorts(q, builder)
	buildPagination(q, builder)

	sqlStr, args, err := builder.Prepared(true).ToSQL()
	if err != nil {
		return nil, err
	}

	pgSqlStr := transformQueryPostgres(sqlStr)
	pgArgs := transformParamsPostgres(args)

	logrus.Traceln(pgSqlStr)
	logrus.Traceln(pgArgs...)

	rows, err := s.db.QueryContext(ctx, pgSqlStr, pgArgs...)
	if err != nil {
		return nil, err
	}

	// result mapping
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	limit := 10
	if q.Window != nil {
		limit = q.Window.Limit
	}

	result := &resource.ItemList{
		Total: -1,
		Limit: limit,
		Items: []*resource.Item{},
	}
	for rows.Next() {
		rowMap := make(map[string]any)
		rowVals := make([]any, len(cols))
		rowValPtrs := make([]any, len(cols))
		var etag string

		for i, _ := range cols {
			rowValPtrs[i] = &rowVals[i]
		}

		err := rows.Scan(rowValPtrs...)
		if err != nil {
			return nil, err
		}

		for i, v := range rowVals {
			b, ok := v.([]byte)
			if ok {
				v = string(b)
			}

			if cols[i] == "etag" {
				etag = v.(string)
			} else {
				rowMap[cols[i]] = v
			}
		}

		// Converting itemID from int64 to int
		itemID := rowMap["id"]
		switch t := itemID.(type) {
		case int64:
			itemID = strconv.Itoa(int(t))
		}

		// Converting json string to json node
		for name, field := range s.jsonFields {
			if c, ok := rowMap[name]; ok {
				if jsonStr, ok := c.(string); ok {
					rowMap[name] = toJsonNode(&field, jsonStr)
				}
			}
		}

		item := &resource.Item{
			ID:      itemID,
			ETag:    etag,
			Payload: rowMap,
		}

		result.Items = append(result.Items, item)
	}

	return result, nil
}

func (s store) Count(ctx context.Context, q *query.Query) (int, error) {
	builder := From(s.table).Select(COUNT(Star()))
	buildWheres(q, builder)

	sqlStr, args, err := builder.Prepared(true).ToSQL()
	if err != nil {
		return 0, err
	}

	pgSqlStr := transformQueryPostgres(sqlStr)
	pgArgs := transformParamsPostgres(args)

	logrus.Traceln(pgSqlStr)
	logrus.Traceln(pgArgs...)

	row := s.db.QueryRowContext(ctx, pgSqlStr, pgArgs...)

	var count int
	err = row.Scan(&count)

	return count, err
}

func toJsonNode(field *schema.Field, cell string) any {
	switch field.Validator.(type) {
	case *schema.Object, *schema.Dict, nil:
		jsonNode := make(map[string]any)
		if err := json.Unmarshal([]byte(cell), &jsonNode); err != nil {
			return nil
		}
		return jsonNode
	case *schema.Array:
		jsonNode := make([]any, 0)
		if err := json.Unmarshal([]byte(cell), &jsonNode); err != nil {
			return nil
		}
		return jsonNode
	}
	return nil
}

func buildPagination(q *query.Query, builder *SelectDataset) {
	limit := 20
	offset := 0

	window := q.Window
	if window != nil {
		limit = window.Limit
		offset = window.Offset
	}
	*builder = *builder.Limit(uint(limit))
	*builder = *builder.Offset(uint(offset))
}

func buildSorts(q *query.Query, builder *SelectDataset) {
	for _, field := range q.Sort {
		if field.Reversed {
			*builder = *builder.Order(C(field.Name).Desc())
		} else {
			*builder = *builder.Order(C(field.Name).Asc())
		}
	}
}

func buildWheres(q *query.Query, builder *SelectDataset) {
	expressions := predicteToExpressions(q.Predicate)
	*builder = *builder.Where(expressions...)
}

func buildSelects(q *query.Query, builder *SelectDataset) {
	pj := q.Projection
	if pj == nil || len(pj) == 0 || hasStar(pj) {
		*builder = *builder.Select(Star())
		return
	}

	selectFields := make([]any, 0, len(pj))
	for _, field := range pj {
		if len(field.Alias) > 0 {
			selectFields = append(selectFields, I(field.Name).As(field.Alias), I(field.Name))
		} else {
			selectFields = append(selectFields, I(field.Name))
		}
	}

	*builder = *builder.Select(selectFields...)
}

func hasStar(pj query.Projection) bool {
	return match(pj, func(pf query.ProjectionField) bool {
		return pf.Name == "*"
	})
}

func match(pj query.Projection, predicate func(pf query.ProjectionField) bool) bool {
	for _, field := range pj {
		if predicate(field) {
			return true
		}
	}
	return false
}

func predicteToExpressions(q query.Predicate) (expressions []Expression) {
	for _, e := range q {
		switch t := e.(type) {
		case *query.And:
			for _, subExp := range *t {
				expressions = append(expressions, And(predicteToExpressions(query.Predicate{subExp})...))
			}
		case *query.Or:
			for _, subExp := range *t {
				expressions = append(expressions, Or(predicteToExpressions(query.Predicate{subExp})...))
			}
		case *query.In:
			expressions = append(expressions, C(postgresJsonbSupport(t.Field)).In(t.Values))
		case *query.NotIn:
			expressions = append(expressions, C(postgresJsonbSupport(t.Field)).NotIn(t.Values))
		case *query.Equal:
			expressions = append(expressions, C(postgresJsonbSupport(t.Field)).Eq(t.Value))
		case *query.NotEqual:
			expressions = append(expressions, C(postgresJsonbSupport(t.Field)).Neq(t.Value))
		case *query.GreaterThan:
			expressions = append(expressions, C(postgresJsonbSupport(t.Field)).Gt(t.Value))
		case *query.GreaterOrEqual:
			expressions = append(expressions, C(postgresJsonbSupport(t.Field)).Gte(t.Value))
		case *query.LowerThan:
			expressions = append(expressions, C(postgresJsonbSupport(t.Field)).Lt(t.Value))
		case *query.LowerOrEqual:
			expressions = append(expressions, C(postgresJsonbSupport(t.Field)).Lte(t.Value))
		case *query.Regex:
			expressions = append(expressions, C(postgresJsonbSupport(t.Field)).RegexpLike(t.Value))

		default:
			logrus.Warnln("not supported predicate. ignored")
		}
	}
	return
}

func postgresJsonbSupport(field string) string {
	if !strings.Contains(field, ".") {
		return field
	}

	strs := strings.Split(field, ".")
	buf := bytes.Buffer{}
	buf.WriteString("jsonb_extract_path_text(")
	buf.WriteString(strs[0])
	buf.WriteRune(',')

	substr := strs[1:]
	length := len(substr)
	for i, str := range substr {
		buf.WriteRune('\'')
		buf.WriteString(str)
		buf.WriteRune('\'')

		if i < length-1 {
			buf.WriteRune(',')
		}
	}
	buf.WriteRune(')')

	return buf.String()
}
