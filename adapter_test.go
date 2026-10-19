package sunspec

import (
	"errors"
	"strconv"
	"testing"

	omwei "github.com/omwei-org/omwei-iot-go"
	gosunspec "github.com/otfabric/go-sunspec"
)

func testMapping() omwei.SourceMapping {
	return omwei.SourceMapping{
		Source: omwei.SourceIdentity{Protocol: "sunspec", Device: "provisioned-inverter-1", Point: "model:103:W"},
		DescriptorID: 69, SourceUnit: "W", CanonicalUnit: "watt",
		ScaleNumerator: 1, ScaleDenominator: 1,
		OffsetNumerator: 0, OffsetDenominator: 1,
	}
}

func int16ptr(v int16) *int16 { return &v }
func floatptr(v float64) *float64 { return &v }

func point(raw any, sf int16, scaled float64) gosunspec.DecodedPoint {
	return gosunspec.DecodedPoint{
		Name: "W", Type: "int16", RawValue: raw, SFRawValue: int16ptr(sf),
		ScaledValue: floatptr(scaled), Units: "W", Implemented: true,
	}
}

func TestSourceSampleFromDecodedPointAndMapping(t *testing.T) {
	mapping := testMapping()
	p := point(int16(1234), -1, 123.4)
	sample, err := SourceSampleFromDecodedPoint(mapping.Source, p)
	if err != nil { t.Fatal(err) }
	if sample.Value != "123.4" || sample.Unit != "W" || sample.Source != mapping.Source {
		t.Fatalf("unexpected sample: %+v", sample)
	}
	result, err := omwei.MapSourceSample(mapping, sample)
	if err != nil { t.Fatal(err) }
	if result.Atom.CanonicalValue != 123 || !result.Rounded {
		t.Fatalf("want atom 123 with Rounded=true, got %+v", result)
	}
}

func TestExactDecimal123PointZeroDoesNotRound(t *testing.T) {
	mapping := testMapping()
	result, err := omwei.MapSourceSample(mapping, omwei.SourceSample{
		Source: mapping.Source, Value: "123.0", Unit: "W",
	})
	if err != nil { t.Fatal(err) }
	if result.Atom.CanonicalValue != 123 || result.Rounded {
		t.Fatalf("want atom 123 with Rounded=false, got %+v", result)
	}
}

func TestHalfwayRoundsAwayFromZero(t *testing.T) {
	mapping := testMapping()
	p := point(int16(1235), -1, 123.5)
	sample, err := SourceSampleFromDecodedPoint(mapping.Source, p)
	if err != nil { t.Fatal(err) }
	result, err := omwei.MapSourceSample(mapping, sample)
	if err != nil { t.Fatal(err) }
	if result.Atom.CanonicalValue != 124 || !result.Rounded {
		t.Fatalf("want atom 124 with Rounded=true, got %+v", result)
	}
}

func TestNegativeActivePowerFailsClosed(t *testing.T) {
	mapping := testMapping()
	p := point(int16(-1234), -1, -123.4)
	sample, err := SourceSampleFromDecodedPoint(mapping.Source, p)
	if err != nil { t.Fatal(err) }
	if _, err := omwei.MapSourceSample(mapping, sample); !errors.Is(err, omwei.ErrOutOfRange) {
		t.Fatalf("want out_of_range for negative active power, got %v", err)
	}
}

func TestNotImplementedPointRejected(t *testing.T) {
	p := point(int16(1234), -1, 123.4)
	p.Implemented = false
	if _, err := SourceSampleFromDecodedPoint(testMapping().Source, p); !errors.Is(err, ErrPointNotImplemented) {
		t.Fatalf("want not-implemented rejection, got %v", err)
	}
}

func TestMissingScaleFactorRejected(t *testing.T) {
	p := point(int16(1234), -1, 123.4)
	p.SFRawValue = nil
	if _, err := SourceSampleFromDecodedPoint(testMapping().Source, p); !errors.Is(err, ErrMissingScaleFactor) {
		t.Fatalf("want missing scale factor rejection, got %v", err)
	}
}

func TestUnexpectedRawTypeRejected(t *testing.T) {
	p := point(int32(1234), -1, 123.4)
	if _, err := SourceSampleFromDecodedPoint(testMapping().Source, p); !errors.Is(err, ErrUnexpectedRawType) {
		t.Fatalf("want raw type rejection, got %v", err)
	}
}

func TestScaledValueCrossCheckIsExact(t *testing.T) {
	p := point(int16(1234), -1, 123.4)
	if _, err := SourceSampleFromDecodedPoint(testMapping().Source, p); err != nil { t.Fatal(err) }
	p.ScaledValue = floatptr(123.40000000000002)
	if _, err := SourceSampleFromDecodedPoint(testMapping().Source, p); !errors.Is(err, ErrScaledValueMismatch) {
		t.Fatalf("want strict mismatch, got %v", err)
	}
}

func TestCrossCheckUsesParseFloatOnlyForComparison(t *testing.T) {
	p := point(int16(1234), -1, 123.4)
	sample, err := SourceSampleFromDecodedPoint(testMapping().Source, p)
	if err != nil { t.Fatal(err) }
	parsed, err := strconv.ParseFloat(sample.Value, 64)
	if err != nil { t.Fatal(err) }
	if parsed != *p.ScaledValue { t.Fatalf("exact float cross-check failed: %.17g != %.17g", parsed, *p.ScaledValue) }
}

func TestInvalidIdentityRejected(t *testing.T) {
	source := testMapping().Source
	source.Device = ""
	if _, err := SourceSampleFromDecodedPoint(source, point(int16(1234), -1, 123.4)); !errors.Is(err, omwei.ErrInvalidSourceMapping) {
		t.Fatalf("want invalid identity rejection, got %v", err)
	}
}
