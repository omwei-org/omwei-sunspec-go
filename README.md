# omwei-sunspec-go

Experimental SunSpec adapter for [omwei-iot-go](https://github.com/omwei-org/omwei-iot-go).

## Scope

This companion module adapts supported decoded SunSpec Model 103 active-power
points into OMWEI's transport-neutral `SourceSample`. The core module remains
Go 1.22 and standard-library-only; this companion targets Go 1.23 because
`github.com/otfabric/go-sunspec v0.6.0` declares Go 1.23.

The adapter requires a provisioned source identity and an implemented `W`
point of type `int16`, with a resolved scale factor. It computes the scaled
decimal using the core's exact-integer decimal helper. The upstream
`ScaledValue` float is used only for a strict consistency cross-check; it
never supplies the value sent to the mapper. Unexpected raw types, unimplemented
points, missing scale factors, and mismatched float cross-checks fail closed.

## Current status and limits

The point-to-sample adapter and synthetic unit tests are present. CI also runs
the core's transport-neutral source-mapping vectors, and the companion includes
a Modbus integration harness using a local simulator. These checks are not a
physical-device or hardware-in-the-loop test.

The OPC UA adapter and SunSpec-to-OPC-UA parity demonstration are not
implemented yet. The intended parity check is to feed equivalent, explicitly
identified measurements through protocol-specific adapters and the same
configured source-mapping contract, then compare the complete canonical atom
encoding and conversion metadata (including `Rounded` and error outcomes).
Parity must not be claimed until the OPC UA node identity, datatype, engineering
unit metadata, and value representation are validated and the shared vectors
pass for both paths.

Synthetic tests, core interoperability vectors, and a local Modbus simulator do
not establish vendor certification, EABC conformance, cryptographic device
identity, register-map correctness for a particular vendor, freshness, or
atomicity of device updates. No physical device has been tested. Negative active
power is currently rejected by descriptor 0x0045's range constraint
(0..100,000,000 W).

## Development

Requires Go 1.23 or newer.

```sh
go mod download
go vet ./...
go test ./...
```

CI runs the core repository's transport-neutral source-mapping interoperability
vectors at a pinned core revision. Those vectors validate the core mapping
contract; they do not by themselves establish adapter or cross-protocol
conformance.

## Dependencies

Dependencies are pinned in `go.mod`. The companion uses the released
`github.com/omwei-org/omwei-iot-go v0.2.0` core module and
`github.com/otfabric/go-sunspec v0.6.0`. Keep the dependency declarations and
this status section synchronized when versions or implemented scope change.
