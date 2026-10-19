package opcua

import (
	"errors"
	"math"
	"testing"

	omwei "github.com/omwei-org/omwei-iot-go"
)

func testSource() omwei.SourceIdentity {
	return omwei.SourceIdentity{Protocol: "opcua", Device: "plant-1", Point: "nsu=urn:plant;s=power.active"}
}

func TestSourceSampleFromValueNumericTypes(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"int16", int16(123), "123"},
		{"uint32", uint32(123), "123"},
		{"float32", float32(123.4), "123.4"},
		{"float64", float64(123.4), "123.4"},
		{"negative", int32(-12), "-12"},
		{"zero", float64(0), "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SourceSampleFromValue(testSource(), tt.in, "W")
			if err != nil { t.Fatal(err) }
			if got.Value != tt.want || got.Unit != "W" || got.Source != testSource() {
				t.Fatalf("got %+v; want value %q, unit W and configured source", got, tt.want)
			}
		})
	}
}

func TestOPCUASampleMapsToSameCanonicalAtomAsSunSpec(t *testing.T) {
	source := testSource()
	sample, err := SourceSampleFromValue(source, float64(123.4), "W")
	if err != nil { t.Fatal(err) }
	mapping := omwei.SourceMapping{
		Source: source, DescriptorID: 69, SourceUnit: "W", CanonicalUnit: "watt",
		ScaleNumerator: 1, ScaleDenominator: 1, OffsetNumerator: 0, OffsetDenominator: 1,
	}
	got, err := omwei.MapSourceSample(mapping, sample)
	if err != nil { t.Fatal(err) }
	if got.Atom.CanonicalValue != 123 || !got.Rounded {
		t.Fatalf("want canonical 123 and Rounded=true, got %+v", got)
	}
}

func TestInvalidInputsFailClosed(t *testing.T) {
	tests := []struct {
		name string
		source omwei.SourceIdentity
		value any
		unit string
		want error
	}{
		{"wrong protocol", omwei.SourceIdentity{Protocol:"sunspec", Device:"d", Point:"p"}, 1, "W", ErrInvalidSource},
		{"missing device", omwei.SourceIdentity{Protocol:"opcua", Point:"p"}, 1, "W", ErrInvalidSource},
		{"missing point", omwei.SourceIdentity{Protocol:"opcua", Device:"d"}, 1, "W", ErrInvalidSource},
		{"missing unit", testSource(), 1, "", ErrMissingUnit},
		{"unsupported type", testSource(), true, "W", ErrUnsupportedValue},
		{"NaN", testSource(), math.NaN(), "W", ErrNonFiniteValue},
		{"positive infinity", testSource(), math.Inf(1), "W", ErrNonFiniteValue},
		{"negative infinity", testSource(), math.Inf(-1), "W", ErrNonFiniteValue},
		{"excess precision", testSource(), 1e-19, "W", ErrDecimalPrecision},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SourceSampleFromValue(tt.source, tt.value, tt.unit)
			if !errors.Is(err, tt.want) { t.Fatalf("want %v, got %v", tt.want, err) }
		})
	}
}

func TestUnitMismatchRejectedByCoreMapper(t *testing.T) {
	source := testSource()
	sample, err := SourceSampleFromValue(source, 123.4, "kW")
	if err != nil { t.Fatal(err) }
	mapping := omwei.SourceMapping{
		Source: source, DescriptorID: 69, SourceUnit: "W", CanonicalUnit: "watt",
		ScaleNumerator: 1, ScaleDenominator: 1, OffsetNumerator: 0, OffsetDenominator: 1,
	}
	if _, err := omwei.MapSourceSample(mapping, sample); !errors.Is(err, omwei.ErrObservedSourceUnitMismatch) {
		t.Fatalf("want observed unit mismatch, got %v", err)
	}
}
