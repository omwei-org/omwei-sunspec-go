package opcua

import (
	"errors"
	"testing"
)

func TestResolveEngineeringUnitUsesNamespaceAndUnitID(t *testing.T) {
	configured := map[EngineeringUnitID]string{
		{NamespaceURI: "urn:example:units", UnitID: 42}: "W",
	}
	got, err := ResolveEngineeringUnit("urn:example:units", 42, configured)
	if err != nil {
		t.Fatal(err)
	}
	if got != "W" {
		t.Fatalf("got unit %q, want W", got)
	}
}

func TestResolveEngineeringUnitFailsClosed(t *testing.T) {
	configured := map[EngineeringUnitID]string{
		{NamespaceURI: "urn:example:units", UnitID: 42}: "W",
	}
	tests := []struct {
		name string
		namespace string
		unitID int32
		configured map[EngineeringUnitID]string
	}{
		{"unknown unit id", "urn:example:units", 43, configured},
		{"unknown namespace", "urn:other:units", 42, configured},
		{"empty namespace", " ", 42, configured},
		{"empty allowlist", "urn:example:units", 42, nil},
		{"blank configured unit", "urn:example:units", 42, map[EngineeringUnitID]string{{NamespaceURI:"urn:example:units",UnitID:42}:"  "}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveEngineeringUnit(tt.namespace, tt.unitID, tt.configured)
			if !errors.Is(err, ErrMissingUnit) {
				t.Fatalf("want ErrMissingUnit, got %v", err)
			}
		})
	}
}

func TestResolvedEngineeringUnitFeedsSampleAdapter(t *testing.T) {
	configured := map[EngineeringUnitID]string{
		{NamespaceURI: "urn:example:units", UnitID: 42}: "W",
	}
	unit, err := ResolveEngineeringUnit("urn:example:units", 42, configured)
	if err != nil {
		t.Fatal(err)
	}
	sample, err := SourceSampleFromValue(testSource(), 123.4, unit)
	if err != nil {
		t.Fatal(err)
	}
	if sample.Unit != "W" || sample.Value != "123.4" {
		t.Fatalf("unexpected sample: %+v", sample)
	}
}
