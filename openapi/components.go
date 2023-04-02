package openapi

import (
	"github.com/getkin/kin-openapi/openapi3"
)

var staticComponents = &openapi3.Components{
	Parameters: map[string]*openapi3.ParameterRef{
		"sort": {
			Value: &openapi3.Parameter{
				Description: "[Sort](http://rest-layer.io/#sorting) Sorting of resource items is defined through the sort query-string parameter. The sort value is a list of resource’s fields separated by comas (,)",
				Name:        "sort",
				In:          "query",
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "string",
					},
				},
			},
		},
		"filter": {
			Value: &openapi3.Parameter{
				Description: "[Filter](http://rest-layer.io/#filtering) which entries to show. Allows a MongoDB-like query syntax.",
				Name:        "filter",
				In:          "query",
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "string",
					},
				},
			},
		},
		"fields": {
			Value: &openapi3.Parameter{
				Description: "[Select](http://rest-layer.io/#field-selection) which fields to show, including [embedding](http://rest-layer.io/#embedding) of related resources.",
				Name:        "fields",
				In:          "query",
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "string",
					},
				},
			},
		},
		"limit": {
			Value: &openapi3.Parameter{
				Description: "Limit maximum entries per [page](http://rest-layer.io/#paginatio).",
				Name:        "limit",
				In:          "query",
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "integer",
						Min:  openapi3.Float64Ptr(0),
					},
				},
			},
		},
		"skip": {
			Value: &openapi3.Parameter{
				Description: "[Skip](http://rest-layer.io/#skipping) the first N entries.",
				Name:        "skip",
				In:          "query",
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "integer",
						Min:  openapi3.Float64Ptr(0),
					},
				},
			},
		},
		"page": {
			Value: &openapi3.Parameter{
				Description: "The [page](http://rest-layer.io/#pagination) number to display, starting at 1.",
				Name:        "page",
				In:          "query",
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:    "integer",
						Default: openapi3.Float64Ptr(1),
						Min:     openapi3.Float64Ptr(1),
					},
				},
			},
		},
		"total": {
			Value: &openapi3.Parameter{
				Description: "Force total number of entries to be included in the response header. This could have performance implications.Use total = 1 to enable.",
				Name:        "total",
				In:          "query",
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:    "integer",
						Default: 0,
						Enum:    []any{0, 1},
					},
				},
			},
		},
	},
	Headers: map[string]*openapi3.HeaderRef{
		"If-Match": {
			Value: &openapi3.Header{
				Parameter: openapi3.Parameter{
					Description: "See also: [If-Match](https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Headers/If-Match).",
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: "string",
						},
					},
				},
			},
		},
		"Date": {
			Value: &openapi3.Header{
				Parameter: openapi3.Parameter{
					Description: "The time this request was served.",
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type:   "string",
							Format: "date-time",
						},
					},
				},
			},
		},
		"Etag": {
			Value: &openapi3.Header{
				Parameter: openapi3.Parameter{
					Description: "Provides [concurrency-control](https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Headers/ETag) down to the storage layer.",
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: "string",
						},
					},
				},
			},
		},
		"Last-Modified": {
			Value: &openapi3.Header{
				Parameter: openapi3.Parameter{
					Description: "When this resource was last modified.",
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type:   "string",
							Format: "date-time",
						},
					},
				},
			},
		},
		"X-Total": {
			Value: &openapi3.Header{
				Parameter: openapi3.Parameter{
					Description: "Total number of entries matching the supplied filter.",
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: "integer",
						},
					},
				},
			},
		},
	},
	Schemas: map[string]*openapi3.SchemaRef{
		"Error": {
			Value: &openapi3.Schema{
				Type:     "object",
				Required: []string{"code", "message"},
				Properties: map[string]*openapi3.SchemaRef{
					"code": {
						Value: &openapi3.Schema{
							Description: "HTTP Status code",
							Type:        "integer",
						},
					},
					"message": {
						Value: &openapi3.Schema{
							Description: "Error message",
							Type:        "string",
						},
					},
				},
			},
		},
		"JSONPatch": {
			Value: &openapi3.Schema{
				Type: "array",
				Items: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:     "object",
						Required: []string{"op", "path"},
						Properties: map[string]*openapi3.SchemaRef{
							"op": {
								Value: &openapi3.Schema{
									Description: "operation",
									Type:        "string",
									Enum:        []any{"test", "remove", "add", "replace", "move", "copy"},
								},
							},
							"path": {
								Value: &openapi3.Schema{
									Description: "operation",
									Type:        "string",
									Example:     "/foo/bar",
								},
							},
							"value": {
								Value: &openapi3.Schema{
									Description: "operation",
									Type:        "string",
									Example:     "hello",
								},
							},
						},
					},
				},
			},
		},
		"ValidationError": {
			Value: &openapi3.Schema{
				Type:     "object",
				Required: []string{"code", "message"},
				Properties: map[string]*openapi3.SchemaRef{
					"code": {
						Value: &openapi3.Schema{
							Description: "HTTP Status code",
							Type:        "integer",
						},
					},
					"message": {
						Value: &openapi3.Schema{
							Description: "Error message",
							Type:        "string",
						},
					},
					"issues": {
						Value: &openapi3.Schema{
							Description: "Error details",
							Type:        "object",
							Properties: openapi3.Schemas{
								"fields": &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Type: "array",
										Items: &openapi3.SchemaRef{
											Value: &openapi3.Schema{
												Type: "string",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	},
	Responses: map[string]*openapi3.ResponseRef{
		"Error": {
			Value: &openapi3.Response{
				Description: StringPtr("Error"),
				Content: map[string]*openapi3.MediaType{
					"application/json": {
						Schema: &openapi3.SchemaRef{
							Ref: "#/components/schemas/Error",
						},
					},
				},
			},
		},
		"ValidationError": {
			Value: &openapi3.Response{
				Description: StringPtr("Validation Error"),
				Content: map[string]*openapi3.MediaType{
					"application/json": {
						Schema: &openapi3.SchemaRef{
							Ref: "#/components/schemas/ValidationError",
						},
					},
				},
			},
		},
	},
}
