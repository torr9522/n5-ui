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

The historical Custom Xray commit was not found. See `docs/CUSTOM_XRAY_PROVENANCE.md` for the exact upstream base, recovered delta, canonical patch, and Golden AMD64 constraints.

## Validation Baseline

The ARM64 candidate was built natively, without QEMU, in a Debian 11 ARM64
builder using glibc 2.31, GCC 10.2.1, Go 1.26.0, and `CGO_ENABLED=1`.
The implementation gates first passed on an Ubuntu 24.04 ARM64 development
host. Final Stable acceptance then passed on an independent clean Debian 12
ARM64 server:

- the canonical patch applies to the exact upstream commit and recreates the
  recovered Custom Xray source tree;
- repeated Custom Xray builds from the reconstructed commit are byte-identical;
- the Custom Xray and panel Go test suites pass;
- the Custom Xray binary is native AArch64, accepts the installed configuration,
  and preserves the ordinary and VMess mux `[inbound-tag]` access-log paths;
- source-mode and package-mode installations select only ARM64 panel and runtime
  assets;
- the panel, API, static assets, login flow, database integrity, port-limit
  timer, N5 nftables table, systemd restart, runtime updater, routing, and a real
  HTTP proxy request pass;
- the access log, parser, database record, API, and browser-rendered Access IP
  page agree on the generated `inbound-<port>` identity.
- GitHub-hosted RC and Stable source/package clean installs passed;
- external VMess, VLESS, Legacy Shadowsocks, and Trojan TCP passed;
- external Legacy Shadowsocks UDP DNS passed;
- the GitHub updater selected the ARM64 asset and fetched no AMD64 asset;
- both the RC package and official Stable package survived real OS reboots;
- public source, tag, assets, installer, and checksums passed disaster readback.

Final candidate checksums and complete command evidence belong in the release
manifest and implementation report, not this packaged source document. This
avoids making the package checksum depend on a checksum embedded inside the
package itself.

## Stable Acceptance

ARM64 became a published Stable architecture in `v0.3.0`. The complete
independent clean-server, external-client, updater, reboot, and public-readback
record is in `docs/history/N5_V0.3.0_ARM64_STABLE.md`.

The original historical Custom Xray commit remains unknown. The documented
reconstructed commit is a reproducible provenance identity, not a claim about
lost history. Clean install is supported; the release does not broaden the
existing in-place upgrade guarantee.
