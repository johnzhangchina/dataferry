package engine

import (
	"testing"

	"github.com/johnzhangchina/dataferry/internal/model"
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

func TestTransform_Template(t *testing.T) {
	source := map[string]any{
		"first": "张",
		"last":  "三",
		"id":    float64(42),
	}
	mappings := []model.Mapping{
		{Target: "fullname", Transform: "template", Value: "{{first}} {{last}}"},
		{Target: "ref", Transform: "template", Value: "ID-{{id}}"},
	}

	result, err := Transform(source, mappings)
	if err != nil {
		t.Fatal(err)
	}

	if result["fullname"] != "张 三" {
		t.Errorf("expected fullname='张 三', got %v", result["fullname"])
	}
	if result["ref"] != "ID-42" {
		t.Errorf("expected ref='ID-42', got %v", result["ref"])
	}
}

func TestTransform_Expression(t *testing.T) {
	source := map[string]any{
		"price":    float64(9.99),
		"quantity": float64(3),
	}
	mappings := []model.Mapping{
		{Target: "amount_cent", Transform: "expression", Value: "price * 100"},
		{Target: "total", Transform: "expression", Value: "price * quantity"},
		{Target: "with_tax", Transform: "expression", Value: "price + 1.5"},
	}

	result, err := Transform(source, mappings)
	if err != nil {
		t.Fatal(err)
	}

	if result["amount_cent"] != 999.0 {
		t.Errorf("expected amount_cent=999, got %v", result["amount_cent"])
	}
	if result["total"] != 29.97 {
		t.Errorf("expected total=29.97, got %v", result["total"])
	}
	if result["with_tax"] != 11.49 {
		t.Errorf("expected with_tax=11.49, got %v", result["with_tax"])
	}
}
