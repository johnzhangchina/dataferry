package engine

import (
	"testing"

	"github.com/nianhe/nianhe/internal/model"
)

func TestTransform_DirectMapping(t *testing.T) {
	source := map[string]any{
		"data": map[string]any{
			"user_name": "zhangsan",
			"email":     "zhangsan@example.com",
		},
	}
	mappings := []model.Mapping{
		{Source: "data.user_name", Target: "username", Transform: "direct"},
		{Source: "data.email", Target: "contact.email"},
	}

	result, err := Transform(source, mappings)
	if err != nil {
		t.Fatal(err)
	}

	if result["username"] != "zhangsan" {
		t.Errorf("expected username=zhangsan, got %v", result["username"])
	}
	contact, ok := result["contact"].(map[string]any)
	if !ok {
		t.Fatal("expected contact to be a map")
	}
	if contact["email"] != "zhangsan@example.com" {
		t.Errorf("expected email=zhangsan@example.com, got %v", contact["email"])
	}
}

func TestTransform_Constant(t *testing.T) {
	source := map[string]any{"name": "test"}
	mappings := []model.Mapping{
		{Target: "source", Transform: "constant", Value: "子公司A"},
		{Source: "name", Target: "name"},
	}

	result, err := Transform(source, mappings)
	if err != nil {
		t.Fatal(err)
	}

	if result["source"] != "子公司A" {
		t.Errorf("expected source=子公司A, got %v", result["source"])
	}
	if result["name"] != "test" {
		t.Errorf("expected name=test, got %v", result["name"])
	}
}

func TestTransform_MissingSourceSkipped(t *testing.T) {
	source := map[string]any{"a": 1}
	mappings := []model.Mapping{
		{Source: "b", Target: "out", Transform: "direct"},
	}

	result, err := Transform(source, mappings)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := result["out"]; ok {
		t.Error("expected missing source to be skipped")
	}
}

func TestTransform_EmptySourceErrors(t *testing.T) {
	source := map[string]any{"a": 1}
	mappings := []model.Mapping{
		{Source: "", Target: "out", Transform: "direct"},
	}

	_, err := Transform(source, mappings)
	if err == nil {
		t.Error("expected error for empty source in direct mapping")
	}
}
