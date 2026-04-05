package engine

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/johnzhangchina/dataferry/internal/model"
)

// EvaluateConditions checks conditions with the given logic ("and" or "or").
// Empty conditions always pass. Default logic is "and".
func EvaluateConditions(source map[string]any, conditions []model.Condition, logic string) (bool, string) {
	if len(conditions) == 0 {
		return true, ""
	}

	if logic == "or" {
		// OR: at least one must pass
		var lastReason string
		for _, c := range conditions {
			if pass, _ := evalOneCondition(source, c); pass {
				return true, ""
			} else {
				lastReason = fmt.Sprintf("no condition matched (last: %s)", conditionDesc(c))
			}
		}
		return false, lastReason
	}

	// AND (default): all must pass
	for _, c := range conditions {
		if pass, reason := evalOneCondition(source, c); !pass {
			return false, reason
		}
	}
	return true, ""
}

func conditionDesc(c model.Condition) string {
	if c.Operator == "exists" {
		return fmt.Sprintf("%s exists", c.Field)
	}
	return fmt.Sprintf("%s %s %q", c.Field, c.Operator, c.Value)
}

func evalOneCondition(source map[string]any, c model.Condition) (bool, string) {
	if c.Operator == "exists" {
		_, err := getNestedValue(source, c.Field)
		if err != nil {
			return false, fmt.Sprintf("field %q does not exist", c.Field)
		}
		return true, ""
	}

	val, err := getNestedValue(source, c.Field)
	if err != nil {
		return false, fmt.Sprintf("field %q not found", c.Field)
	}

	valStr := fmt.Sprintf("%v", val)

	switch c.Operator {
	case "==":
		if valStr != c.Value {
			return false, fmt.Sprintf("%s == %q, got %q", c.Field, c.Value, valStr)
		}
	case "!=":
		if valStr == c.Value {
			return false, fmt.Sprintf("%s != %q, got %q", c.Field, c.Value, valStr)
		}
	case ">":
		fVal, err1 := strconv.ParseFloat(valStr, 64)
		fExpected, err2 := strconv.ParseFloat(c.Value, 64)
		if err1 != nil || err2 != nil {
			return false, fmt.Sprintf("%s > %s: non-numeric comparison", c.Field, c.Value)
		}
		if fVal <= fExpected {
			return false, fmt.Sprintf("%s > %s, got %s", c.Field, c.Value, valStr)
		}
	case "<":
		fVal, err1 := strconv.ParseFloat(valStr, 64)
		fExpected, err2 := strconv.ParseFloat(c.Value, 64)
		if err1 != nil || err2 != nil {
			return false, fmt.Sprintf("%s < %s: non-numeric comparison", c.Field, c.Value)
		}
		if fVal >= fExpected {
			return false, fmt.Sprintf("%s < %s, got %s", c.Field, c.Value, valStr)
		}
	case "contains":
		if !strings.Contains(valStr, c.Value) {
			return false, fmt.Sprintf("%s contains %q, got %q", c.Field, c.Value, valStr)
		}
	default:
		return false, fmt.Sprintf("unknown operator %q", c.Operator)
	}
	return true, ""
}

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
