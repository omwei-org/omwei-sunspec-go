package omwei

import (
	"errors"

	core "github.com/omwei-org/omwei-iot-go"
)

type SourceIdentity = core.SourceIdentity
type SourceMapping = core.SourceMapping
type SourceSample = core.SourceSample
type ConversionResult = core.ConversionResult
type Atom = core.Atom

var (
	ErrSourceIdentityMismatch = core.ErrSourceIdentityMismatch
	ErrInvalidSourceMapping = core.ErrInvalidSourceMapping
	ErrOutOfRange = core.ErrOutOfRange
	ErrInvalidSunSpecScaleFactor = core.ErrInvalidSunSpecScaleFactor
	ErrInvalidSunSpecValue = errors.New("invalid or not-implemented SunSpec value")
)

func ValidateSourceMapping(m SourceMapping) error { return core.ValidateSourceMapping(m) }
func Encode(a Atom) [core.AtomSize]byte { return core.Encode(a) }
func FromDecimalWithReport(id uint16, value string) (core.ConversionResult, error) {
	return core.FromDecimalWithReport(id, value)
}
func MapSourceSample(m SourceMapping, s SourceSample) (ConversionResult, error) {
	return core.MapSourceSample(m, s)
}
func DecodeSunSpecScaledInteger(raw int64, sf int16) (string, error) {
	return core.DecodeSunSpecScaledInteger(raw, sf)
}
