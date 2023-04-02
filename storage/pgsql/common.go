package pgsql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	"github.com/entropyinf/rest-layer/schema"
	"reflect"
)

func copyRow(row map[string]any) map[string]any {
	out := make(map[string]any)
	for k, v := range row {
		out[k] = v
	}
	return out
}

func toJsonString(jsonFields schema.Fields, row map[string]any) error {
	if len(jsonFields) == 0 {
		return nil
	}

	buf := bytes.Buffer{}
	for name := range jsonFields {
		if col, ok := row[name]; ok {
			if err := json.NewEncoder(&buf).Encode(col); err != nil {
				return err
			}
			row[name] = buf.String()
			buf.Reset()
		}
	}

	return nil
}

func transformParamsPostgres(sqlParams []any) []any {
	var newSqlParams []any

	for _, p := range sqlParams {
		t := reflect.TypeOf(p)

		if t == nil {
			newSqlParams = append(newSqlParams, p)
		} else {
			switch t.Kind() {
			case reflect.Slice, reflect.Array:
				newSqlParams = append(newSqlParams, pq.Array(p))
			default:
				newSqlParams = append(newSqlParams, p)
			}
		}
	}

	return newSqlParams
}
func transformQueryPostgres(sqlQuery string) (newSqlQuery string) {
	idx := 1
	for _, ch := range sqlQuery {
		if ch == '?' {
			newSqlQuery += fmt.Sprintf("$%d", idx)
			idx++
		} else {
			newSqlQuery += string(ch)
		}

	}
	return
}
