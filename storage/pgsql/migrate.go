package pgsql

import (
	"context"
	"fmt"
	"github.com/rs/rest-layer/schema"
	"github.com/sirupsen/logrus"
	"log"
	schemax "mall/pkg/schema"
	"reflect"
	"strings"
)

func (s store) Migrate(ctx context.Context, sc *schema.Schema) (err error) {
	sqlQuery, sqlParams, err := buildCreateQuery(s.table, sc)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, sqlQuery, sqlParams...)
	return err
}

func buildCreateQuery(tableName string, s *schema.Schema) (sqlQuery string, sqlParams []any, err error) {
	schemaQuery, schemaParams, err := buildCreateTable(s)
	if err != nil {
		return "", []any{}, err
	}

	sqlQuery = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s,PRIMARY KEY(id))", tableName, schemaQuery)

	logrus.Traceln(sqlQuery)

	sqlParams = append(sqlParams, schemaParams...)

	return transformQueryPostgres(sqlQuery), transformParamsPostgres(sqlParams), nil
}

func buildCreateTable(s *schema.Schema) (sqlQuery string, sqlParams []any, err error) {
	fieldStrings := make([]string, 0, len(s.Fields))

	for fieldName, field := range s.Fields {
		if fieldName == "id" && reflect.DeepEqual(field, schemax.SerialID) {
			fieldStrings = append(fieldStrings, "id SERIAL")
			continue
		}

		fieldName = `"` + fieldName + `"`
		switch f := field.Validator.(type) {
		case *schema.String:
			if f.MaxLen > 0 {
				fieldStrings = append(fieldStrings, fmt.Sprintf("%s VARCHAR(%d)", fieldName, f.MaxLen))
			} else {
				fieldStrings = append(fieldStrings, fmt.Sprintf("%s VARCHAR", fieldName))
			}
		case *schema.Integer:
			fieldStrings = append(fieldStrings, fmt.Sprintf("%s %s", fieldName, getIntegerScale(f)))
		case *schema.Float:
			fieldStrings = append(fieldStrings, fmt.Sprintf("%s DECIMAL", fieldName))
		case *schema.Bool:
			fieldStrings = append(fieldStrings, fmt.Sprintf("%s BOOLEAN", fieldName))
		case *schema.Time:
			fieldStrings = append(fieldStrings, fmt.Sprintf("%s TIMESTAMP", fieldName))
		case *schema.URL, *schema.IP, *schema.Password:
			fieldStrings = append(fieldStrings, fmt.Sprintf("%s VARCHAR", fieldName))
		case *schema.Reference:
			fieldStrings = append(fieldStrings, fmt.Sprintf("%s BIGINT", fieldName))
		case *schema.Object, *schema.Dict, *schema.Array:
			fieldStrings = append(fieldStrings, fmt.Sprintf("%s JSONB", fieldName))
		case nil:
			log.Fatalln("validator required")
		default:
			log.Fatalln("unsupported types of " + fieldName)
		}
	}

	fieldStrings = append(fieldStrings, "etag CHAR(32)")

	return strings.Join(fieldStrings, ","), []any{}, nil
}

func getIntegerScale(f *schema.Integer) string {
	if f.Boundaries == nil {
		return "INTEGER"
	}
	if f.Boundaries.Max == 0 || f.Boundaries.Max > 1<<31-1 {
		return "BIGINT"
	}
	if f.Boundaries.Max > 1<<15-1 {
		return "INTEGER"
	}
	return "SMALLINT"
}
