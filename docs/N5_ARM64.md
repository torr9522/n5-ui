# N5 ARM64 Support

## Architecture Contract

N5 uses one source tree and one installer for both supported Linux architectures:

| Kernel/userspace names | Normalized architecture | Panel package | Runtime asset |
|---|---|---|---|
| `x86_64`, `x64`, `amd64` | `amd64` | `x-ui-linux-amd64.tar.gz` | `Xray-linux-64.zip` |
| `aarch64`, `arm64` | `arm64` | `x-ui-linux-arm64.tar.gz` | `Xray-linux-arm64.zip` |

Other architectures fail before installation. ARM64 never falls back to the AMD64 runtime.

## Panel Build

The panel uses `go-sqlite3`, so formal Linux packages require CGO. In the Debian 11 ARM64 builder:

```bash
scripts/build-panel.sh arm64 /tmp/x-ui /opt/go1.26.0/bin/go
```

The build script requires Go 1.26.0, a clean tracked source tree, and `CGO_ENABLED=1`. ARM support does not change the DB schema, routing semantics, API behavior, UI behavior, service name, or installation paths.

## Release Package

After creating the panel binary and Custom Xray runtime ZIP:

```bash
scripts/package-release.sh \
  arm64 \
  /tmp/x-ui \
  /tmp/Xray-linux-arm64.zip \
  /tmp/release
```

The package mirrors the existing AMD64 layout under a top-level `x-ui/` directory. It embeds only the matching runtime ZIP. The script validates the panel architecture and required runtime ZIP members before packaging.

The same packaging path supports `amd64` while preserving the historical asset names. It never rebuilds or overwrites the Golden AMD64 runtime.

## Runtime Updater

The panel updater selects the runtime asset from `runtime.GOARCH`:

```text
amd64 -> Xray-linux-64.zip
arm64 -> Xray-linux-arm64.zip
```

Local package assets are preferred. The default remote base is GitHub `releases/latest/download`, allowing the same installer to remain valid after a new dual-architecture release without rewriting a historical tag. `XUI_RELEASES_BASE` remains available for a controlled release mirror or alternate release host. AMD64 keeps its historical asset name and immutable Golden payload.

## Reconstructed Provenance

The historical Custom Xray commit was not found. See `docs/CUSTOM_XRAY_PROVENANCE.md` for the exact upstream base, recovered delta, canonical patch, and Golden AMD64 constraints. Validation results and final candidate checksums are appended to this document after the formal build and installation tests.
