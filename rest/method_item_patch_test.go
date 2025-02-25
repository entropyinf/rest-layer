package rest_test

import (
	"bytes"
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/entropyinf/rest-layer/resource"
	"github.com/entropyinf/rest-layer/resource/testing/mem"
	"github.com/entropyinf/rest-layer/schema"
	"github.com/entropyinf/rest-layer/schema/query"
)

func TestPatchItem(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	sharedInit := func() *requestTestVars {
		s1 := mem.NewHandler()
		s1.Insert(context.Background(), []*resource.Item{
			{ID: "1", ETag: "a", Updated: now, Payload: map[string]interface{}{"id": "1", "foo": "odd", "bar": "baz"}},
			{ID: "2", ETag: "b", Updated: yesterday, Payload: map[string]interface{}{"id": "2", "foo": "even", "bar": "baz"}},
			{ID: "3", ETag: "c", Updated: yesterday, Payload: map[string]interface{}{"id": "3", "foo": "odd", "bar": "baz"}},
		})
		s2 := mem.NewHandler()
		s2.Insert(context.Background(), []*resource.Item{
			{ID: "1", ETag: "d", Updated: now, Payload: map[string]interface{}{"id": "1", "foo": "3"}},
		})

		idx := resource.NewIndex()
		foo := idx.Bind("foo", schema.Schema{
			Fields: schema.Fields{
				"id":  {Sortable: true, Filterable: true},
				"foo": {Filterable: true},
				"bar": {Filterable: true},
			},
		}, s1, resource.DefaultConf)
		foo.Bind("sub", "foo", schema.Schema{
			Fields: schema.Fields{
				"id":  {Sortable: true, Filterable: true},
				"foo": {Filterable: true, Validator: &schema.Reference{Path: "foo"}},
			},
		}, s2, resource.DefaultConf)

		return &requestTestVars{
			Index:   idx,
			Storers: map[string]resource.Storer{"foo": s1, "foo.sub": s2},
		}
	}
	checkPayload := func(name string, id interface{}, payload map[string]interface{}) requestCheckerFunc {
		return func(t *testing.T, vars *requestTestVars) {
			var item *resource.Item

			s := vars.Storers[name]
			q := query.Query{Predicate: query.Predicate{&query.Equal{Field: "id", Value: id}}, Window: &query.Window{Limit: 1}}
			if items, err := s.Find(context.Background(), &q); err != nil {
				t.Errorf("s.Find failed: %s", err)
				return
			} else if len(items.Items) != 1 {
				t.Errorf("item with ID %v not found", id)
				return
			} else {
				item = items.Items[0]
			}
			if !reflect.DeepEqual(payload, item.Payload) {
				t.Errorf("Unexpected stored payload for item %v:\nexpect: %#v\ngot: %#v", id, payload, item.Payload)
			}
		}
	}

	tests := map[string]requestTest{
		`NoStorage`: {
			// FIXME: For NoStorage, it's probably better to error early (during Bind).
			Init: func() *requestTestVars {
				index := resource.NewIndex()
				index.Bind("foo", schema.Schema{}, nil, resource.DefaultConf)
				return &requestTestVars{Index: index}
			},
			NewRequest: func() (*http.Request, error) {
				return http.NewRequest("PATCH", "/foo/1", nil)
			},
			ResponseCode: http.StatusNotImplemented,
			ResponseBody: `{"code": 501, "message": "No Storage Defined"}`,
		},
		`pathID:not-found`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`{"foo": "baz"}`))
				return http.NewRequest("PATCH", `/foo/66`, body)
			},
			ResponseCode: http.StatusNotFound,
			ResponseBody: `{"code": 404, "message": "Not Found"}`,
		},
		`pathID:found,body:invalid-json`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`invalid`))
				return http.NewRequest("PATCH", "/foo/2", body)
			},
			ResponseCode: http.StatusBadRequest,
			ResponseBody: `{
				"code": 400,
				"message": "Malformed body: invalid character 'i' looking for beginning of value"
			}`,
			ExtraTest: checkPayload("foo", "2", map[string]interface{}{"id": "2", "foo": "even", "bar": "baz"}),
		},
		`pathID:found,body:invalid-field`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`{"invalid": true}`))
				return http.NewRequest("PATCH", "/foo/2", body)
			},
			ResponseCode: http.StatusUnprocessableEntity,
			ResponseBody: `{
				"code":422,
				"message": "Document contains error(s)",
				"issues": {
					"invalid": ["invalid field"]
				}
			}`,
			ExtraTest: checkPayload("foo", "2", map[string]interface{}{"id": "2", "foo": "even", "bar": "baz"}),
		},
		`pathID:found,body:alter-id`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`{"id": "3"}`))
				return http.NewRequest("PATCH", "/foo/2", body)
			},
			ResponseCode: http.StatusUnprocessableEntity,
			ResponseBody: `{
				"code":422,
				"message": "Cannot change document ID"
			}`,
			ExtraTest: checkPayload("foo", "2", map[string]interface{}{"id": "2", "foo": "even", "bar": "baz"}),
		},
		`pathID:found,body:valid`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`{"foo": "baz"}`))
				return http.NewRequest("PATCH", "/foo/2", body)
			},
			ResponseCode:   http.StatusOK,
			ResponseBody:   `{"id": "2", "foo": "baz", "bar": "baz"}`,
			ResponseHeader: http.Header{"Etag": []string{`W/"53c7f8b8a84dd407e1491f5339fca757"`}},
			ExtraTest:      checkPayload("foo", "2", map[string]interface{}{"id": "2", "foo": "baz", "bar": "baz"}),
		},
		`pathID:found,body:valid:minimal`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`{"foo": "baz"}`))
				r, err := http.NewRequest("PATCH", "/foo/2", body)
				r.Header.Set("Prefer", "return=minimal")
				return r, err
			},
			ResponseCode:   http.StatusNoContent,
			ResponseBody:   ``,
			ResponseHeader: http.Header{"Etag": []string{`W/"53c7f8b8a84dd407e1491f5339fca757"`}},
			ExtraTest:      checkPayload("foo", "2", map[string]interface{}{"id": "2", "foo": "baz", "bar": "baz"}),
		},
		`pathID:found,body:valid,fields:invalid`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`{"foo": "baz"}`))
				return http.NewRequest("PATCH", "/foo/2?fields=invalid", body)
			},
			ResponseCode: http.StatusUnprocessableEntity,
			ResponseBody: `{
				"code": 422,
				"message": "URL parameters contain error(s)",
				"issues": {
					"fields": ["invalid: unknown field"]
				}
			}`,
			ExtraTest: checkPayload("foo", "2", map[string]interface{}{"id": "2", "foo": "even", "bar": "baz"}),
		},
		`pathID:found,body:valid,fields:valid`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`{"foo": "baz"}`))
				return http.NewRequest("PATCH", "/foo/2?fields=foo", body)
			},
			ResponseCode: http.StatusOK,
			ResponseBody: `{"foo": "baz"}`,
			ExtraTest:    checkPayload("foo", "2", map[string]interface{}{"id": "2", "foo": "baz", "bar": "baz"}),
		},
		`pathID:found,body:valid,header["If-Match"]:not-matching`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`{"foo": "baz"}`))
				r, err := http.NewRequest("PATCH", "/foo/2", body)
				if err != nil {
					return nil, err
				}
				r.Header.Set("If-Match", "W/x")
				return r, nil
			},
			ResponseCode: http.StatusPreconditionFailed,
			ResponseBody: `{"code": 412, "message": "Precondition Failed"}`,
			ExtraTest:    checkPayload("foo", "2", map[string]interface{}{"id": "2", "foo": "even", "bar": "baz"}),
		},
		`pathID:found,body:valid,header["If-Match"]:matching`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`{"foo": "baz"}`))
				r, err := http.NewRequest("PATCH", "/foo/2", body)
				if err != nil {
					return nil, err
				}
				r.Header.Set("If-Match", "W/b")
				return r, nil
			},
			ResponseCode: http.StatusOK,
			ResponseBody: `{"id": "2", "foo": "baz", "bar": "baz"}`,
			ExtraTest:    checkPayload("foo", "2", map[string]interface{}{"id": "2", "foo": "baz", "bar": "baz"}),
		},
		`pathID:found,body:valid,header["If-Unmodified-Since"]:invalid`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`{"foo": "baz"}`))
				r, err := http.NewRequest("PATCH", "/foo/1", body)
				if err != nil {
					return nil, err
				}
				r.Header.Set("If-Unmodified-Since", "invalid")
				return r, nil
			},
			ResponseCode: http.StatusBadRequest,
			ResponseBody: `{"code": 400, "message": "Invalid If-Unmodified-Since header"}`,
			ExtraTest:    checkPayload("foo", "1", map[string]interface{}{"id": "1", "foo": "odd", "bar": "baz"}),
		},
		`pathID:found,body:valid,header["If-Unmodified-Since"]:not-matching`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`{"foo": "baz"}`))
				r, err := http.NewRequest("PATCH", "/foo/1", body)
				if err != nil {
					return nil, err
				}
				r.Header.Set("If-Unmodified-Since", yesterday.Format(time.RFC1123))
				return r, nil
			},
			ResponseCode: http.StatusPreconditionFailed,
			ResponseBody: `{"code": 412, "message": "Precondition Failed"}`,
			ExtraTest:    checkPayload("foo", "1", map[string]interface{}{"id": "1", "foo": "odd", "bar": "baz"}),
		},
		`parentPathID:found,pathID:found,body:alter-parent-id`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`{"foo": "2"}`))
				r, err := http.NewRequest("PATCH", "/foo/3/sub/1", body)
				if err != nil {
					return nil, err
				}
				return r, nil
			},
			ResponseCode: http.StatusOK,
			ResponseBody: `{"id": "1", "foo": "2"}`,
			ExtraTest:    checkPayload("foo.sub", "1", map[string]interface{}{"id": "1", "foo": "2"}),
		},
	}

	for n, tc := range tests {
		tc := tc // capture range variable
		t.Run(n, tc.Test)
	}
}

