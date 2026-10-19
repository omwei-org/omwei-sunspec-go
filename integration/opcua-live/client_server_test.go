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

	valueDV, err := client.ReadValue(ctx, valueNode.ID())
	if err != nil {
		t.Fatalf("read ActivePower: %v", err)
	}
	if valueDV == nil || valueDV.Value == nil {
		t.Fatal("ActivePower returned no value")
	}
	value := valueDV.Value.Value()
	if value != float64(123.4) {
		t.Fatalf("unexpected OPC UA value: %T(%v)", value, value)
	}

	unitDV, err := client.ReadValue(ctx, unitNode.ID())
	if err != nil {
		t.Fatalf("read EngineeringUnits: %v", err)
	}
	if unitDV == nil || unitDV.Value == nil {
		t.Fatal("EngineeringUnits returned no value")
	}
	extension, ok := unitDV.Value.Value().(*ua.ExtensionObject)
	if !ok {
		t.Fatalf("EngineeringUnits has unexpected wire value type %T", unitDV.Value.Value())
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
