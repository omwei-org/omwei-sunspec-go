package omwei

import (
	"context"
	"errors"
	"testing"
)

type fakeModel103ModbusClient struct {
	gotCtx context.Context
	gotUnit uint8
	gotAddr uint16
	gotQuantity uint16
	gotInput bool
	registers []uint16
	err error
}

func (f *fakeModel103ModbusClient) ReadHoldingRegisters(ctx context.Context, unit uint8, addr uint16, quantity uint16) ([]uint16, error) {
	f.gotCtx, f.gotUnit, f.gotAddr, f.gotQuantity, f.gotInput = ctx, unit, addr, quantity, false
	return f.registers, f.err
}

func (f *fakeModel103ModbusClient) ReadInputRegisters(ctx context.Context, unit uint8, addr uint16, quantity uint16) ([]uint16, error) {
	f.gotCtx, f.gotUnit, f.gotAddr, f.gotQuantity, f.gotInput = ctx, unit, addr, quantity, true
	return f.registers, f.err
}

func validModel103Adapter(client SunSpecModel103RegisterClient) *SunSpecModel103ModbusAdapter {
	return &SunSpecModel103ModbusAdapter{
		Client: client,
		Source: SourceIdentity{Protocol: "sunspec-modbus", Device: "inverter-1", Point: "model:103:W;scale-factor:W_SF"},
		UnitID: 7, RegisterType: SunSpecInputRegister, ModelID: 103, ModelLength: 14, ModelStartAddress: 40010,
	}
}

func TestSunSpecModel103ModbusAdapterReadsWAndScaleFactorTogether(t *testing.T) {
	client := &fakeModel103ModbusClient{registers: []uint16{1234, 0xffff}}
	adapter := validModel103Adapter(client)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	adapter.Context = ctx

	source, modelID, w, sf, snapshot, err := adapter.ReadModel103ActivePowerRegisters()
	if err != nil {
		t.Fatal(err)
	}
	if source != adapter.Source || modelID != 103 || w != 1234 || sf != 0xffff || snapshot == "" {
		t.Fatalf("unexpected snapshot: source=%+v model=%d W=%d SF=%d id=%q", source, modelID, w, sf, snapshot)
	}
	if client.gotCtx != ctx || client.gotUnit != 7 || client.gotAddr != 40022 || client.gotQuantity != 2 || !client.gotInput {
		t.Fatalf("wrong Modbus request: ctx=%v unit=%d addr=%d quantity=%d input-registers=%v", client.gotCtx, client.gotUnit, client.gotAddr, client.gotQuantity, client.gotInput)
	}
}

func TestSunSpecModel103ModbusAdapterRejectsBadModelAndShortRead(t *testing.T) {
	client := &fakeModel103ModbusClient{registers: []uint16{1}}
	adapter := validModel103Adapter(client)
	adapter.ModelID = 102
	if _, _, _, _, _, err := adapter.ReadModel103ActivePowerRegisters(); !errors.Is(err, ErrInvalidSourceMapping) {
		t.Fatalf("wrong model ID: got %v", err)
	}
	adapter.ModelID = 103
	adapter.ModelLength = 13
	if _, _, _, _, _, err := adapter.ReadModel103ActivePowerRegisters(); !errors.Is(err, ErrInvalidSourceMapping) {
		t.Fatalf("short model: got %v", err)
	}
	adapter.ModelLength = 14
	if _, _, _, _, _, err := adapter.ReadModel103ActivePowerRegisters(); !errors.Is(err, ErrInvalidSunSpecValue) {
		t.Fatalf("short register read: got %v", err)
	}
}

func TestSunSpecModel103ModbusAdapterRejectsSentinelAndPropagatesTransportError(t *testing.T) {
	client := &fakeModel103ModbusClient{registers: []uint16{0x8000, 0}}
	adapter := validModel103Adapter(client)
	if _, _, _, _, _, err := adapter.ReadModel103ActivePowerRegisters(); !errors.Is(err, ErrInvalidSunSpecValue) {
		t.Fatalf("sentinel: got %v", err)
	}
	transportErr := errors.New("modbus transport unavailable")
	client.err = transportErr
	if _, _, _, _, _, err := adapter.ReadModel103ActivePowerRegisters(); !errors.Is(err, transportErr) {
		t.Fatalf("transport error: got %v", err)
	}
}

func TestSunSpecModel103ModbusAdapterRejectsAddressOverflowAndTypedNil(t *testing.T) {
	adapter := validModel103Adapter(&fakeModel103ModbusClient{registers: []uint16{1, 0}})
	adapter.ModelStartAddress = 65530
	adapter.ModelLength = 14
	if _, _, _, _, _, err := adapter.ReadModel103ActivePowerRegisters(); !errors.Is(err, ErrInvalidSourceMapping) {
		t.Fatalf("address overflow: got %v", err)
	}
	var typedNil *fakeModel103ModbusClient
	adapter = validModel103Adapter(typedNil)
	if _, _, _, _, _, err := adapter.ReadModel103ActivePowerRegisters(); !errors.Is(err, ErrSourceIdentityMismatch) {
		t.Fatalf("typed nil client: got %v", err)
	}
}
