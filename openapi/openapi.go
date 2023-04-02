package openapi

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/entropyinf/rest-layer/resource"
)

func FillOpenapiFromIndex(index resource.Index, doc *openapi3.T) {
	doc.OpenAPI = "3.0.3"
	doc.Components = staticComponents
	for _, rsc := range index.GetResources() {
		addResource(doc, []*resource.Resource{}, rsc)
	}
}