func TestJSONPatchItem(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	sharedInit := func() *requestTestVars {
		s1 := mem.NewHandler()
		s1.Insert(context.Background(), []*resource.Item{
			{ID: "1", ETag: "a", Updated: now, Payload: map[string]interface{}{"id": "1", "foo": "odd", "bar": "baz"}},
			{ID: "2", ETag: "b", Updated: yesterday, Payload: map[string]interface{}{"id": "2", "foo": "even", "bar": "baz"}},
			{ID: "3", ETag: "c", Updated: yesterday, Payload: map[string]interface{}{"id": "3", "foo": "odd", "bar": "baz"}},
			{ID: "4", ETag: "d", Updated: yesterday, Payload: map[string]interface{}{"id": "4", "foo": "odd"}},
			{ID: "5", ETag: "e", Updated: yesterday, Payload: map[string]interface{}{"id": "5", "foo": "odd", "oar": map[string]interface{}{"a": "original"}}},
			{ID: "6", ETag: "f", Updated: yesterday, Payload: map[string]interface{}{"id": "6", "foo": "odd", "aar": []string{"value-1"}}},
		})
		s2 := mem.NewHandler()
		s2.Insert(context.Background(), []*resource.Item{
			{ID: "1", ETag: "d", Updated: now, Payload: map[string]interface{}{"id": "1", "foo": "3"}},
		})

		idx := resource.NewIndex()
		foo := idx.Bind("foo", schema.Schema{
			Fields: schema.Fields{
				"id":  {Sortable: true, Filterable: true},
				"foo": {Filterable: true},
				"bar": {Filterable: true},
				"oar": {
					Validator: &schema.Object{
						Schema: &schema.Schema{
							Fields: schema.Fields{
								"a": {Validator: &schema.String{}},
								"b": {Validator: &schema.String{}},
							},
						},
					},
				},
				"aar": {
					Validator: &schema.Array{
						Values: schema.Field{
							Validator: &schema.String{},
						},
					},
				},
			},
		}, s1, resource.DefaultConf)
		foo.Bind("sub", "foo", schema.Schema{
			Fields: schema.Fields{
				"id":  {Sortable: true, Filterable: true},
				"foo": {Filterable: true, Validator: &schema.Reference{Path: "foo"}},
			},
		}, s2, resource.DefaultConf)

		return &requestTestVars{
			Index:   idx,
			Storers: map[string]resource.Storer{"foo": s1, "foo.sub": s2},
		}
	}
	checkPayload := func(name string, id interface{}, payload map[string]interface{}) requestCheckerFunc {
		return func(t *testing.T, vars *requestTestVars) {
			var item *resource.Item

			s := vars.Storers[name]
			q := query.Query{Predicate: query.Predicate{&query.Equal{Field: "id", Value: id}}, Window: &query.Window{Limit: 1}}
			if items, err := s.Find(context.Background(), &q); err != nil {
				t.Errorf("s.Find failed: %s", err)
				return
			} else if len(items.Items) != 1 {
				t.Errorf("item with ID %v not found", id)
				return
			} else {
				item = items.Items[0]
			}
			if !reflect.DeepEqual(payload, item.Payload) {
				t.Errorf("Unexpected stored payload for item %v:\nexpect: %#v\ngot: %#v", id, payload, item.Payload)
			}
		}
	}

	tests := map[string]requestTest{
		`NoStorage`: {
			// FIXME: For NoStorage, it's probably better to error early (during Bind).
			Init: func() *requestTestVars {
				index := resource.NewIndex()
				index.Bind("foo", schema.Schema{}, nil, resource.DefaultConf)
				return &requestTestVars{Index: index}
			},
			NewRequest: func() (*http.Request, error) {
				r, err := http.NewRequest("PATCH", "/foo/1", nil)
				r.Header.Set("Content-Type", "application/json-patch+json")
				return r, err
			},
			ResponseCode: http.StatusNotImplemented,
			ResponseBody: `{"code": 501, "message": "No Storage Defined"}`,
		},
		`pathID:not-found`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`{"foo": "baz"}`))
				r, err := http.NewRequest("PATCH", `/foo/66`, body)
				r.Header.Set("Content-Type", "application/json-patch+json")
				return r, err
			},
			ResponseCode: http.StatusNotFound,
			ResponseBody: `{"code": 404, "message": "Not Found"}`,
		},
		`pathID:found,body:invalid-json`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`invalid`))
				r, err := http.NewRequest("PATCH", "/foo/2", body)
				r.Header.Set("Content-Type", "application/json-patch+json")
				return r, err
			},
			ResponseCode: http.StatusBadRequest,
			ResponseBody: `{
				"code": 400,
				"message": "Malformed patch document: invalid character 'i' looking for beginning of value"
			}`,
			ExtraTest: checkPayload("foo", "2", map[string]interface{}{"id": "2", "foo": "even", "bar": "baz"}),
		},
		`pathID:found,body:invalid-field`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`[
					{
							"op": "add",
							"path": "/invalid",
							"value": true
					}
			]`))
				r, err := http.NewRequest("PATCH", "/foo/2", body)
				r.Header.Set("Content-Type", "application/json-patch+json")
				return r, err
			},
			ResponseCode: http.StatusUnprocessableEntity,
			ResponseBody: `{
				"code":422,
				"message": "Document contains error(s)",
				"issues": {
					"invalid": ["invalid field"]
				}
			}`,
			ExtraTest: checkPayload("foo", "2", map[string]interface{}{"id": "2", "foo": "even", "bar": "baz"}),
		},
		`pathID:found,body:alter-id`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`[
					{
						"op": "replace",
						"path": "/id",
						"value": "3"
					}
				]`))
				r, err := http.NewRequest("PATCH", "/foo/2", body)
				r.Header.Set("Content-Type", "application/json-patch+json")
				return r, err
			},
			ResponseCode: http.StatusUnprocessableEntity,
			ResponseBody: `{
				"code":422,
				"message": "Cannot change document ID"
			}`,
			ExtraTest: checkPayload("foo", "2", map[string]interface{}{"id": "2", "foo": "even", "bar": "baz"}),
		},
		`pathID:found,body:valid`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`[
					{
						"op": "replace",
						"path": "/foo",
						"value": "baz"
					}
				]`))
				r, err := http.NewRequest("PATCH", "/foo/2", body)
				r.Header.Set("Content-Type", "application/json-patch+json")
				return r, err
			},
			ResponseCode: http.StatusOK,
			ResponseBody: `{"id": "2", "foo": "baz", "bar": "baz"}`,
			ExtraTest:    checkPayload("foo", "2", map[string]interface{}{"id": "2", "foo": "baz", "bar": "baz"}),
		},
		`pathID:found,body:valid,fields:invalid`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`[
					{
						"op": "replace",
						"path": "/foo",
						"value": "baz"
					}
				]`))
				r, err := http.NewRequest("PATCH", "/foo/2?fields=invalid", body)
				r.Header.Set("Content-Type", "application/json-patch+json")
				return r, err
			},
			ResponseCode: http.StatusUnprocessableEntity,
			ResponseBody: `{
				"code": 422,
				"message": "URL parameters contain error(s)",
				"issues": {
					"fields": ["invalid: unknown field"]
				}
			}`,
			ExtraTest: checkPayload("foo", "2", map[string]interface{}{"id": "2", "foo": "even", "bar": "baz"}),
		},
		`pathID:found,body:valid,fields:valid`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`[
					{
						"op": "replace",
						"path": "/foo",
						"value": "baz"
					}
				]`))
				r, err := http.NewRequest("PATCH", "/foo/2?fields=foo", body)
				r.Header.Set("Content-Type", "application/json-patch+json")
				return r, err
			},
			ResponseCode: http.StatusOK,
			ResponseBody: `{"foo": "baz"}`,
			ExtraTest:    checkPayload("foo", "2", map[string]interface{}{"id": "2", "foo": "baz", "bar": "baz"}),
		},
		`pathID:found,body:valid,header["If-Match"]:not-matching`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`[
					{
						"op": "replace",
						"path": "/foo",
						"value": "baz"
					}
				]`))
				r, err := http.NewRequest("PATCH", "/foo/2", body)
				if err != nil {
					return nil, err
				}
				r.Header.Set("If-Match", "W/x")
				r.Header.Set("Content-Type", "application/json-patch+json")
				return r, nil
			},
			ResponseCode: http.StatusPreconditionFailed,
			ResponseBody: `{"code": 412, "message": "Precondition Failed"}`,
			ExtraTest:    checkPayload("foo", "2", map[string]interface{}{"id": "2", "foo": "even", "bar": "baz"}),
		},
		`pathID:found,body:valid,header["If-Match"]:matching`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`[
					{
						"op": "replace",
						"path": "/foo",
						"value": "baz"
					}
				]`))
				r, err := http.NewRequest("PATCH", "/foo/2", body)
				if err != nil {
					return nil, err
				}
				r.Header.Set("If-Match", "W/b")
				r.Header.Set("Content-Type", "application/json-patch+json")
				return r, nil
			},
			ResponseCode: http.StatusOK,
			ResponseBody: `{"id": "2", "foo": "baz", "bar": "baz"}`,
			ExtraTest:    checkPayload("foo", "2", map[string]interface{}{"id": "2", "foo": "baz", "bar": "baz"}),
		},
		`pathID:found,body:valid,header["If-Unmodified-Since"]:invalid`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`[
					{
						"op": "replace",
						"path": "/foo",
						"value": "baz"
					}
				]`))
				r, err := http.NewRequest("PATCH", "/foo/1", body)
				if err != nil {
					return nil, err
				}
				r.Header.Set("If-Unmodified-Since", "invalid")
				r.Header.Set("Content-Type", "application/json-patch+json")
				return r, nil
			},
			ResponseCode: http.StatusBadRequest,
			ResponseBody: `{"code": 400, "message": "Invalid If-Unmodified-Since header"}`,
			ExtraTest:    checkPayload("foo", "1", map[string]interface{}{"id": "1", "foo": "odd", "bar": "baz"}),
		},
		`pathID:found,body:valid,header["If-Unmodified-Since"]:not-matching`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`[
					{
						"op": "replace",
						"path": "/foo",
						"value": "baz"
					}
				]`))
				r, err := http.NewRequest("PATCH", "/foo/1", body)
				if err != nil {
					return nil, err
				}
				r.Header.Set("If-Unmodified-Since", yesterday.Format(time.RFC1123))
				r.Header.Set("Content-Type", "application/json-patch+json")
				return r, nil
			},
			ResponseCode: http.StatusPreconditionFailed,
			ResponseBody: `{"code": 412, "message": "Precondition Failed"}`,
			ExtraTest:    checkPayload("foo", "1", map[string]interface{}{"id": "1", "foo": "odd", "bar": "baz"}),
		},
		`parentPathID:found,pathID:found,body:alter-parent-id`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`[
					{
						"op": "replace",
						"path": "/foo",
						"value": "2"
					}
				]`))
				r, err := http.NewRequest("PATCH", "/foo/3/sub/1", body)
				if err != nil {
					return nil, err
				}
				r.Header.Set("Content-Type", "application/json-patch+json")
				return r, nil
			},
			ResponseCode: http.StatusOK,
			ResponseBody: `{"id": "1", "foo": "2"}`,
			ExtraTest:    checkPayload("foo.sub", "1", map[string]interface{}{"id": "1", "foo": "2"}),
		},
		`pathID:found,body:add-field`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`[
					{
						"op": "add",
						"path": "/bar",
						"value": "value"
					}
				]`))
				r, err := http.NewRequest("PATCH", "/foo/4", body)
				r.Header.Set("Content-Type", "application/json-patch+json")
				return r, err
			},
			ResponseCode: http.StatusOK,
			ResponseBody: `{"id": "4", "foo": "odd", "bar": "value"}`,
			ExtraTest:    checkPayload("foo", "4", map[string]interface{}{"id": "4", "foo": "odd", "bar": "value"}),
		},
		`pathID:found,body:remove-field`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`[
					{
						"op": "remove",
						"path": "/bar"
					}
				]`))
				r, err := http.NewRequest("PATCH", "/foo/3", body)
				r.Header.Set("Content-Type", "application/json-patch+json")
				return r, err
			},
			ResponseCode: http.StatusOK,
			ResponseBody: `{"id": "3", "foo": "odd"}`,
			ExtraTest:    checkPayload("foo", "3", map[string]interface{}{"id": "3", "foo": "odd"}),
		},
		`pathID:found,body:valid,object:invalid`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`[
					{
						"op": "add",
						"path": "/oar/x",
						"value": "baz"
					}
				]`))
				r, err := http.NewRequest("PATCH", "/foo/5", body)
				r.Header.Set("Content-Type", "application/json-patch+json")
				return r, err
			},
			ResponseCode: http.StatusUnprocessableEntity,
			ResponseBody: `{
				"code":422,
				"message": "Document contains error(s)",
				"issues": {
					"oar": ["x is [invalid field]"]
				}
			}`,
		},
		`pathID:found,body:valid,object:valid`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`[
					{
						"op": "add",
						"path": "/oar/b",
						"value": "baz"
					}
				]`))
				r, err := http.NewRequest("PATCH", "/foo/5", body)
				r.Header.Set("Content-Type", "application/json-patch+json")
				return r, err
			},
			ResponseCode: http.StatusOK,
			ResponseBody: `{"id": "5", "foo": "odd", "oar": {"a": "original", "b": "baz"}}`,
			ExtraTest:    checkPayload("foo", "5", map[string]interface{}{"id": "5", "foo": "odd", "oar": map[string]interface{}{"a": "original", "b": "baz"}}),
		},
		`pathID:found,body:valid,array:valid`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`[
					{
						"op": "add",
						"path": "/aar/0",
						"value": "baz"
					}
				]`))
				r, err := http.NewRequest("PATCH", "/foo/6", body)
				r.Header.Set("Content-Type", "application/json-patch+json")
				return r, err
			},
			ResponseCode:   http.StatusOK,
			ResponseBody:   `{"id": "6", "foo": "odd", "aar": ["baz", "value-1"]}`,
			ResponseHeader: http.Header{"Etag": []string{`W/"ad278e57a1abd1794df1ce05519639b2"`}},
			ExtraTest:      checkPayload("foo", "6", map[string]interface{}{"id": "6", "foo": "odd", "aar": []interface{}{"baz", "value-1"}}),
		},
		`pathID:found,body:valid:minimal`: {
			Init: sharedInit,
			NewRequest: func() (*http.Request, error) {
				body := bytes.NewReader([]byte(`[
					{
						"op": "replace",
						"path": "/foo",
						"value": "baz"
					}
				]`))
				r, err := http.NewRequest("PATCH", "/foo/2", body)
				r.Header.Set("Content-Type", "application/json-patch+json")
				r.Header.Set("Prefer", "return=minimal")
				return r, err
			},
			ResponseCode:   http.StatusNoContent,
			ResponseBody:   ``,
			ResponseHeader: http.Header{"Etag": []string{`W/"53c7f8b8a84dd407e1491f5339fca757"`}},
			ExtraTest:      checkPayload("foo", "2", map[string]interface{}{"id": "2", "foo": "baz", "bar": "baz"}),
		},
	}

	for n, tc := range tests {
		tc := tc // capture range variable
		t.Run(n, tc.Test)
	}
}
