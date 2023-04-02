package pgsql

import (
	"context"
	"database/sql"
	"github.com/entropyinf/rest-layer/resource"
	"github.com/entropyinf/rest-layer/schema"
	"github.com/sirupsen/logrus"
)

type Option func(s *store)

type store struct {
	table      string
	db         *sql.DB
	schema     *schema.Schema
	jsonFields schema.Fields
}

func NewStore(table string, db *sql.DB, sc *schema.Schema, options ...Option) resource.Storer {
	s := &store{
		table:      table,
		db:         db,
		schema:     sc,
		jsonFields: getJsonFields(sc.Fields),
	}

	for _, opt := range options {
		opt(s)
	}

	return s
}

func getJsonFields(fields schema.Fields) schema.Fields {
	jsonColumns := make(map[string]schema.Field, 0)
	for name, field := range fields {
		switch field.Validator.(type) {
		case *schema.Object, *schema.Array, *schema.Dict:
			jsonColumns[name] = field
		case nil:
			if field.Schema != nil {
				jsonColumns[name] = field
			}
		}
	}
	return jsonColumns
}

func AutoMigrate() Option {
	return func(s *store) {
		err := s.Migrate(context.TODO(), s.schema)
		if err != nil {
			logrus.Warnln(err)
		}
	}
}
