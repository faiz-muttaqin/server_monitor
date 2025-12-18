package utils

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// TryConvertStringTo mencoba convert string ke tipe target.
// Kalau gagal, return zero value + error.
func TryConvertStringTo[T ~int | ~int64 | ~int32 | ~int16 | ~int8 |
	~uint | ~uint64 | ~uint32 | ~uint16 | ~uint8 |
	~float32 | ~float64 |
	~complex64 | ~complex128 |
	~bool | ~string](input string) (T, error) {

	var result any
	var err error

	switch any(*new(T)).(type) {
	case int:
		var v int
		v, err = strconv.Atoi(input)
		result = v
	case int8:
		var v int64
		v, err = strconv.ParseInt(input, 10, 8)
		result = int8(v)
	case int16:
		var v int64
		v, err = strconv.ParseInt(input, 10, 16)
		result = int16(v)
	case int32:
		var v int64
		v, err = strconv.ParseInt(input, 10, 32)
		result = int32(v)
	case int64:
		var v int64
		v, err = strconv.ParseInt(input, 10, 64)
		result = v

	case uint:
		var v uint64
		v, err = strconv.ParseUint(input, 10, 64)
		result = uint(v)
	case uint8:
		var v uint64
		v, err = strconv.ParseUint(input, 10, 8)
		result = uint8(v)
	case uint16:
		var v uint64
		v, err = strconv.ParseUint(input, 10, 16)
		result = uint16(v)
	case uint32:
		var v uint64
		v, err = strconv.ParseUint(input, 10, 32)
		result = uint32(v)
	case uint64:
		var v uint64
		v, err = strconv.ParseUint(input, 10, 64)
		result = v

	case float32:
		var v float64
		v, err = strconv.ParseFloat(input, 32)
		result = float32(v)
	case float64:
		var v float64
		v, err = strconv.ParseFloat(input, 64)
		result = v

	case complex64:
		var v complex128
		v, err = strconv.ParseComplex(input, 64)
		result = complex64(v)
	case complex128:
		var v complex128
		v, err = strconv.ParseComplex(input, 128)
		result = v
	case time.Duration:
		var v time.Duration
		v, err = time.ParseDuration(input)
		result = v
	case bool:
		lower := strings.ToLower(strings.TrimSpace(input))
		switch lower {
		case "true", "1", "-1":
			result = true
		case "false", "0", "":
			result = false
		default:
			err = fmt.Errorf("invalid bool: %s", input)
		}

	case string:
		result = input

	default:
		err = fmt.Errorf("unsupported type")
	}

	if err != nil {
		var zero T
		return zero, err
	}
	return result.(T), nil
}

// ConvertStringTo versi simple, tidak return error.
// Kalau gagal → log error, return zero value.
func ConvertStringTo[T ~int | ~int64 | ~int32 | ~int16 | ~int8 |
	~uint | ~uint64 | ~uint32 | ~uint16 | ~uint8 |
	~float32 | ~float64 |
	~complex64 | ~complex128 |
	~bool | ~string](input string, default_value ...T) T {

	v, err := TryConvertStringTo[T](input)
	if err != nil {
		logrus.Errorf("ConvertStringTo[%T]: cannot convert '%s': %v", v, input, err)
		if len(default_value) > 0 {
			return default_value[0]
		}
		var zero T
		return zero
	}
	return v
}

// ConvertToString mengubah berbagai tipe angka & bool ke string (base 10 untuk numerik)
func ConvertToString[T ~int | ~int64 | ~int32 | ~int16 | ~int8 |
	~uint | ~uint64 | ~uint32 | ~uint16 | ~uint8 |
	~float32 | ~float64 |
	~complex64 | ~complex128 |
	~bool | ~string](v T) string {

	switch val := any(v).(type) {
	case int, int64, int32, int16, int8:
		return fmt.Sprintf("%d", val)

	case uint, uint64, uint32, uint16, uint8:
		return fmt.Sprintf("%d", val)

	case float32:
		return fmt.Sprintf("%f", float64(val))
	case float64:
		return fmt.Sprintf("%f", val)

	case complex64, complex128:
		// pakai fmt biar output seperti "(1+2i)"
		return fmt.Sprint(val)

	case bool:
		// true/false jadi string langsung
		if val {
			return "true"
		}
		return "false"

	default:
		return fmt.Sprint(val)
	}
}

// ============================================================================
// Main Generic Conversion Functions (Interface{} → Any Type)
// ============================================================================

// ConvertTo converts interface{} value to target type T
// Supports: int variants, uint variants, float variants, bool, string, time.Time, time.Duration
// Returns zero value if conversion fails
func ConvertTo[T any](value interface{}) T {
	result, err := TryConvertTo[T](value)
	if err != nil {
		logrus.Errorf("ConvertTo[%T]: cannot convert '%v' (%T): %v", result, value, value, err)
		var zero T
		return zero
	}
	return result
}

// TryConvertTo attempts to convert interface{} value to target type T
// Returns the converted value and error if conversion fails
func TryConvertTo[T any](value interface{}) (T, error) {
	var zero T

	// Handle nil input
	if value == nil {
		return zero, fmt.Errorf("cannot convert nil value")
	}

	// If value is already target type, return directly
	if v, ok := value.(T); ok {
		return v, nil
	}

	// Get the target type
	targetType := reflect.TypeOf(zero)
	sourceValue := reflect.ValueOf(value)

	// Handle pointer dereferencing
	if sourceValue.Kind() == reflect.Ptr {
		if sourceValue.IsNil() {
			return zero, fmt.Errorf("cannot convert nil pointer")
		}
		sourceValue = sourceValue.Elem()
		value = sourceValue.Interface()
	}

	// Try direct type conversion first
	if sourceValue.Type().ConvertibleTo(targetType) {
		converted := sourceValue.Convert(targetType)
		return converted.Interface().(T), nil
	}

	// Handle string source conversions
	if sourceStr, ok := value.(string); ok {
		return convertStringToType[T](sourceStr)
	}

	// Handle numeric conversions
	return convertNumericToType[T](value, sourceValue)
}

