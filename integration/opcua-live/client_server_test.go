package opqualive

import (
	"context"
	"testing"
	"time"

	omwei "github.com/omwei-org/omwei-iot-go"
	omweiopcua "github.com/omwei-org/omwei-sunspec-go/opcua"
	opcua "github.com/otfabric/go-opcua"
	"github.com/otfabric/go-opcua/server"
	"github.com/otfabric/go-opcua/ua"
)

const (
	testNamespaceURI = "urn:omwei:cross-protocol-test"
	testUnitsURI     = "urn:omwei:test-unit-registry"
	testUnitID       = int32(42)
)

func TestLiveClientReadsValueAndEUInformation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	srv, err := server.New(
		server.EndPoint("localhost", 48491),
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

	client, err := opcua.NewClient("opc.tcp://localhost:48491")
	if err != nil {
		t.Fatalf("create OPC UA client: %v", err)
	}
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect OPC UA client: %v", err)
	}
	defer client.Close(context.Background())

	// Read the numeric value and EUInformation in one OPC UA Read service call,
	// so the sample does not combine two independently acquired read contexts.
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
		if result.StatusCode != ua.StatusOK {
			t.Fatalf("read result %d returned status %v", i, result.StatusCode)
		}
		if result.DataValue == nil || result.DataValue.Value == nil {
			t.Fatalf("read result %d returned no value", i)
		}
	}

	value := results[0].DataValue.Value.Value()
	if value != float64(123.4) {
		t.Fatalf("unexpected OPC UA value: %T(%v)", value, value)
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
		t.Fatalf("resolve server EUInformation: %v", err)
	}

	source := omwei.SourceIdentity{
		Protocol: "opcua",
		Device:   "loopback-test-server",
		Point:    "nsu=" + testNamespaceURI + ";s=ActivePower",
	}
	sample, err := omweiopcua.SourceSampleFromValue(source, value, unit)
	if err != nil {
		t.Fatalf("normalize live OPC UA read: %v", err)
	}
	if sample.Value != "123.4" || sample.Unit != "W" {
		t.Fatalf("unexpected normalized sample: %+v", sample)
	}
}
