# Reproducible cross-protocol semantic demonstration

This directory contains a software integration test for the OPC UA path and a
side-by-side comparison against the existing SunSpec adapter. It is designed
to be repeatable in CI without access to plant equipment.

## What the test runs

`TestCrossProtocolLiveOPCUAAgainstDecodedSunSpecFixture` performs these steps:

1. Starts an OPC UA server on loopback with a synthetic active-power value
   (`123.4`) and synthetic `EUInformation` metadata.
2. Connects an OPC UA client and reads the value and unit metadata in one
   OPC UA Read service request.
3. Resolves the unit using a test-only, explicitly configured
   `NamespaceURI + UnitID` mapping. The localized display label is not the key.
4. Normalizes the acquired OPC UA value through the OMWEI OPC UA adapter.
5. Constructs a `go-sunspec.DecodedPoint` fixture for Model 103 point `W`
   (raw `1234`, scale factor `-1`, decoded value `123.4 W`) and passes it
   through the existing SunSpec adapter.
6. Sends both `SourceSample` values through the same core
   `SourceMapping` contract and compares the resulting canonical atoms and
   `Rounded` flags.

Expected result: both paths produce the same canonical atom, with canonical
value `123` and `Rounded=true` for the illustrative input `123.4 W`.

The test is intentionally asymmetric at the transport boundary: OPC UA uses a
real client/server exchange over loopback, while SunSpec starts at the
`DecodedPoint` output boundary. It does **not** start a Modbus server or make a
live `go-sunspec` acquisition. The SunSpec fixture is shaped like the upstream
decoded output and is not evidence that a physical device produced that value.

## Prerequisites

- Go 1.25.x for this isolated integration module.
- Network loopback access and free local TCP ports 48491 and 48492.
- A checkout of this repository.

## Run it

From the repository root, first run the production module tests:

```sh
go test -race -count=1 ./...
```

Then run the live OPC UA and cross-protocol integration tests:

```sh
cd integration/opcua-live
go mod tidy
go test -race -count=1 -v ./...
```

The isolated module uses a local `replace` directive to test the checked-out
companion package, and keeps the Go 1.25 OPC UA client/server dependency out of
the production Go 1.23 module. CI runs the same integration test command.

## What this demonstrates

- The OPC UA client can read a numeric value and its `EUInformation` in one
  service request against the test server.
- Explicitly configured unit metadata can be resolved without trusting a
  localized display name.
- The existing SunSpec and OPC UA normalization adapters can feed the same
  core mapping contract.
- For the selected synthetic inputs and mapping, the canonical atom and
  rounding result agree.

## What this does not demonstrate

- A live Modbus read, live `go-sunspec` model discovery, or live SunSpec
  register decoding in this side-by-side test.
- That both protocol inputs represent the same physical measurement, device,
  timestamp, or acquisition event.
- Correctness of a vendor-specific register map, OPC UA namespace, unit
  registry, or deployed configuration.
- Device authentication, source freshness, update atomicity, network security,
  performance, hardware-in-the-loop behavior, or production readiness.
- OPC UA/SunSpec interoperability certification, vendor certification, or
  EABC/SLC conformance.

This is a reproducible semantic-pipeline software demonstration, not a
physical-source integration test. Any future claim about equivalent real
measurements requires a controlled setup that acquires both protocol values
from a known source and documents the source identity, acquisition timing,
unit definitions, and device configuration.
