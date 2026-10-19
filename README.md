# omwei-sunspec-go

Experimental SunSpec adapter for [omwei-iot-go](https://github.com/omwei-org/omwei-iot-go).

## Scope

This companion module is the home for the optional SunSpec protocol adapter and
its transport-level integration tests. The core module remains Go 1.22 and
standard-library-only; this companion targets Go 1.23 because
`github.com/otfabric/go-sunspec v0.6.0` declares Go 1.23.

The planned adapter will normalize supported decoded SunSpec points into
`omwei-iot-go` `SourceSample` values and use exact decimal arithmetic for
scale-factor handling. The adapter must fail closed for unimplemented points,
unexpected raw-value types, and missing required scale factors. Float-valued
upstream convenience fields are for test cross-checks only, never the value path.

## Current status

This repository is an experimental companion scaffold. Until the adapter and
its tests are implemented, it does not provide a usable point-to-atom adapter.
Tests against synthetic inputs or a localhost simulator are not hardware-in-loop
evidence and do not establish vendor certification, conformance, device identity,
freshness, or atomicity of device updates. No physical device has been tested.

## Development

Requires Go 1.23 or newer.

```sh
go mod tidy
go vet ./...
go test ./...
```

The CI workflow also runs the core repository's transport-neutral source-mapping
interoperability vectors at the pinned core revision. Those vectors validate the
core mapping contract, not this adapter's conformance.

## Dependencies

Versions are pinned in `go.mod`. The core dependency currently uses a
commit-derived pseudo-version because a released version for the required
source-mapping API has not been verified. Replace it with a tagged release when
one is available and verified.
