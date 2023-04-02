package openapi

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/jinzhu/inflection"
	"github.com/rs/rest-layer/resource"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func addResource(doc *openapi3.T, prevRscList []*resource.Resource, rsc *resource.Resource) {

	schemaNamePlural := rsc.Name()
	schemaNameSingular := inflection.Singular(rsc.Name())
	schemaIdParameter := schemaNameSingular + "Id"
	schemaNameSingularSource := schemaNameSingular + "Source"

	doc.Components.Schemas[schemaNameSingular] = &openapi3.SchemaRef{
		Value: generateSchema(rsc.Schema(), false),
	}

	doc.Components.Schemas[schemaNameSingularSource] = &openapi3.SchemaRef{
		Value: generateSchema(rsc.Schema(), true),
	}

	doc.Components.Parameters[schemaIdParameter] = &openapi3.ParameterRef{
		Value: &openapi3.Parameter{
			Name:        schemaIdParameter,
			Description: fmt.Sprintf("The %s's ID", schemaNameSingular),
			In:          "path",
			Required:    true,
			Schema: &openapi3.SchemaRef{
				Ref: fmt.Sprintf("#/components/schemas/%s/properties/id", schemaNameSingular),
			},
		},
	}

	var path string
	var operationSufix string
	var params []*openapi3.ParameterRef
	for _, prevRsc := range prevRscList {
		prevSchemaNamePlural := prevRsc.Name()
		prevSchemaNameSingular := inflection.Singular(prevRsc.Name())
		prevSchemaIdParameter := prevSchemaNameSingular + "Id"

		path = path + fmt.Sprintf("/%s/{%s}", prevSchemaNamePlural, prevSchemaIdParameter)
		operationSufix = operationSufix + fmt.Sprintf("On%s", cases.Title(language.English).String(prevSchemaNameSingular))

		param := &openapi3.ParameterRef{
			Ref: fmt.Sprintf("#/components/parameters/%s", prevSchemaIdParameter),
		}

		params = append(params, param)
	}

	path = path + fmt.Sprintf("/%s", schemaNamePlural)
	resourceName := cases.Title(language.English).String(schemaNamePlural) + operationSufix
	var topResource *resource.Resource
	if prevRscList != nil && len(prevRscList) > 0 {
		topResource = prevRscList[0]
	} else {
		topResource = rsc
	}
	tagName := cases.Title(language.English).String(topResource.Name())

	if tag := doc.Tags.Get(tagName); tag == nil {
		doc.Tags = append(doc.Tags, &openapi3.Tag{
			Name:        tagName,
			Description: rsc.Schema().Description,
		})
	}

	if rsc.Conf().IsModeAllowed(resource.List) {
		op := &openapi3.Operation{
			Tags:        []string{tagName},
			Summary:     "List" + resourceName,
			OperationID: "List" + resourceName,
			Parameters: append(
				[]*openapi3.ParameterRef{
					{Ref: "#/components/parameters/filter"},
					{Ref: "#/components/parameters/fields"},
					{Ref: "#/components/parameters/limit"},
					{Ref: "#/components/parameters/page"},
					{Ref: "#/components/parameters/skip"},
					{Ref: "#/components/parameters/total"},
					{Ref: "#/components/parameters/sort"},
				},
				params...,
			),
			Responses: map[string]*openapi3.ResponseRef{
				"200": {
					Value: &openapi3.Response{
						Description: StringPtr(fmt.Sprintf("List of %s", schemaNamePlural)),
						Headers: map[string]*openapi3.HeaderRef{
							"Date":    {Ref: "#/components/headers/Date"}, // TODO: Verify
							"X-Total": {Ref: "#/components/headers/X-Total"},
						},
						Content: map[string]*openapi3.MediaType{
							"application/json": {
								Schema: &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Type: "array",
										Items: &openapi3.SchemaRef{
											Ref: fmt.Sprintf("#/components/schemas/%s", schemaNameSingular),
										},
									},
								},
							},
						},
					},
				},
				"default": {
					Ref: "#/components/responses/Error",
				},
			},
		}
		doc.AddOperation(path, "GET", op)
	}

	if rsc.Conf().IsModeAllowed(resource.Create) {
		op := &openapi3.Operation{
			Tags:        []string{tagName},
			Summary:     "Create" + cases.Title(language.English).String(schemaNameSingular) + operationSufix,
			OperationID: "Create" + cases.Title(language.English).String(schemaNameSingular) + operationSufix,
			Parameters: append(
				[]*openapi3.ParameterRef{},
				params...,
			),
			RequestBody: &openapi3.RequestBodyRef{
				Value: &openapi3.RequestBody{
					Content: map[string]*openapi3.MediaType{
						"application/json": {
							Schema: &openapi3.SchemaRef{
								Ref: fmt.Sprintf("#/components/schemas/%s", schemaNameSingularSource),
							},
						},
					},
				},
			},
			Responses: map[string]*openapi3.ResponseRef{
				"201": {
					Value: &openapi3.Response{
						Description: StringPtr(fmt.Sprintf("Created %s", schemaNameSingular)),
						Headers: map[string]*openapi3.HeaderRef{
							"Etag":          {Ref: "#/components/headers/Etag"},
							"Last-Modified": {Ref: "#/components/headers/Last-Modified"},
						},
						Content: map[string]*openapi3.MediaType{
							"application/json": {
								Schema: &openapi3.SchemaRef{
									Ref: fmt.Sprintf("#/components/schemas/%s", schemaNameSingular),
								},
							},
						},
					},
				},
				"422": {
					Ref: "#/components/responses/ValidationError",
				},
				"default": {
					Ref: "#/components/responses/Error",
				},
			},
		}
		doc.AddOperation(path, "POST", op)
	}

	if rsc.Conf().IsModeAllowed(resource.Clear) {
		op := &openapi3.Operation{
			Tags:        []string{tagName},
			Summary:     "Clear" + cases.Title(language.English).String(rsc.Name()) + operationSufix,
			OperationID: "Clear" + cases.Title(language.English).String(rsc.Name()) + operationSufix,
			Parameters: append(
				[]*openapi3.ParameterRef{
					{Ref: "#/components/parameters/filter"},
				},
				params...,
			),
			Responses: map[string]*openapi3.ResponseRef{
				"204": {
					Value: &openapi3.Response{
						Description: StringPtr(fmt.Sprintf("Clear %s", rsc.Name())),
						Headers: map[string]*openapi3.HeaderRef{
							"Date":    {Ref: "#/components/headers/Date"}, // TODO: Verify
							"X-Total": {Ref: "#/components/headers/X-Total"},
						},
					},
				},
				"default": {
					Ref: "#/components/responses/Error",
				},
			},
		}
		doc.AddOperation(path, "DELETE", op)
	}

	path = path + fmt.Sprintf("/{%s}", schemaIdParameter)

	if rsc.Conf().IsModeAllowed(resource.Read) {
		op := &openapi3.Operation{
			Tags:        []string{tagName},
			Summary:     "Read" + cases.Title(language.English).String(schemaNameSingular) + operationSufix,
			OperationID: "Read" + cases.Title(language.English).String(schemaNameSingular) + operationSufix,
			Parameters: append(
				[]*openapi3.ParameterRef{
					{Ref: "#/components/parameters/fields"},
					{Ref: fmt.Sprintf("#/components/parameters/%s", schemaIdParameter)},
				},
				params...,
			),
			Responses: map[string]*openapi3.ResponseRef{
				"200": {
					Value: &openapi3.Response{
						Description: StringPtr(fmt.Sprintf("Read %s", rsc.Name())),
						Headers: map[string]*openapi3.HeaderRef{
							"Date":    {Ref: "#/components/headers/Date"},
							"X-Total": {Ref: "#/components/headers/X-Total"},
						},
						Content: map[string]*openapi3.MediaType{
							"application/json": {
								Schema: &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Type: "array",
										Items: &openapi3.SchemaRef{
											Ref: fmt.Sprintf("#/components/schemas/%s", schemaNameSingular),
										},
									},
								},
							},
						},
					},
				},
				"default": {
					Ref: "#/components/responses/Error",
				},
			},
		}
		doc.AddOperation(path, "GET", op)
	}

	if rsc.Conf().IsModeAllowed(resource.Replace) {
		op := &openapi3.Operation{
			Tags:        []string{tagName},
			Summary:     "Replace" + cases.Title(language.English).String(schemaNameSingular) + operationSufix,
			OperationID: "Replace" + cases.Title(language.English).String(schemaNameSingular) + operationSufix,
			Parameters: append(
				[]*openapi3.ParameterRef{
					{Ref: fmt.Sprintf("#/components/parameters/%s", schemaIdParameter)},
					{Value: &openapi3.Parameter{
						Name: "If-Match",
						In:   "header",
						Schema: &openapi3.SchemaRef{
							Ref: "#/components/headers/If-Match",
						},
					}},
				},
				params...,
			),
			RequestBody: &openapi3.RequestBodyRef{
				Value: &openapi3.RequestBody{
					Content: map[string]*openapi3.MediaType{
						"application/json": {
							Schema: &openapi3.SchemaRef{
								Ref: fmt.Sprintf("#/components/schemas/%s", schemaNameSingularSource),
							},
						},
					},
				},
			},
			Responses: map[string]*openapi3.ResponseRef{
				"200": {
					Value: &openapi3.Response{
						Description: StringPtr(fmt.Sprintf("Replace %s", rsc.Name())),
						Headers: map[string]*openapi3.HeaderRef{
							"Etag":          {Ref: "#/components/headers/Etag"},
							"Last-Modified": {Ref: "#/components/headers/Last-Modified"},
						},
						Content: map[string]*openapi3.MediaType{
							"application/json": {
								Schema: &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Type: "array",
										Items: &openapi3.SchemaRef{
											Ref: fmt.Sprintf("#/components/schemas/%s", schemaNameSingular),
										},
									},
								},
							},
						},
					},
				},
				"422": {
					Ref: "#/components/responses/ValidationError",
				},
				"default": {
					Ref: "#/components/responses/Error",
				},
			},
		}
		doc.AddOperation(path, "PUT", op)
	}

	if rsc.Conf().IsModeAllowed(resource.Update) {
		op := &openapi3.Operation{
			Tags:        []string{tagName},
			Summary:     "Update" + cases.Title(language.English).String(schemaNameSingular) + operationSufix,
			OperationID: "Update" + cases.Title(language.English).String(schemaNameSingular) + operationSufix,
			Parameters: append(
				[]*openapi3.ParameterRef{
					{Ref: fmt.Sprintf("#/components/parameters/%s", schemaIdParameter)},
					{Value: &openapi3.Parameter{
						Name: "If-Match",
						In:   "header",
						Schema: &openapi3.SchemaRef{
							Ref: "#/components/headers/If-Match",
						},
					}},
				},
				params...,
			),
			RequestBody: &openapi3.RequestBodyRef{
				Value: &openapi3.RequestBody{
					Content: map[string]*openapi3.MediaType{
						"application/json-patch+json": {
							Schema: &openapi3.SchemaRef{
								Ref: "#/components/schemas/JSONPatch",
							},
						},
						"application/json": {
							Schema: &openapi3.SchemaRef{
								Ref: fmt.Sprintf("#/components/schemas/%s", schemaNameSingularSource),
							},
						},
					},
				},
			},
			Responses: map[string]*openapi3.ResponseRef{
				"200": {
					Value: &openapi3.Response{
						Description: StringPtr(fmt.Sprintf("Update %s", rsc.Name())),
						Headers: map[string]*openapi3.HeaderRef{
							"Etag":          {Ref: "#/components/headers/Etag"},
							"Last-Modified": {Ref: "#/components/headers/Last-Modified"},
						},
						Content: map[string]*openapi3.MediaType{
							"application/json": {
								Schema: &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Type: "array",
										Items: &openapi3.SchemaRef{
											Ref: fmt.Sprintf("#/components/schemas/%s", schemaNameSingular),
										},
									},
								},
							},
						},
					},
				},
				"422": {
					Ref: "#/components/responses/ValidationError",
				},
				"default": {
					Ref: "#/components/responses/Error",
				},
			},
		}
		doc.AddOperation(path, "PATCH", op)
	}

	if rsc.Conf().IsModeAllowed(resource.Delete) {
		op := &openapi3.Operation{
			Tags:        []string{tagName},
			Summary:     "Delete" + cases.Title(language.English).String(schemaNameSingular) + operationSufix,
			OperationID: "Delete" + cases.Title(language.English).String(schemaNameSingular) + operationSufix,
			Parameters: append(
				[]*openapi3.ParameterRef{
					{Ref: fmt.Sprintf("#/components/parameters/%s", schemaIdParameter)},
					{Value: &openapi3.Parameter{
						Name: "If-Match",
						In:   "header",
						Schema: &openapi3.SchemaRef{
							Ref: "#/components/headers/If-Match",
						},
					}},
				},
				params...,
			),
			Responses: map[string]*openapi3.ResponseRef{
				"204": {
					Value: &openapi3.Response{
						Description: StringPtr(fmt.Sprintf("Delete %s", rsc.Name())),
					},
				},
				"422": {
					Ref: "#/components/responses/ValidationError",
				},
				"default": {
					Ref: "#/components/responses/Error",
				},
			},
		}
		doc.AddOperation(path, "DELETE", op)
	}

	for _, subRsc := range rsc.GetResources() {
		addResource(doc, append(prevRscList, rsc), subRsc)
	}
}
