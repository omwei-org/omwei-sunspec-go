package sunspec

import (
	"errors"
	"reflect"
	"testing"

	omwei "github.com/omwei-org/omwei-iot-go"
	"github.com/omwei-org/omwei-sunspec-go/opcua"
)

// TestCrossProtocolSemanticParity uses synthetic values to prove that the
// protocol-specific adapters converge on the same core atom contract. It does
// not claim a live OPC UA server or physical SunSpec device was exercised.
func TestCrossProtocolSemanticParity(t *testing.T) {
	cases := []struct {
		name       string
		raw        int16
		sf         int16
		scaled     float64
		opcuaValue float64
		wantValue  int32
		wantRounded bool
	}{
		{name: "fractional watt", raw: 1234, sf: -1, scaled: 123.4, opcuaValue: 123.4, wantValue: 123, wantRounded: true},
		{name: "halfway rounds away from zero", raw: 1235, sf: -1, scaled: 123.5, opcuaValue: 123.5, wantValue: 124, wantRounded: true},
		{name: "exact integer", raw: 123, sf: 0, scaled: 123, opcuaValue: 123, wantValue: 123, wantRounded: false},
		{name: "negative power rejected", raw: -1234, sf: -1, scaled: -123.4, opcuaValue: -123.4},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sunSource := omwei.SourceIdentity{Protocol: "sunspec", Device: "inverter-1", Point: "model:103:W"}
			opcSource := omwei.SourceIdentity{Protocol: "opcua", Device: "inverter-1", Point: "nsu=urn:plant;s=power.active"}

			sunSample, err := SourceSampleFromDecodedPoint(sunSource, point(tc.raw, tc.sf, tc.scaled))
			if err != nil {
				t.Fatalf("SunSpec adapter: %v", err)
			}
			opcSample, err := opcua.SourceSampleFromValue(opcSource, tc.opcuaValue, "W")
			if err != nil {
				t.Fatalf("OPC UA adapter: %v", err)
			}

			sunMapping := testMapping()
			sunMapping.Source = sunSource
			opcMapping := sunMapping
			opcMapping.Source = opcSource

			sunResult, sunErr := omwei.MapSourceSample(sunMapping, sunSample)
			opcResult, opcErr := omwei.MapSourceSample(opcMapping, opcSample)
			if (sunErr != nil) != (opcErr != nil) {
				t.Fatalf("error mismatch: SunSpec=%v OPC UA=%v", sunErr, opcErr)
			}
			if sunErr != nil {
				if !errors.Is(sunErr, omwei.ErrOutOfRange) || !errors.Is(opcErr, omwei.ErrOutOfRange) {
					t.Fatalf("expected matching out-of-range errors, got SunSpec=%v OPC UA=%v", sunErr, opcErr)
				}
				return
			}
			if !reflect.DeepEqual(sunResult.Atom, opcResult.Atom) {
				t.Fatalf("atom mismatch: SunSpec=%+v OPC UA=%+v", sunResult.Atom, opcResult.Atom)
			}
			if sunResult.Rounded != opcResult.Rounded {
				t.Fatalf("Rounded mismatch: SunSpec=%v OPC UA=%v", sunResult.Rounded, opcResult.Rounded)
			}
			if sunResult.Atom.CanonicalValue != tc.wantValue || sunResult.Rounded != tc.wantRounded {
				t.Fatalf("unexpected canonical result: got value=%d Rounded=%v, want value=%d Rounded=%v",
					sunResult.Atom.CanonicalValue, sunResult.Rounded, tc.wantValue, tc.wantRounded)
			}
		})
	}
}
