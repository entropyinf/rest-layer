package rest

import (
	"testing"

	"github.com/entropyinf/rest-layer/resource"
	mem "github.com/entropyinf/rest-layer/resource/testing/mem"
	"github.com/entropyinf/rest-layer/schema"
	"github.com/stretchr/testify/assert"
)

func TestResourcePathValues(t *testing.T) {
	index := resource.NewIndex()
	users := index.Bind("users", schema.Schema{
		Fields: schema.Fields{
			"id": {
				Validator: &schema.String{},
			},
		},
	}, mem.NewHandler(), resource.DefaultConf)
	posts := users.Bind("posts", "user", schema.Schema{
		Fields: schema.Fields{
			"id": {
				Validator: &schema.Integer{},
			},
			"user": {
				Validator: &schema.Reference{Path: "users"},
			},
		},
	}, mem.NewHandler(), resource.DefaultConf)
	p := ResourcePath{
		&ResourcePathComponent{
			Name:     "users",
			Field:    "user",
			Value:    "john",
			Resource: users,
		},
		&ResourcePathComponent{
			Name:     "posts",
			Field:    "id",
			Value:    "123",
			Resource: posts,
		},
	}
	assert.Equal(t, map[string]interface{}{"id": "123", "user": "john"}, p.Values())
}

func TestResourcePathInvalidValues(t *testing.T) {
	index := resource.NewIndex()
	users := index.Bind("users", schema.Schema{
		Fields: schema.Fields{
			"id": {
				Validator: &schema.String{},
			},
		},
	}, mem.NewHandler(), resource.DefaultConf)
	posts := users.Bind("posts", "user", schema.Schema{
		Fields: schema.Fields{
			"id": {
				Validator: &schema.Integer{},
			},
			"user": {
				Validator: &schema.Reference{Path: "users"},
			},
		},
	}, mem.NewHandler(), resource.DefaultConf)
	p := ResourcePath{
		&ResourcePathComponent{
			Name:     "users",
			Field:    "user",
			Value:    "john",
			Resource: users,
		},
		&ResourcePathComponent{
			Name:     "posts",
			Field:    "id",
			Value:    "123",
			Resource: posts,
		},
		&ResourcePathComponent{
			Name:     "users1",
			Field:    "user1",
			Value:    "john1",
			Resource: posts,
		},
	}
	assert.Equal(t, map[string]interface{}{"id": "123"}, p.Values())
}

func TestResourcePathAppend(t *testing.T) {
	index := resource.NewIndex()
	users := index.Bind("users", schema.Schema{
		Fields: schema.Fields{
			"id": {
				Validator: &schema.String{},
			},
		},
	}, mem.NewHandler(), resource.DefaultConf)
	posts := users.Bind("posts", "user", schema.Schema{
		Fields: schema.Fields{
			"id": {
				Validator: &schema.Integer{},
			},
			"user": {
				Validator: &schema.Reference{Path: "users"},
			},
		},
	}, mem.NewHandler(), resource.DefaultConf)
	p := ResourcePath{}
	err := p.append(users, "user", 123, "users")
	assert.EqualError(t, err, "not a string")
	err = p.append(users, "user", "john", "users")
	assert.NoError(t, err)
	err = p.append(posts, "id", "123", "posts")
	assert.EqualError(t, err, "not an integer")
	err = p.append(posts, "id", 123, "posts")
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"id": 123, "user": "john"}, p.Values())

}

func TestResourcePathPrepend(t *testing.T) {
	p := ResourcePath{
		&ResourcePathComponent{
			Name:  "users",
			Field: "user",
			Value: "john",
		},
	}
	p.Prepend(nil, "foo", "bar")
	assert.Equal(t, ResourcePath{
		&ResourcePathComponent{
			Field: "foo",
			Value: "bar",
		},
		&ResourcePathComponent{
			Name:  "users",
			Field: "user",
			Value: "john",
		},
	}, p)
}
