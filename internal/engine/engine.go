package engine

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/johnzhangchina/dataferry/internal/model"
)

// Transform applies mapping rules to a source JSON object and produces a target JSON object.
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

		case "template":
			// String template with {{field.path}} placeholders.
			// e.g. "{{first_name}} {{last_name}}" or "ID-{{data.id}}"
			value = resolveTemplate(source, m.Value)

		case "expression":
			// Simple arithmetic: "field * 100", "field + 1", "field1 + field2"
			v, err := evalExpression(source, m.Value)
			if err != nil {
				continue
			}
			value = v

		default:
			return nil, fmt.Errorf("unsupported transform type: %s", m.Transform)
		}
		if err := setNestedValue(result, m.Target, value); err != nil {
			return nil, fmt.Errorf("set %s: %w", m.Target, err)
		}
	}
	return result, nil
}

var templatePattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// resolveTemplate replaces {{field.path}} with actual values from source.
func resolveTemplate(source map[string]any, template string) string {
	return templatePattern.ReplaceAllStringFunc(template, func(match string) string {
		path := strings.TrimSpace(match[2 : len(match)-2])
		v, err := getNestedValue(source, path)
		if err != nil {
			return match
		}
		return fmt.Sprintf("%v", v)
	})
}

// evalExpression evaluates simple arithmetic: "field * 100", "field1 + field2"
// Supports: +, -, *, /
// Operands can be field paths or numeric literals.
func evalExpression(source map[string]any, expr string) (float64, error) {
	expr = strings.TrimSpace(expr)

	// Find operator
	var op byte
	var opIdx int
	for i := 1; i < len(expr); i++ {
		if expr[i] == '+' || expr[i] == '-' || expr[i] == '*' || expr[i] == '/' {
			// Make sure it's not inside a negative number
			if expr[i] == '-' && i > 0 && (expr[i-1] == '*' || expr[i-1] == '/' || expr[i-1] == '+') {
				continue
			}
			op = expr[i]
			opIdx = i
			break
		}
	}

	if op == 0 {
		return resolveNumber(source, expr)
	}

	left, err := resolveNumber(source, strings.TrimSpace(expr[:opIdx]))
	if err != nil {
		return 0, err
	}
	right, err := resolveNumber(source, strings.TrimSpace(expr[opIdx+1:]))
	if err != nil {
		return 0, err
	}

	switch op {
	case '+':
		return left + right, nil
	case '-':
		return left - right, nil
	case '*':
		return left * right, nil
	case '/':
		if right == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return left / right, nil
	}
	return 0, fmt.Errorf("unknown operator: %c", op)
}

// resolveNumber gets a number from a field path or parses a literal.
func resolveNumber(source map[string]any, s string) (float64, error) {
	s = strings.TrimSpace(s)
	// Try as literal number first
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		return n, nil
	}
	// Try as field path
	v, err := getNestedValue(source, s)
	if err != nil {
		return 0, err
	}
	return toFloat64(v)
}

func toFloat64(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case string:
		return strconv.ParseFloat(n, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to number", v)
	}
}

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
