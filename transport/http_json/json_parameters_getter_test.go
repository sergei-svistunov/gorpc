package http_json

import (
	"encoding/json"
	"testing"
)

func TestJsonParametersGetter(t *testing.T) {
	p := &JsonParametersGetter{
		values: map[string]interface{}{
			"int": json.Number("3"),
			"nested": map[string]interface{}{
				"b": true,
			},
		},
	}
	if !p.IsExists(nil, "int") {
		t.Fatal("'int' not found")
	}
	if v, err := p.GetInt(nil, "int"); err != nil || v != 3 {
		t.Fatalf("'int'(%d) != 3, error: %v", v, err)
	}
	if !p.IsExists([]string{"nested"}, "b") {
		t.Fatal("'nested.b' not found")
	}
	if v, err := p.GetBool([]string{"nested"}, "b"); err != nil || v != true {
		t.Fatalf("'nested.b'(%z) != true, error: %v", v, err)
	}
}
