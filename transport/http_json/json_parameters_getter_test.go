package http_json

import (
	"encoding/json"
	"testing"

	"github.com/sergei-svistunov/gorpc"
	"github.com/sergei-svistunov/gorpc/test/handler1"

	"golang.org/x/net/context"
)

func TestJsonParametersGetter(t *testing.T) {
	p := &JsonParametersGetter{
		values: map[string]interface{}{
			"int": json.Number("3"),
			"nested": map[string]interface{}{
				"b": true,
			},
			"slice_in_slice": []interface{}{
				[]interface{}{json.Number("1"), json.Number("2"), json.Number("3")},
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
	if v, err := p.GetInt([]string{"slice_in_slice", "0"}, "1"); err != nil || v != 2 {
		t.Fatalf("'slice_in_slice[0][1]'(%d) != 2, error: %v", v, err)
	}
}

func TestHandlerManager_PrepareParameters_SliceInSlice(t *testing.T) {
	hm := gorpc.NewHandlersManager("github.com/sergei-svistunov/gorpc", gorpc.HandlersManagerCallbacks{})
	hm.RegisterHandler(handler1.NewHandler())

	pg := &JsonParametersGetter{
		values: map[string]interface{}{
			"slice_in_slice": []interface{}{
				[]interface{}{
					map[string]interface{}{
						"f1": json.Number("1"),
					},
					map[string]interface{}{
						"f1": json.Number("2"),
						"f2": json.Number("20"),
					},
					map[string]interface{}{
						"f1": json.Number("3"),
						"f2": json.Number("30"),
					},
				},
			},
		},
	}

	hanlderVersion := hm.FindHandler("/test/handler1", 4)
	if hanlderVersion == nil {
		t.Fatal("Handler wasn't found")
	}

	v, err := hm.UnmarshalParameters(context.TODO(), hanlderVersion, pg)
	if err != nil {
		t.Fatal(err)
	}

	if v.Interface().(*handler1.V4Request).SliceInSlice[0][1].F1 != 2 {
		t.Fatalf("Error in parsing slice in slice")
	}
}
