# Building N5 Custom Xray

## Required Environment

Formal N5 runtime candidates use:

```text
Operating system: Debian 11 arm64
glibc: 2.31
GCC: 10.2.1
Go: 1.26.0 linux/arm64
CGO_ENABLED: 1
GOOS: linux
```

The Go archive must come from `https://go.dev/dl/go1.26.0.linux-arm64.tar.gz`. Its checksum, as published by the Go download JSON endpoint, is:

```text
bd03b743eb6eb4193ea3c3fd3956546bf0e3ca5b7076c8226334afe6b75704cd
```

Do not silently use another Go release for a formal candidate.

## Reconstruct Source

From the N5 repository root:

```bash
runtime/custom-xray/reconstruct.sh /tmp/n5-custom-xray-26.5.3
```

The script checks out exact upstream commit `228f1e13...`, verifies the patch with `git apply --check`, applies it, and requires the resulting Git tree to equal `94bed1ef...`.

## Build

Run inside the Debian 11 builder with Go 1.26.0 available:

```bash
runtime/custom-xray/build.sh \
  /tmp/n5-custom-xray-26.5.3 \
  arm64 \
  /tmp/xray \
  /opt/go1.26.0/bin/go
```

The build script rejects unsupported architectures, a non-Go-1.26.0 toolchain, or a source tree other than the recovered N5 tree. For a freshly patched standalone checkout it computes the complete worktree tree through an isolated temporary Git index, so the source does not need an artificial provenance commit. It always builds with CGO enabled.

The same script accepts `amd64` for reproducibility experiments. It does not authorize replacing the immutable AMD64 Golden runtime.

## Package Runtime

Use the fixed geo files extracted from the AMD64 Golden runtime:

```bash
runtime/custom-xray/package-runtime.sh \
  /tmp/xray \
  /path/to/golden/geoip.dat \
  /path/to/golden/geosite.dat \
  /tmp/Xray-linux-arm64.zip
```

The packaging script refuses geo files whose SHA256 values differ from the Golden N5 assets. The ZIP contains exactly the runtime payload names used by the updater and installer: `xray`, `geoip.dat`, and `geosite.dat`.

## Audit

For every candidate, record at least:

```bash
sha256sum xray Xray-linux-arm64.zip
file xray
readelf -h -l -V xray
ldd xray
go version -m xray
```

The result must be native AArch64, dynamically linked against no newer glibc ABI than the declared builder baseline, and report `CGO_ENABLED=1`.
