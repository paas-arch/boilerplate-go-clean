package main

import "testing"

func TestGoTypeComponents_AdditionalProperties(t *testing.T) {
	schema := &Schema{
		Type: "object",
		AdditionalProperties: &Schema{
			Type: "integer",
		},
	}

	if got, want := goTypeComponents(schema), "map[string]int64"; got != want {
		t.Fatalf("goTypeComponents() = %q, want %q", got, want)
	}
}

func TestBuildCompSchema_RefFieldType(t *testing.T) {
	tests := []struct {
		name     string
		required []string
		want     string
	}{
		{
			name: "необязательный объект — указатель",
			want: "*FoodDetails",
		},
		{
			name:     "обязательный объект — значение",
			required: []string{"food"},
			want:     "FoodDetails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Schema{
				Required:   tt.required,
				Properties: map[string]*Schema{"food": {Ref: "#/components/schemas/FoodDetails"}},
			}

			if got := buildCompSchema("PlaceCategory", s).Fields[0].GoType; got != tt.want {
				t.Fatalf("GoType = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOptionalBoolean(t *testing.T) {
	for _, build := range []func(string, *Schema) CompSchema{buildCompSchema, buildInlineSchema} {
		s := &Schema{Properties: map[string]*Schema{"halal": {Type: "boolean"}}}
		if got := build("Details", s).Fields[0]; got.GoType != "*bool" || !got.OmitEmpty {
			t.Fatalf("optional boolean must preserve false: %+v", got)
		}
		s.Required = []string{"halal"}
		if got := build("Details", s).Fields[0]; got.GoType != "bool" || got.OmitEmpty {
			t.Fatalf("required boolean: %+v", got)
		}
	}
}
