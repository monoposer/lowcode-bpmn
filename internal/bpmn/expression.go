package bpmn

import (
	"fmt"
	"strconv"
	"strings"
)

// EvalCondition evaluates a simple BPMN condition against process variables.
// Supported forms:
//   - "field == value" (string or numeric)
//   - "field != value"
//   - "field >= value", "field > value", "field <= value", "field < value" (numeric)
//   - "field" (truthy check)
// Empty condition evaluates to true.
func EvalCondition(expr string, vars map[string]any) (bool, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return true, nil
	}

	for _, op := range []string{">=", "<=", "!=", "==", ">", "<"} {
		if strings.Contains(expr, op) {
			parts := strings.SplitN(expr, op, 2)
			if len(parts) != 2 {
				return false, fmt.Errorf("invalid expression: %s", expr)
			}
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])
			right = strings.Trim(right, "\"'")
			v, ok := vars[left]
			if !ok {
				if op == "!=" {
					return true, nil
				}
				return false, nil
			}
			switch op {
			case "==":
				return compareValues(v, right), nil
			case "!=":
				return !compareValues(v, right), nil
			default:
				return compareNumericOp(v, right, op)
			}
		}
	}

	// Bare field name — truthy.
	v, ok := vars[expr]
	if !ok {
		return false, nil
	}
	switch t := v.(type) {
	case bool:
		return t, nil
	case string:
		return t != "" && t != "false" && t != "0", nil
	case float64:
		return t != 0, nil
	case int:
		return t != 0, nil
	default:
		return v != nil, nil
	}
}

func compareNumericOp(left any, right string, op string) (bool, error) {
	lf, err1 := toFloat(left)
	rf, err2 := strconv.ParseFloat(right, 64)
	if err1 != nil || err2 != nil {
		return false, fmt.Errorf("numeric comparison requires numbers: %s", op)
	}
	switch op {
	case ">=":
		return lf >= rf, nil
	case ">":
		return lf > rf, nil
	case "<=":
		return lf <= rf, nil
	case "<":
		return lf < rf, nil
	default:
		return false, fmt.Errorf("unknown op: %s", op)
	}
}

func toFloat(v any) (float64, error) {
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
		return 0, fmt.Errorf("not numeric")
	}
}

func compareValues(left any, right string) bool {
	switch actual := left.(type) {
	case string:
		return actual == right
	case bool:
		rb, err := strconv.ParseBool(right)
		if err != nil {
			return fmt.Sprint(actual) == right
		}
		return actual == rb
	case float64:
		r, err := strconv.ParseFloat(right, 64)
		if err != nil {
			return fmt.Sprint(actual) == right
		}
		return actual == r
	case int:
		r, err := strconv.Atoi(right)
		if err != nil {
			return fmt.Sprint(actual) == right
		}
		return actual == r
	default:
		return fmt.Sprint(actual) == right
	}
}
