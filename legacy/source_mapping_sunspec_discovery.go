package omwei

import (
	"context"
	"fmt"
	"math"
	"reflect"
)

// SunSpecModelHeader is a discovered model header. StartAddress points to the
// first data register, after the two-register (model ID, model length) header.
type SunSpecModelHeader struct {
	ModelID      uint16
	ModelLength  uint16
	HeaderAddress uint16
	StartAddress uint16
}

// SunSpecModelDiscoveryConfig bounds a sequential walk of a SunSpec model
// chain. FirstHeaderAddress must point to the first model ID register, not the
// SunSpec signature/common-block registers. MaxModels prevents an unbounded
// walk when a device returns malformed headers.
type SunSpecModelDiscoveryConfig struct {
	FirstHeaderAddress uint16
	MaxModels           uint16
}

// DiscoverSunSpecModel103 walks model headers until it finds Model 103 or the
// SunSpec end marker (0xffff). Header ID and length are read together. The
// returned StartAddress is suitable for SunSpecModel103ModbusAdapter.
func DiscoverSunSpecModel103(
	ctx context.Context,
	client SunSpecModel103RegisterClient,
	unitID uint8,
	registerType SunSpecRegisterType,
	config SunSpecModelDiscoveryConfig,
) (SunSpecModelHeader, error) {
	if client == nil || (isTypedNilSunSpecClient(client)) {
		return SunSpecModelHeader{}, ErrSourceIdentityMismatch
	}
	if registerType != SunSpecHoldingRegister && registerType != SunSpecInputRegister {
		return SunSpecModelHeader{}, ErrInvalidSourceMapping
	}
	if config.MaxModels == 0 {
		return SunSpecModelHeader{}, ErrInvalidSourceMapping
	}
	if ctx == nil {
		ctx = context.Background()
	}
	address := uint32(config.FirstHeaderAddress)
	for i := uint16(0); i < config.MaxModels; i++ {
		if address+1 > math.MaxUint16 {
			return SunSpecModelHeader{}, ErrInvalidSourceMapping
		}
		var registers []uint16
		var err error
		if registerType == SunSpecInputRegister {
			registers, err = client.ReadInputRegisters(ctx, unitID, uint16(address), 2)
		} else {
			registers, err = client.ReadHoldingRegisters(ctx, unitID, uint16(address), 2)
		}
		if err != nil {
			return SunSpecModelHeader{}, err
		}
		if len(registers) != 2 {
			return SunSpecModelHeader{}, fmt.Errorf("%w: SunSpec header read returned %d registers, want 2", ErrInvalidSunSpecValue, len(registers))
		}
		modelID, length := registers[0], registers[1]
		if modelID == 0xffff {
			return SunSpecModelHeader{}, fmt.Errorf("%w: Model 103 not found before SunSpec end marker", ErrInvalidSourceMapping)
		}
		if length == 0 {
			return SunSpecModelHeader{}, fmt.Errorf("%w: model %d has zero length", ErrInvalidSunSpecValue, modelID)
		}
		dataStart := address + 2
		lastData := dataStart + uint32(length) - 1
		nextHeader := dataStart + uint32(length)
		if lastData > math.MaxUint16 {
			return SunSpecModelHeader{}, ErrInvalidSourceMapping
		}
		if modelID == 103 {
			if length < 14 {
				return SunSpecModelHeader{}, fmt.Errorf("%w: Model 103 length %d is shorter than 14 registers", ErrInvalidSunSpecValue, length)
			}
			return SunSpecModelHeader{
				ModelID: modelID, ModelLength: length,
				HeaderAddress: uint16(address), StartAddress: uint16(dataStart),
			}, nil
		}
		if nextHeader > math.MaxUint16 {
			return SunSpecModelHeader{}, ErrInvalidSourceMapping
		}
		address = nextHeader
	}
	return SunSpecModelHeader{}, fmt.Errorf("%w: Model 103 not found within %d model headers", ErrInvalidSourceMapping, config.MaxModels)
}

func isTypedNilSunSpecClient(client SunSpecModel103RegisterClient) bool {
	v := reflect.ValueOf(client)
	return v.IsValid() && v.Kind() == reflect.Ptr && v.IsNil()
}

 
// DiscoverSunSpecModel103Adapter performs bounded model discovery and returns
// an adapter configured with the discovered model ID, length, and data address.
// Source identity must still be provisioned independently of device-reported
// registers; discovery does not establish cryptographic device identity.
func DiscoverSunSpecModel103Adapter(
	ctx context.Context,
	client SunSpecModel103RegisterClient,
	source SourceIdentity,
	unitID uint8,
	registerType SunSpecRegisterType,
	config SunSpecModelDiscoveryConfig,
) (*SunSpecModel103ModbusAdapter, error) {
	header, err := DiscoverSunSpecModel103(ctx, client, unitID, registerType, config)
	if err != nil {
		return nil, err
	}
	if source.Protocol != "sunspec-modbus" || source.Device == "" ||
		source.Point != "model:103:W;scale-factor:W_SF" {
		return nil, ErrSourceIdentityMismatch
	}
	return &SunSpecModel103ModbusAdapter{
		Client: client, Source: source, UnitID: unitID, RegisterType: registerType,
		ModelID: header.ModelID, ModelLength: header.ModelLength,
		ModelStartAddress: header.StartAddress, Context: ctx,
	}, nil
}
