package sunspec

import (
	"errors"
	"fmt"
	"strconv"

	omwei "github.com/omwei-org/omwei-iot-go"
	gosunspec "github.com/otfabric/go-sunspec"
)

var (
	ErrPointNotImplemented = errors.New("SunSpec point is not implemented")
	ErrUnexpectedRawType   = errors.New("unexpected SunSpec raw value type")
	ErrMissingScaleFactor  = errors.New("required SunSpec scale factor is missing")
	ErrScaledValueMismatch = errors.New("SunSpec exact decimal disagrees with upstream scaled value")
)

// SourceSampleFromDecodedPoint converts a decoded signed-int16 SunSpec point
// into a transport-neutral OMWEI sample. The upstream DecodedPoint must come
// from one successful ReadModel snapshot: its RawValue, SFRawValue and
// ScaledValue must describe the same decoded acquisition.
//
// The exact decimal produced from RawValue and SFRawValue is the only value
// sent to the OMWEI mapper. ScaledValue is used solely as a strict consistency
// cross-check; it never supplies or modifies the mapped value.
func SourceSampleFromDecodedPoint(source omwei.SourceIdentity, point gosunspec.DecodedPoint) (omwei.SourceSample, error) {
	if !point.Implemented {
		return omwei.SourceSample{}, ErrPointNotImplemented
	}
	if point.Name != "W" || point.Type != "int16" {
		return omwei.SourceSample{}, fmt.Errorf("%w: point must be W/int16, got %s/%s", ErrUnexpectedRawType, point.Name, point.Type)
	}
	raw, ok := point.RawValue.(int16)
	if !ok {
		return omwei.SourceSample{}, fmt.Errorf("%w: W has dynamic type %T, want int16", ErrUnexpectedRawType, point.RawValue)
	}
	if point.SFRawValue == nil {
		return omwei.SourceSample{}, ErrMissingScaleFactor
	}
	if source.Protocol != "sunspec" || source.Device == "" || source.Point != "model:103:W" {
		return omwei.SourceSample{}, omwei.ErrInvalidSourceMapping
	}
	value, err := omwei.DecodeSunSpecScaledInteger(int64(raw), *point.SFRawValue)
	if err != nil {
		return omwei.SourceSample{}, fmt.Errorf("decode SunSpec scale factor: %w", err)
	}
	if point.ScaledValue == nil {
		return omwei.SourceSample{}, fmt.Errorf("%w: upstream ScaledValue is nil", ErrScaledValueMismatch)
	}
	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return omwei.SourceSample{}, fmt.Errorf("cross-check exact decimal: %w", err)
	}
	if floatValue != *point.ScaledValue {
		return omwei.SourceSample{}, fmt.Errorf("%w: decimal %q parses to %.17g, upstream has %.17g", ErrScaledValueMismatch, value, floatValue, *point.ScaledValue)
	}
	return omwei.SourceSample{Source: source, Value: value, Unit: point.Units}, nil
}
