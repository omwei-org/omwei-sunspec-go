package opqualive

import (
	"context"
	"reflect"
	"testing"
	"time"

	omwei "github.com/omwei-org/omwei-iot-go"
	sunspecadapter "github.com/omwei-org/omwei-sunspec-go"
	omweiopcua "github.com/omwei-org/omwei-sunspec-go/opcua"
	gosunspec "github.com/otfabric/go-sunspec"
	opcua "github.com/otfabric/go-opcua"
	"github.com/otfabric/go-opcua/server"
	"github.com/otfabric/go-opcua/ua"
)

// TestCrossProtocolLiveOPCUAAgainstDecodedSunSpecFixture runs a real OPC UA
// client/server exchange and compares its normalized result with the OMWEI
// SunSpec adapter fed a go-sunspec DecodedPoint fixture. The SunSpec point is
// a decoded-output fixture, not a live Modbus/go-sunspec acquisition, and both
// values are synthetic. This test demonstrates protocol-to-semantic parity;
// it does not claim parity of physical measurements.
func TestCrossProtocolLiveOPCUAAgainstDecodedSunSpecFixture(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	srv, err := server.New(
		server.EndPoint("localhost", 48492),
		server.EnableSecurity("None", ua.MessageSecurityModeNone),
		server.EnableAuthMode(ua.UserTokenTypeAnonymous),
	)
	if err != nil {
		t.Fatalf("create OPC UA test server: %v", err)
	}
	ns := server.NewNodeNameSpace(srv, testNamespaceURI)
	valueNode := ns.AddNewVariableStringNode("ActivePower", float64(123.4))
	unitNode := ns.AddNewVariableStringNode("ActivePowerEngineeringUnits", ua.NewExtensionObject(&ua.EUInformation{
		NamespaceURI: testUnitsURI,
		UnitID:       testUnitID,
		DisplayName:  ua.NewLocalizedText("W"),
		Description:  ua.NewLocalizedText("test watt unit"),
	}))
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("start OPC UA test server: %v", err)
	}
	defer srv.Close()

	client, err := opcua.NewClient("opc.tcp://localhost:48492")
	if err != nil {
		t.Fatalf("create OPC UA client: %v", err)
	}
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect OPC UA client: %v", err)
	}
	defer client.Close(context.Background())

	// Read the value and its EUInformation in one OPC UA Read service call.
	results, err := client.ReadMulti(ctx, []opcua.ReadItem{
		{NodeID: valueNode.ID(), AttributeID: ua.AttributeIDValue},
		{NodeID: unitNode.ID(), AttributeID: ua.AttributeIDValue},
	})
	if err != nil {
		t.Fatalf("read ActivePower and EngineeringUnits: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("ReadMulti returned %d results, want 2", len(results))
	}
	for i, result := range results {
		if result.StatusCode != ua.StatusOK || result.DataValue == nil || result.DataValue.Value == nil {
			t.Fatalf("read result %d invalid: status=%v result=%+v", i, result.StatusCode, result)
		}
	}

	opcValue, ok := results[0].DataValue.Value.Value().(float64)
	if !ok {
		t.Fatalf("unexpected OPC UA value type %T", results[0].DataValue.Value.Value())
	}
	extension, ok := results[1].DataValue.Value.Value().(*ua.ExtensionObject)
	if !ok {
		t.Fatalf("EngineeringUnits has unexpected wire value type %T", results[1].DataValue.Value.Value())
	}
	unitInfo, ok := extension.Value.(*ua.EUInformation)
	if !ok {
		t.Fatalf("EUInformation extension body has unexpected type %T", extension.Value)
	}
	configuredUnits := map[omweiopcua.EngineeringUnitID]string{
		{NamespaceURI: testUnitsURI, UnitID: testUnitID}: "W",
	}
	unit, err := omweiopcua.ResolveEngineeringUnit(unitInfo.NamespaceURI, unitInfo.UnitID, configuredUnits)
	if err != nil {
		t.Fatalf("resolve OPC UA unit: %v", err)
	}

	opcSource := omwei.SourceIdentity{
		Protocol: "opcua", Device: "loopback-test-server",
		Point: "nsu=" + testNamespaceURI + ";s=ActivePower",
	}
	opcSample, err := omweiopcua.SourceSampleFromValue(opcSource, opcValue, unit)
	if err != nil {
		t.Fatalf("normalize OPC UA sample: %v", err)
	}

	// Model the decoded output supplied by go-sunspec to the existing adapter.
	// The protocol acquisition/decoding itself is intentionally not simulated
	// as a live Modbus path in this test.
	sf := int16(-1)
	scaled := float64(123.4)
	sunSource := omwei.SourceIdentity{
		Protocol: "sunspec", Device: "decoded-fixture",
		Point: "model:103:W",
	}
	decoded := gosunspec.DecodedPoint{
		Name: "W", Type: "int16", RawValue: int16(1234),
		SFRawValue: &sf, ScaledValue: &scaled, Units: "W", Implemented: true,
	}
	sunSample, err := sunspecadapter.SourceSampleFromDecodedPoint(sunSource, decoded)
	if err != nil {
		t.Fatalf("normalize go-sunspec decoded point fixture: %v", err)
	}

	baseMapping := omwei.SourceMapping{
		DescriptorID: 69, SourceUnit: "W", CanonicalUnit: "watt",
		ScaleNumerator: 1, ScaleDenominator: 1,
		OffsetNumerator: 0, OffsetDenominator: 1,
	}
	sunMapping := baseMapping
	sunMapping.Source = sunSource
	opcMapping := baseMapping
	opcMapping.Source = opcSource

	sunResult, err := omwei.MapSourceSample(sunMapping, sunSample)
	if err != nil {
		t.Fatalf("map SunSpec decoded fixture: %v", err)
	}
	opcResult, err := omwei.MapSourceSample(opcMapping, opcSample)
	if err != nil {
		t.Fatalf("map live OPC UA sample: %v", err)
	}
	if !reflect.DeepEqual(sunResult.Atom, opcResult.Atom) {
		t.Fatalf("canonical atom mismatch: SunSpec=%+v OPC UA=%+v", sunResult.Atom, opcResult.Atom)
	}
	if sunResult.Rounded != opcResult.Rounded {
		t.Fatalf("Rounded mismatch: SunSpec=%v OPC UA=%v", sunResult.Rounded, opcResult.Rounded)
	}
	if sunResult.Atom.CanonicalValue != 123 || !sunResult.Rounded {
		t.Fatalf("unexpected parity result: atom=%+v Rounded=%v", sunResult.Atom, sunResult.Rounded)
	}
}
