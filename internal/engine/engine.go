package engine

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nianhe/nianhe/internal/model"
)

// Transform applies mapping rules to a source JSON object and produces a target JSON object.
// Both input and output are map[string]any (unmarshalled JSON).
func Transform(source map[string]any, mappings []model.Mapping) (map[string]any, error) {
	result := make(map[string]any)
	for _, m := range mappings {
		var value any
		switch m.Transform {
		case "direct", "":
			if m.Source == "" {
				return nil, fmt.Errorf("mapping to %q: source field is required for direct mapping", m.Target)
			}
			v, err := getNestedValue(source, m.Source)
			if err != nil {
				continue // skip if source field not found in this payload
			}
			value = v
		case "constant":
			value = m.Value
		default:
			return nil, fmt.Errorf("unsupported transform type: %s", m.Transform)
		}
		if err := setNestedValue(result, m.Target, value); err != nil {
			return nil, fmt.Errorf("set %s: %w", m.Target, err)
		}
	}
	return result, nil
}

// getNestedValue retrieves a value from a nested map using dot notation.
// e.g. "data.user.name" -> source["data"]["user"]["name"]
func getNestedValue(data map[string]any, path string) (any, error) {
	parts := strings.Split(path, ".")
	var current any = data
	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			val, ok := v[part]
			if !ok {
				return nil, fmt.Errorf("key %q not found", part)
			}
			current = val
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(v) {
				return nil, fmt.Errorf("invalid array index %q", part)
			}
			current = v[idx]
		default:
			return nil, fmt.Errorf("cannot traverse into %T at %q", current, part)
		}
	}
	return current, nil
}

// setNestedValue sets a value in a nested map using dot notation.
// Creates intermediate maps as needed.
func setNestedValue(data map[string]any, path string, value any) error {
	parts := strings.Split(path, ".")
	current := data
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return nil
		}
		next, ok := current[part]
		if !ok {
			next = make(map[string]any)
			current[part] = next
		}
		m, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("path conflict at %q: expected map, got %T", part, next)
		}
		current = m
	}
	return nil
}
