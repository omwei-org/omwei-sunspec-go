package opcua

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	omwei "github.com/omwei-org/omwei-iot-go"
)

var (
	ErrUnsupportedValue = errors.New("unsupported OPC UA numeric value type")
	ErrNonFiniteValue   = errors.New("OPC UA value is not finite")
	ErrMissingUnit      = errors.New("OPC UA engineering unit is missing")
	ErrInvalidSource    = errors.New("invalid OPC UA source identity")
	ErrDecimalPrecision = errors.New("OPC UA value exceeds OMWEI decimal precision")
)

// SourceSampleFromValue converts a numeric OPC UA value whose engineering unit
// has already been resolved into OMWEI's transport-neutral SourceSample.
//
// The caller must obtain value and unit from the same successful OPC UA read
// context and must resolve the engineering unit explicitly. This function does
// not browse the address space, infer units from display names, authenticate a
// server/device, or establish sample freshness.
//
// source.Point should be a stable, provisioned node identity (prefer namespace
// URI + identifier), not a display name. Floating-point values are rendered as
// their shortest round-trippable base-10 decimal; the semantic mapper then
// performs its existing exact decimal/rational conversion without float math.
func SourceSampleFromValue(source omwei.SourceIdentity, value any, unit string) (omwei.SourceSample, error) {
	if source.Protocol != "opcua" || strings.TrimSpace(source.Device) == "" ||
		strings.TrimSpace(source.Point) == "" {
		return omwei.SourceSample{}, fmt.Errorf("%w: protocol must be opcua and device/point must be provisioned", ErrInvalidSource)
	}
	if strings.TrimSpace(unit) == "" {
		return omwei.SourceSample{}, ErrMissingUnit
	}

	decimal, err := decimalString(value)
	if err != nil {
		return omwei.SourceSample{}, err
	}
	if err := checkDecimalContract(decimal); err != nil {
		return omwei.SourceSample{}, err
	}
	return omwei.SourceSample{Source: source, Value: decimal, Unit: unit}, nil
}

func decimalString(value any) (string, error) {
	switch v := value.(type) {
	case int8:
		return strconv.FormatInt(int64(v), 10), nil
	case int16:
		return strconv.FormatInt(int64(v), 10), nil
	case int32:
		return strconv.FormatInt(int64(v), 10), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case int:
		return strconv.FormatInt(int64(v), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint64:
		return strconv.FormatUint(v, 10), nil
	case uint:
		return strconv.FormatUint(uint64(v), 10), nil
	case float32:
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return "", ErrNonFiniteValue
		}
		return strconv.FormatFloat(float64(v), 'f', -1, 32), nil
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return "", ErrNonFiniteValue
		}
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	default:
		return "", fmt.Errorf("%w: %T", ErrUnsupportedValue, value)
	}
}

// checkDecimalContract keeps the adapter output inside the core's documented
// plain-decimal grammar (at most 39 integer digits and 18 fractional digits).
func checkDecimalContract(s string) error {
	unsigned := strings.TrimPrefix(strings.TrimPrefix(s, "-"), "+")
	whole, fraction, hasDot := strings.Cut(unsigned, ".")
	if len(strings.TrimLeft(whole, "0")) > 39 || (hasDot && len(fraction) > 18) {
		return fmt.Errorf("%w: %q", ErrDecimalPrecision, s)
	}
	return nil
}
