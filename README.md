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
`ScaledValue` float is used only for a strict test/consistency cross-check; it
never supplies the value sent to the mapper. Unexpected raw types, unimplemented
points, missing scale factors, and mismatched float cross-checks fail closed.

## Current status and limits

The point-to-sample adapter and synthetic unit tests are present. The adapter
does not yet include a live device integration or hardware-in-the-loop testing.
Synthetic tests and core interoperability vectors do not establish vendor
certification, EABC conformance, cryptographic device identity, register-map
correctness for a particular vendor, freshness, or atomicity of device updates.
No physical device has been tested. Negative active power is currently rejected
by descriptor 0x0045's range constraint (0..100,000,000 W).

## Development

Requires Go 1.23 or newer.

```sh
go mod download
go vet ./...
go test ./...
```

CI also runs the core repository's transport-neutral source-mapping
interoperability vectors at a pinned core revision. Those vectors validate the
core mapping contract, not this adapter's conformance.

## Dependencies

Versions are pinned in `go.mod`. The core dependency currently uses a
commit-derived pseudo-version because a released version for the required
source-mapping API has not been verified. Replace it with a tagged release when
one is available and verified.
