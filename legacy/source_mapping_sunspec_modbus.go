package omwei

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"sync/atomic"
)

// SunSpecRegisterType selects the Modbus register table for the adapter.
type SunSpecRegisterType uint8

const (
	SunSpecHoldingRegister SunSpecRegisterType = iota
	SunSpecInputRegister
)

// SunSpecModel103RegisterClient is the narrow part of the Modbus client used
// by SunSpecModel103ModbusAdapter. *modbus.Client implements both methods.
type SunSpecModel103RegisterClient interface {
	ReadHoldingRegisters(ctx context.Context, unitID uint8, addr uint16, quantity uint16) ([]uint16, error)
	ReadInputRegisters(ctx context.Context, unitID uint8, addr uint16, quantity uint16) ([]uint16, error)
}

// SunSpecModel103ModbusAdapter reads Model 103 active power (W + W_SF) from
// a previously discovered SunSpec model. ModelStartAddress is the first data
// register after the two-register model header, as reported by sunspec.ModelHeader.StartAddress.
type SunSpecModel103ModbusAdapter struct {
	Client            SunSpecModel103RegisterClient
	Source            SourceIdentity
	UnitID            uint8
	RegisterType      SunSpecRegisterType
	ModelID           uint16
	ModelLength       uint16
	ModelStartAddress uint16
	Context           context.Context
}

var sunSpecModel103AcquisitionSequence atomic.Uint64

// ReadModel103ActivePowerRegisters reads W and W_SF in a single Modbus request.
// SnapshotID identifies this successful client-side acquisition only; it does
// not prove that the device updated both underlying values atomically.
func (a *SunSpecModel103ModbusAdapter) ReadModel103ActivePowerRegisters() (
	source SourceIdentity,
	modelID uint16,
	wRegister uint16,
	wSFRegister uint16,
	snapshotID string,
	err error,
) {
	if a == nil || a.Client == nil || (reflect.ValueOf(a.Client).Kind() == reflect.Ptr && reflect.ValueOf(a.Client).IsNil()) {
		return SourceIdentity{}, 0, 0, 0, "", ErrSourceIdentityMismatch
	}
	if a.ModelID != 103 || a.ModelLength < 14 {
		return SourceIdentity{}, a.ModelID, 0, 0, "", ErrInvalidSourceMapping
	}
	if a.Source.Protocol != "sunspec-modbus" ||
		a.Source.Device == "" ||
		a.Source.Point != "model:103:W;scale-factor:W_SF" {
		return SourceIdentity{}, a.ModelID, 0, 0, "", ErrSourceIdentityMismatch
	}
	if a.RegisterType != SunSpecHoldingRegister && a.RegisterType != SunSpecInputRegister {
		return SourceIdentity{}, a.ModelID, 0, 0, "", ErrInvalidSourceMapping
	}
	// Model 103 offsets are zero-based: W is data register 12 and W_SF is 13.
	// Check the full model span before calculating the requested address.
	last := uint32(a.ModelStartAddress) + uint32(a.ModelLength) - 1
	if last > math.MaxUint16 {
		return SourceIdentity{}, a.ModelID, 0, 0, "", ErrInvalidSourceMapping
	}
	address := uint32(a.ModelStartAddress) + 12
	if address+1 > math.MaxUint16 {
		return SourceIdentity{}, a.ModelID, 0, 0, "", ErrInvalidSourceMapping
	}
	ctx := a.Context
	if ctx == nil {
		ctx = context.Background()
	}
	var registers []uint16
	var readErr error
	if a.RegisterType == SunSpecInputRegister {
		registers, readErr = a.Client.ReadInputRegisters(ctx, a.UnitID, uint16(address), 2)
	} else {
		registers, readErr = a.Client.ReadHoldingRegisters(ctx, a.UnitID, uint16(address), 2)
	}
	if readErr != nil {
		return SourceIdentity{}, a.ModelID, 0, 0, "", readErr
	}
	if len(registers) != 2 {
		return SourceIdentity{}, a.ModelID, 0, 0, "", fmt.Errorf("%w: Model 103 read returned %d registers, want 2", ErrInvalidSunSpecValue, len(registers))
	}
	// Validate SunSpec signed-int16 sentinels before advertising a successful
	// snapshot to the downstream mapper.
	if _, _, decodeErr := DecodeSunSpecModel103Registers(registers[0], registers[1]); decodeErr != nil {
		return SourceIdentity{}, a.ModelID, 0, 0, "", decodeErr
	}
	sequence := sunSpecModel103AcquisitionSequence.Add(1)
	return a.Source, a.ModelID, registers[0], registers[1],
		fmt.Sprintf("sunspec103-unit-%d-addr-%d-read-%d", a.UnitID, address, sequence), nil
}
