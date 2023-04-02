package pgsql

import (
	"context"
	"database/sql"
	. "github.com/doug-martin/goqu/v9"
	"github.com/entropyinf/rest-layer/schema/query"
	"github.com/sirupsen/logrus"
)

func (s store) Clear(ctx context.Context, q *query.Query) (count int, err error) {
	var tx *sql.Tx

	tx, err = s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer rollbackIfErrExisted(tx, err)

	builder := Delete(s.table)

	buildDeleteWheres(q, builder)

	sqlStr, args, err := builder.Prepared(true).ToSQL()
	if err != nil {
		return
	}

	pgSqlStr := transformQueryPostgres(sqlStr)
	pgArgs := transformParamsPostgres(args)

	logrus.Traceln(pgSqlStr)
	logrus.Traceln(pgArgs...)

	res, err := tx.ExecContext(ctx, pgSqlStr, pgArgs...)
	if err != nil {
		return 0, err
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		return
	}

	return int(cnt), err
}
func buildDeleteWheres(q *query.Query, builder *DeleteDataset) {
	expressions := predicteToExpressions(q.Predicate)
	*builder = *builder.Where(expressions...)
}
