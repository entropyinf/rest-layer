package schema_test

import (
	"testing"

	"github.com/entropyinf/rest-layer/schema"
)

func TestReferenceValidate(t *testing.T) {
	cases := []fieldValidatorTestCase{
		{
			Name:      `{Path:valid}.Validate(valid)`,
			Validator: &schema.Reference{Path: "foobar"},
			ReferenceChecker: fakeReferenceChecker{
				"foobar": {IDs: []interface{}{"a", "b"}, Validator: &schema.String{}, SchemaValidator: &schema.Schema{}},
			},
			Input:  "a",
			Expect: "a",
		},
		{
			Name:      `{Path:valid}.Validate(invalid)`,
			Validator: &schema.Reference{Path: "foobar"},
			ReferenceChecker: fakeReferenceChecker{
				"foobar": {IDs: []interface{}{"a", "b"}, Validator: &schema.String{}, SchemaValidator: &schema.Schema{}},
			},
			Input: "c",
			Error: "not found",
		},
	}
	for i := range cases {
		cases[i].Run(t)
	}
}
