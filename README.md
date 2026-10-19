# omwei-sunspec-go

Experimental protocol adapters for [omwei-iot-go](https://github.com/omwei-org/omwei-iot-go).

## SunSpec adapter

This module adapts supported decoded SunSpec Model 103 active-power points into OMWEI's transport-neutral `SourceSample`, using `github.com/otfabric/go-sunspec v0.6.0`.

The adapter requires a provisioned source identity and an implemented `W` point of type `int16`, with a resolved scale factor. It computes the scaled decimal using the core's exact-integer decimal helper. The upstream `ScaledValue` float is used only for a strict consistency cross-check; it never supplies the value sent to the mapper. Unexpected raw types, unimplemented points, missing scale factors, and mismatched float cross-checks fail closed.

## OPC UA semantic adapter

The `opcua` package converts a numeric OPC UA value and an explicitly resolved engineering unit into the same `SourceSample` contract. It is deliberately transport-neutral and does not implement an OPC UA client, browse node metadata, infer units from display names, authenticate a server/device, or establish sample freshness. The caller must bind the value and unit to the same successful OPC UA read context.

The current released `otfabric/go-opcua` v1.2.0 requires Go 1.25, while this production companion uses Go 1.23. The dependency is therefore isolated in `integration/opcua-live` and is not imported by the production adapter package.

## Cross-protocol demonstration

`cross_protocol_parity_test.go` feeds equivalent synthetic values through both protocol adapters and the same core mapping contract. It checks matching atom values, `Rounded` results, and rejection of negative active power.

The isolated `integration/opcua-live` module starts a loopback OPC UA server/client, reads a synthetic numeric value and `EUInformation` in one OPC UA Read service request, then compares its normalized canonical result with a `go-sunspec.DecodedPoint` fixture passed through the existing SunSpec adapter. The OPC UA transport path is live software communication; the SunSpec side starts at the decoded-output boundary and does not perform a live Modbus acquisition. Both values are synthetic.

For exact prerequisites, commands, expected output, and explicit scope limitations, see [the reproducible demonstration guide](integration/opcua-live/README.md).

The purpose is to demonstrate that protocol-specific inputs can converge on a common canonical representation through OMWEI's semantic mapping. It does not claim both inputs describe the same physical measurement. It also does not claim live SunSpec device interoperability, vendor certification, EABC conformance, cryptographic device identity, register-map correctness for a particular vendor, freshness, or atomicity of device updates.

## Development

Requires Go 1.23 or newer.

```sh
go mod tidy
go vet ./...
go test ./...
```

CI runs the core repository's transport-neutral source-mapping interoperability vectors at a pinned core revision. The separate `integration/opcua-live` module uses Go 1.25 and tests the live OPC UA client/server protocol path plus the cross-protocol comparison against a decoded SunSpec fixture; it does not raise the production module's Go 1.23 requirement. These tests validate software behavior, not physical-device integration or protocol-adapter conformance.

## Dependencies

The core dependency is pinned to released `github.com/omwei-org/omwei-iot-go v0.2.0`. SunSpec support uses `github.com/otfabric/go-sunspec v0.6.0`; indirect dependencies are pinned in `go.mod`.
