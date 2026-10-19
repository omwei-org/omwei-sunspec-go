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
`Rounded`, and errors. Current tests use synthetic values and a synthetic
unit allowlist; they do not claim a live OPC UA server or physical-device test.

The adapter intentionally has no dependency on a particular OPC UA client
library. This keeps protocol client lifecycle/security policy and Go toolchain
requirements separate from semantic normalization. A live integration can
compose it with an OPC UA client after selecting and validating that client's
supported Go version and metadata APIs.

## Development

Requires Go 1.23 or newer.

```sh
go mod tidy
go vet ./...
go test ./...
```

## Limits

- No live OPC UA server integration or hardware-in-the-loop test.
- No automatic conversion between units; the core mapping must explicitly
  define any conversion.
- `SourceSample` is not proof of device authenticity, freshness, or atomicity.
- No OPC UA/SunSpec interoperability or conformance claim is made by these
  synthetic unit tests.
