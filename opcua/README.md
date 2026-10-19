# OPC UA to OMWEI adapter

Experimental, transport-neutral OPC UA value adapter for
[omwei-iot-go](https://github.com/omwei-org/omwei-iot-go).

## Scope

`SourceSampleFromValue` converts a numeric value and an explicitly resolved
engineering unit into OMWEI's `SourceSample`. It does not implement OPC UA
client transport or browse nodes. It rejects missing units, unsupported value
types, non-finite floats, and values exceeding the core's plain-decimal
precision contract.

`ResolveEngineeringUnit` resolves an OPC UA `EUInformation` pair
(`NamespaceURI`, numeric `UnitID`) through an explicit, provisioned allowlist.
It intentionally ignores localized `DisplayName` and `Description`; unknown
namespace/unit pairs fail closed. The allowlist must be based on the applicable
OPC UA/UN-CEFACT unit registry and deployment configuration. No numeric unit
ID is hard-coded or inferred by this adapter.

The caller must bind the value and engineering-unit metadata to the same
successful OPC UA read context. Configure `SourceIdentity.Point` with a stable
node identity (prefer namespace URI plus node identifier), not a display name.
This adapter does not authenticate device identity or establish
freshness/atomicity.

## Cross-protocol demonstration

The intended comparison feeds both this adapter and the existing SunSpec
companion adapter into the same core `SourceMapping`. For equivalent physical
values and matching source-unit configuration, compare canonical atom bytes,
`Rounded`, and errors. The repository includes a Go 1.25 isolated integration
module under `integration/opcua-live`: it starts a loopback OPC UA server, reads
a numeric value and an `EUInformation` ExtensionObject in one OPC UA Read
request, resolves the unit, and normalizes a `SourceSample`. Its values and
unit registry are synthetic; no physical device is exercised.

The production adapter intentionally has no dependency on a particular OPC UA
client library. This keeps protocol client lifecycle/security policy and Go
toolchain requirements separate from semantic normalization. The isolated
integration module currently validates composition with `github.com/otfabric/go-opcua v1.2.0`; that module requires Go 1.25 and is not imported by the Go 1.23 production module.

## Development

Requires Go 1.23 or newer.

```sh
go mod tidy
go vet ./...
go test ./...
```

## Limits

- Live loopback OPC UA client/server software test exists; no physical-device or hardware-in-the-loop test.
- No automatic conversion between units; the core mapping must explicitly
  define any conversion.
- `SourceSample` is not proof of device authenticity, freshness, or atomicity.
- No OPC UA/SunSpec interoperability or conformance claim is made by these
  synthetic unit tests.
