package omwei

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

type sunSpecVectorDocument struct {
	SchemaVersion string `json:"schema_version"`
	Vectors []struct {
		Name string `json:"name"`
		RawValue int64 `json:"raw_value"`
		ScaleFactor int16 `json:"scale_factor"`
		EngineeringValue string `json:"engineering_value"`
		DescriptorID uint16 `json:"descriptor_id"`
		EncodedValue int32 `json:"encoded_value"`
		WireHex string `json:"wire_hex"`
		Rounded bool `json:"rounded"`
	} `json:"vectors"`
}

func TestSunSpecModel103ConformanceVectors(t *testing.T) {
	data, err := os.ReadFile("testdata/sunspec-model-103-vectors-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc sunSpecVectorDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.SchemaVersion != "1" || len(doc.Vectors) == 0 {
		t.Fatalf("invalid conformance vector document: %+v", doc)
	}
	for _, vector := range doc.Vectors {
		t.Run(vector.Name, func(t *testing.T) {
			engineering, err := DecodeSunSpecScaledInteger(vector.RawValue, vector.ScaleFactor)
			if err != nil {
				t.Fatal(err)
			}
			if engineering != vector.EngineeringValue {
				t.Fatalf("engineering value: want %s, got %s", vector.EngineeringValue, engineering)
			}
			result, err := FromDecimalWithReport(vector.DescriptorID, engineering)
			if err != nil {
				t.Fatal(err)
			}
			if result.Atom.CanonicalValue != vector.EncodedValue {
				t.Fatalf("encoded value: want %d, got %d", vector.EncodedValue, result.Atom.CanonicalValue)
			}
			if result.Rounded != vector.Rounded {
				t.Fatalf("rounded: want %t, got %t", vector.Rounded, result.Rounded)
			}
			wire := Encode(result.Atom)
			wantWire, err := hex.DecodeString(vector.WireHex)
			if err != nil {
				t.Fatal(err)
			}
			if hex.EncodeToString(wire[:]) != hex.EncodeToString(wantWire) {
				t.Fatalf("wire bytes: want %x, got %x", wantWire, wire)
			}
		})
	}
}
