package modbusadapter

import (
	"context"
	"testing"

	omwei "github.com/omwei-org/omwei-sunspec-go/legacy"
	"github.com/otfabric/go-modbus"
)

// Compile-time contract: the pinned upstream client's public methods must
// remain compatible with the transport-neutral EABC adapter interface.
var _ omwei.SunSpecModel103RegisterClient = (*modbus.Client)(nil)

func TestProductionClientFitsAdapterContract(t *testing.T) {
	var client *modbus.Client
	// Compile-check the method signatures without opening a socket.
	var readInput func(context.Context, uint8, uint16, uint16) ([]uint16, error) = client.ReadInputRegisters
	var readHolding func(context.Context, uint8, uint16, uint16) ([]uint16, error) = client.ReadHoldingRegisters
	_, _ = readInput, readHolding
}
