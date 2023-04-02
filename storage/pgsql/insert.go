package pgsql

import (
	"context"
	"database/sql"
	. "github.com/doug-martin/goqu/v9"
	"github.com/entropyinf/rest-layer/resource"
	"github.com/entropyinf/rest-layer/schema"
	"github.com/sirupsen/logrus"
	"reflect"
)

func (s store) Insert(ctx context.Context, items []*resource.Item) (err error) {
	var tx *sql.Tx
	if tx, err = s.db.Begin(); err != nil {
		return
	}
	defer rollbackIfErrExisted(tx, err)

	for _, item := range items {
		if err = s.insertOne(ctx, tx, item); err != nil {
			return
		}
	}

	err = tx.Commit()

	return
}

func (s store) insertOne(ctx context.Context, tx *sql.Tx, item *resource.Item) error {
	row := copyRow(item.Payload)
	row["etag"] = item.ETag

	useSerial := reflect.DeepEqual(s.schema.Fields["id"], schema.SerialID)

	if useSerial {
		delete(row, "id")
	}

	// Converting json node to string for adapting goqu framework
	if err := toJsonString(s.jsonFields, row); err != nil {
		return err
	}

	builder := Insert(s.table)

	if useSerial {
		builder = builder.Returning(L("id"))
	}

	sqlStr, args, err := builder.Prepared(true).Rows(row).ToSQL()
	if err != nil {
		return err
	}

	pgSqlStr := transformQueryPostgres(sqlStr)
	pgArgs := transformParamsPostgres(args)

	logrus.Traceln(pgSqlStr)
	logrus.Traceln(pgArgs...)

	result := tx.QueryRowContext(ctx, pgSqlStr, pgArgs...)

	if result.Err() != nil {
		return result.Err()
	}

	if useSerial {
		var id string
		if err = result.Scan(&id); err != nil {
			return err
		}
		item.Payload["id"] = id
		item.ID = id
	}

	return nil
}

func rollbackIfErrExisted(tx *sql.Tx, err error) {
	if err != nil {
		_ = tx.Rollback()
	}
}
