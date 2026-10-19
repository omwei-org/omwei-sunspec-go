package modbusadapter

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	omwei "github.com/omwei-org/omwei-sunspec-go/legacy"
	"github.com/otfabric/go-modbus"
)

// realClientSimulator is intentionally small but speaks ordinary Modbus TCP.
// Unlike the root package's minimal client, this test uses the pinned upstream
// production client for all requests and responses.
type realClientSimulator struct {
	listener net.Listener
	mu sync.Mutex
	serveErr error
	faultMode string
	wg sync.WaitGroup
}

func startRealClientSimulator(t *testing.T) *realClientSimulator {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &realClientSimulator{listener: listener}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				defer conn.Close()
				if err := s.serveConnection(conn); err != nil {
					s.mu.Lock()
					if s.serveErr == nil {
						s.serveErr = err
					}
					s.mu.Unlock()
				}
			}()
		}
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		s.wg.Wait()
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.serveErr != nil {
			t.Errorf("Modbus TCP simulator: %v", s.serveErr)
		}
	})
	return s
}

func (s *realClientSimulator) address() string { return s.listener.Addr().String() }

func (s *realClientSimulator) setFaultMode(mode string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.faultMode = mode
}

func (s *realClientSimulator) serveConnection(conn net.Conn) error {
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	for {
		header := make([]byte, 7)
		if _, err := io.ReadFull(conn, header); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			return err
		}
		transactionID := binary.BigEndian.Uint16(header[0:2])
		protocolID := binary.BigEndian.Uint16(header[2:4])
		length := binary.BigEndian.Uint16(header[4:6])
		unitID := header[6]
		if protocolID != 0 || length < 2 || length > 254 || unitID != 7 {
			return fmt.Errorf("invalid request header protocol=%d length=%d unit=%d", protocolID, length, unitID)
		}
		pdu := make([]byte, int(length)-1)
		if _, err := io.ReadFull(conn, pdu); err != nil {
			return err
		}
		if len(pdu) != 5 || (pdu[0] != 3 && pdu[0] != 4) {
			return fmt.Errorf("unexpected PDU function/length: %v", pdu)
		}
		function := pdu[0]
		start := binary.BigEndian.Uint16(pdu[1:3])
		quantity := binary.BigEndian.Uint16(pdu[3:5])
		if quantity == 0 || quantity > 125 {
			return fmt.Errorf("invalid read quantity %d", quantity)
		}

		s.mu.Lock()
		faultMode := s.faultMode
		s.mu.Unlock()
		if faultMode == "exception" {
			// A valid Modbus exception PDU: function | 0x80, exception code 2.
			response := make([]byte, 9)
			binary.BigEndian.PutUint16(response[0:2], transactionID)
			binary.BigEndian.PutUint16(response[2:4], 0)
			binary.BigEndian.PutUint16(response[4:6], 3)
			response[6] = unitID
			response[7] = function | 0x80
			response[8] = 2
			if _, err := conn.Write(response); err != nil {
				return err
			}
			continue
		}

		response := make([]byte, 9+int(quantity)*2)
		binary.BigEndian.PutUint16(response[0:2], transactionID)
		binary.BigEndian.PutUint16(response[2:4], 0)
		binary.BigEndian.PutUint16(response[4:6], uint16(3+quantity*2))
		response[6] = unitID
		response[7] = function
		response[8] = byte(quantity * 2)
		for i := uint16(0); i < quantity; i++ {
			var value uint16
			switch start + i {
			case 100:
				value = 103 // SunSpec Model ID
			case 101:
				value = 14 // model data length
			case 114:
				value = 1234 // W
			case 115:
				value = 0xffff // W_SF = -1
			}
			binary.BigEndian.PutUint16(response[9+int(i)*2:11+int(i)*2], value)
		}
		if _, err := conn.Write(response); err != nil {
			return err
		}
	}
}

func TestPinnedOtfabricClientOverLocalModbusTCPToCanonicalWattValue(t *testing.T) {
	simulator := startRealClientSimulator(t)
	client, err := modbus.New(modbus.Config{
		URL: "tcp://" + simulator.address(),
		Timeout: 2 * time.Second,
		DialTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("construct otfabric client: %v", err)
	}
	if err := client.Open(); err != nil {
		t.Fatalf("open otfabric client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Errorf("close otfabric client: %v", err)
		}
	}()

	source := omwei.SourceIdentity{
		Protocol: "sunspec-modbus",
		Device: "simulated-inverter-7",
		Point: "model:103:W;scale-factor:W_SF",
	}
	adapter, err := omwei.DiscoverSunSpecModel103Adapter(
		context.Background(), client, source, 7, omwei.SunSpecInputRegister,
		omwei.SunSpecModelDiscoveryConfig{FirstHeaderAddress: 100, MaxModels: 16},
	)
	if err != nil {
		t.Fatalf("discover Model 103 using production client: %v", err)
	}
	mapping := omwei.SourceMapping{
		Source: source, DescriptorID: 69,
		SourceUnit: "watt (after W_SF application)", CanonicalUnit: "watt",
		ScaleNumerator: 1, ScaleDenominator: 1,
		OffsetNumerator: 0, OffsetDenominator: 1,
	}
	result, err := omwei.ReadAndMapSunSpecModel103ActivePower(adapter, mapping)
	if err != nil {
		t.Fatalf("map Model 103 sample using production client: %v", err)
	}
	if result.Atom.CanonicalValue != 123 {
		t.Fatalf("want 123 W from raw W=1234 and W_SF=-1, got %d", result.Atom.CanonicalValue)
	}
	if client.LastObservedTransactionID() == 0 {
		t.Fatal("expected upstream client to observe a Modbus TCP transaction ID")
	}
}

// Test the pinned upstream client against a valid Modbus exception response.
// The TCP/MBAP frame is well-formed; the server rejects the requested function.
func TestPinnedOtfabricClientRejectsModbusException(t *testing.T) {
	simulator := startRealClientSimulator(t)
	simulator.setFaultMode("exception")

	client, err := modbus.New(modbus.Config{
		URL: "tcp://" + simulator.address(),
		Timeout: 2 * time.Second,
		DialTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("construct otfabric client: %v", err)
	}
	if err := client.Open(); err != nil {
		t.Fatalf("open otfabric client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Errorf("close otfabric client: %v", err)
		}
	}()

	_, err = client.ReadInputRegisters(context.Background(), 7, 114, 2)
	if err == nil {
		t.Fatal("expected upstream client to reject a valid Modbus exception response")
	}
}