// ConvertToWithDefault converts interface{} to target type with default value fallback
func ConvertToWithDefault[T any](value interface{}, defaultValue T) T {
	result, err := TryConvertTo[T](value)
	if err != nil {
		logrus.Debugf("ConvertToWithDefault[%T]: using default value due to error: %v", defaultValue, err)
		return defaultValue
	}
	return result
}

// MustConvertTo converts interface{} to target type, panics on failure
func MustConvertTo[T any](value interface{}) T {
	result, err := TryConvertTo[T](value)
	if err != nil {
		panic(fmt.Sprintf("MustConvertTo[%T]: conversion failed: %v", result, err))
	}
	return result
}

// ============================================================================
// Helper Functions
// ============================================================================

// convertStringToType handles string to various type conversions
func convertStringToType[T any](input string) (T, error) {
	var result any
	var err error
	var zero T

	// Clean input
	input = strings.TrimSpace(input)

	switch any(zero).(type) {
	// Integer types
	case int:
		var v int
		v, err = strconv.Atoi(input)
		result = v
	case int8:
		var v int64
		v, err = strconv.ParseInt(input, 10, 8)
		result = int8(v)
	case int16:
		var v int64
		v, err = strconv.ParseInt(input, 10, 16)
		result = int16(v)
	case int32:
		var v int64
		v, err = strconv.ParseInt(input, 10, 32)
		result = int32(v)
	case int64:
		var v int64
		v, err = strconv.ParseInt(input, 10, 64)
		result = v

	// Unsigned integer types
	case uint:
		var v uint64
		v, err = strconv.ParseUint(input, 10, 64)
		result = uint(v)
	case uint8:
		var v uint64
		v, err = strconv.ParseUint(input, 10, 8)
		result = uint8(v)
	case uint16:
		var v uint64
		v, err = strconv.ParseUint(input, 10, 16)
		result = uint16(v)
	case uint32:
		var v uint64
		v, err = strconv.ParseUint(input, 10, 32)
		result = uint32(v)
	case uint64:
		var v uint64
		v, err = strconv.ParseUint(input, 10, 64)
		result = v

	// Float types
	case float32:
		var v float64
		v, err = strconv.ParseFloat(input, 32)
		result = float32(v)
	case float64:
		var v float64
		v, err = strconv.ParseFloat(input, 64)
		result = v

	// Complex types
	case complex64:
		var v complex128
		v, err = strconv.ParseComplex(input, 64)
		result = complex64(v)
	case complex128:
		var v complex128
		v, err = strconv.ParseComplex(input, 128)
		result = v

	// Boolean type
	case bool:
		result, err = parseBool(input)

	// String type
	case string:
		result = input

	// Time types
	case time.Time:
		result, err = parseTime(input)
	case time.Duration:
		var v time.Duration
		v, err = time.ParseDuration(input)
		result = v

	default:
		err = fmt.Errorf("unsupported target type: %T", zero)
	}

	if err != nil {
		return zero, err
	}
	return result.(T), nil
}

// convertNumericToType handles numeric to numeric type conversions
func convertNumericToType[T any](value interface{}, sourceValue reflect.Value) (T, error) {
	var zero T

	// Get numeric value as float64 for universal conversion
	var numValue float64

	switch sourceValue.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		numValue = float64(sourceValue.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		numValue = float64(sourceValue.Uint())
	case reflect.Float32, reflect.Float64:
		numValue = sourceValue.Float()
	case reflect.Bool:
		if sourceValue.Bool() {
			numValue = 1
		} else {
			numValue = 0
		}
	default:
		return zero, fmt.Errorf("cannot convert %T to %T", value, zero)
	}

	// Convert to target type
	var result any
	switch any(zero).(type) {
	case int:
		result = int(numValue)
	case int8:
		result = int8(numValue)
	case int16:
		result = int16(numValue)
	case int32:
		result = int32(numValue)
	case int64:
		result = int64(numValue)
	case uint:
		result = uint(numValue)
	case uint8:
		result = uint8(numValue)
	case uint16:
		result = uint16(numValue)
	case uint32:
		result = uint32(numValue)
	case uint64:
		result = uint64(numValue)
	case float32:
		result = float32(numValue)
	case float64:
		result = numValue
	case bool:
		result = numValue != 0
	case string:
		result = fmt.Sprintf("%v", value)
	default:
		return zero, fmt.Errorf("unsupported target type: %T", zero)
	}

	return result.(T), nil
}

// parseBool handles flexible boolean parsing
func parseBool(input string) (bool, error) {
	lower := strings.ToLower(input)
	switch lower {
	case "true", "1", "-1", "yes", "y", "on", "enabled":
		return true, nil
	case "false", "0", "", "no", "n", "off", "disabled":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool value: %s", input)
	}
}

// parseTime handles flexible time parsing
func parseTime(input string) (time.Time, error) {
	// Common time formats to try
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
		"15:04:05",
		"2006/01/02 15:04:05",
		"2006/01/02",
		"01/02/2006 15:04:05",
		"01/02/2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, input); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %s", input)
}
