package openapi

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/entropyinf/rest-layer/schema"
	"log"
	"reflect"
)

func generateSchema(s schema.Schema, hideReadOnly bool) *openapi3.Schema {
	ret := &openapi3.Schema{
		Type:        "object",
		Description: s.Description,
		Properties:  map[string]*openapi3.SchemaRef{},
	}

	for fieldName, field := range s.Fields {
		if !(hideReadOnly && field.ReadOnly) {
			ret.Properties[fieldName] = &openapi3.SchemaRef{}
			ret.Properties[fieldName].Value = generateSchemaFromField(field, hideReadOnly)
		}
	}

	return ret
}

func generateSchemaFromField(field schema.Field, hideReadOnly bool) *openapi3.Schema {
	if field.Validator == nil {
		log.Fatalln("validator required")
	}

	switch t := field.Validator.(type) {
	case *schema.String:
		return generateSchemaFromFieldString(field)
	case *schema.Array:
		return generateSchemaFromFieldArray(field, hideReadOnly)
	case *schema.Reference:
		return generateSchemaFromFieldReference(field)
	case *schema.Time:
		return generateSchemaFromFieldTime(field)
	case *schema.Bool:
		return generateSchemaFromFieldBool(field)
	case *schema.Null:
		return generateSchemaFromFieldNil(field)
	case *schema.Integer:
		return generateSchemaFromFieldInteger(field)
	case *schema.Dict:
		return generateSchemaFieldDict(field)
	case *schema.Password:
		return generateSchemaFromFieldPassword(field)
	case *schema.URL:
		return generateSchemaFromFieldURL(field)
	case *schema.Float:
		return generateSchemaFromFieldFloat(field)

	case *schema.Object:
		if t.Schema != nil {
			return generateSchema(*t.Schema, hideReadOnly)
		}
		log.Fatalln("validator required.")
	default:
		log.Fatalln("unsupported Type:", reflect.TypeOf(t))
	}
	return nil
}

func generateSchemaFromFieldFloat(f schema.Field) *openapi3.Schema {
	return &openapi3.Schema{
		Type:        "number",
		Description: f.Description,
		ReadOnly:    f.ReadOnly,
		Default:     f.Default,
	}
}

func generateSchemaFieldDict(f schema.Field) *openapi3.Schema {
	return &openapi3.Schema{
		Type:        "object",
		Description: f.Description,
		ReadOnly:    f.ReadOnly,
		Default:     f.Default,
	}
}

func generateSchemaFromFieldInteger(f schema.Field) *openapi3.Schema {
	return &openapi3.Schema{
		Type:        "integer",
		Description: f.Description,
		ReadOnly:    f.ReadOnly,
		Default:     f.Default,
		Example:     f.Default,
	}
}

func generateSchemaFromFieldNil(f schema.Field) *openapi3.Schema {
	ret := &openapi3.Schema{
		Type:        "string",
		Description: f.Description,
		ReadOnly:    f.ReadOnly,
		Default:     f.Default,
		Example:     f.Default,
	}

	return ret
}

func generateSchemaFromFieldBool(f schema.Field) *openapi3.Schema {
	ret := &openapi3.Schema{
		Type:        "boolean",
		Description: f.Description,
		ReadOnly:    f.ReadOnly,
		Default:     f.Default,
	}

	return ret

}

func generateSchemaFromFieldTime(f schema.Field) *openapi3.Schema {
	ret := &openapi3.Schema{
		Type:        "string",
		Format:      "date-time",
		Description: f.Description,
		ReadOnly:    f.ReadOnly,
		Default:     f.Default,
	}

	return ret

}

func generateSchemaFromFieldString(f schema.Field) *openapi3.Schema {
	v := f.Validator.(*schema.String)
	ret := &openapi3.Schema{
		Type:        "string",
		Description: f.Description,
		MinLength:   uint64(v.MinLen),
		Pattern:     v.Regexp,
		Example:     f.Default,
		Enum:        []any{},
	}
	if v.MaxLen > 0 {
		ret.MaxLength = openapi3.Uint64Ptr(uint64(v.MaxLen))
	}

	allowed := v.Allowed
	if allowed != nil && len(allowed) > 0 {
		for _, a := range allowed {
			ret.Enum = append(ret.Enum, a)
		}
	}

	return ret
}

func generateSchemaFromFieldPassword(f schema.Field) *openapi3.Schema {
	v := f.Validator.(*schema.Password)
	ret := &openapi3.Schema{
		Type:      "string",
		MinLength: uint64(v.MinLen),
	}
	if v.MaxLen > 0 {
		ret.MaxLength = openapi3.Uint64Ptr(uint64(v.MaxLen))
	}

	return ret
}

func generateSchemaFromFieldURL(f schema.Field) *openapi3.Schema {
	ret := &openapi3.Schema{
		Type:        "string",
		Description: f.Description,
	}

	return ret
}

func generateSchemaFromFieldArray(f schema.Field, hideReadOnly bool) *openapi3.Schema {
	v := f.Validator.(*schema.Array)
	ret := &openapi3.Schema{
		Type:     "array",
		MinItems: uint64(v.MinLen),
		Items: &openapi3.SchemaRef{
			Value: generateSchemaFromField(v.Values, hideReadOnly),
		},
	}
	if v.MaxLen > 0 {
		ret.MaxItems = openapi3.Uint64Ptr(uint64(v.MaxLen))
	}

	return ret
}

func generateSchemaFromFieldReference(f schema.Field) *openapi3.Schema {
	ret := &openapi3.Schema{
		Description: "Reference id",
		Type:        "string",
		ReadOnly:    false,
		Format:      "^[0-9a-v]{20}$|^$",
	}

	return ret
}
