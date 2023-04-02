package pgsql

import (
	"context"
	. "github.com/doug-martin/goqu/v9"
	"github.com/rs/rest-layer/resource"
	"github.com/rs/rest-layer/rest"
	"github.com/sirupsen/logrus"
)

func (s store) Delete(ctx context.Context, item *resource.Item) error {
	sqlStr, args, err := Delete(s.table).Where(L("id").Eq(item.ID), L("etag").Eq(item.ETag)).Prepared(true).ToSQL()
	if err != nil {
		return err
	}

	pgSqlStr := transformQueryPostgres(sqlStr)
	pgArgs := transformParamsPostgres(args)

	logrus.Traceln(pgSqlStr)
	logrus.Traceln(pgArgs...)

	affect, err := s.db.ExecContext(ctx, pgSqlStr, pgArgs...)
	if err != nil {
		return err
	}

	count, err := affect.RowsAffected()
	if err != nil {
		return err
	}

	if count != 1 {
		return rest.ErrPreconditionFailed
	}

	return nil
}
