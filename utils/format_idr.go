package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// FormatIDR formats an integer as Indonesian currency with thousands separators.
func FormatIDR(value interface{}) string {
	var isNegative bool
	var valueStr string

	switch v := value.(type) {
	case int:
		isNegative = v < 0
		if isNegative {
			v = -v
		}
		valueStr = fmt.Sprintf("%d", v)
	case int64:
		isNegative = v < 0
		if isNegative {
			v = -v
		}
		valueStr = fmt.Sprintf("%d", v)
	case float32, float64:
		floatValue := fmt.Sprintf("%f", v)
		parts := strings.SplitN(floatValue, ".", 2)
		intPart := parts[0]
		isNegative = strings.HasPrefix(intPart, "-")
		if isNegative {
			intPart = strings.TrimPrefix(intPart, "-")
		}
		valueStr = intPart
		if len(parts) > 1 {
			valueStr += "." + parts[1][:2] // Keep two decimal places
		}
	default:
		return "Invalid type"
	}

	length := len(valueStr)
	var result strings.Builder

	// Insert points every three digits from the right
	for i, digit := range valueStr {
		if digit == '.' {
			result.WriteRune(digit)
			continue
		}
		if i > 0 && (length-i)%3 == 0 && valueStr[i-1] != '.' {
			result.WriteString(".")
		}
		result.WriteRune(digit)
	}

	// Add a negative sign if the value was negative
	if isNegative {
		return "-Rp" + result.String()
	}

	return "Rp" + result.String()
}

func FormatNumberWithSpaces3(n int) string {
	s := strconv.Itoa(n)
	var result []string

	// process from the end
	for i, count := len(s)-1, 0; i >= 0; i, count = i-1, count+1 {
		result = append([]string{string(s[i])}, result...)
		if count%3 == 2 && i != 0 {
			result = append([]string{" "}, result...)
		}
	}

	return strings.Join(result, "")
}
