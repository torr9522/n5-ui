# N5 Custom Xray 26.5.3 Provenance

## Status

The historical fork URL, historical custom commit, and original build command were not recovered. The provenance in this repository is therefore explicitly reconstructed, not claimed as the original development history.

The recovered source is the `xray-core/` snapshot first imported into N5 and retained unchanged through N5 v0.2.0 and `main`. Its Git tree is:

```text
94bed1effd3c690b45b1b2978566d710b08723d6
```

The reconstructed local commit created during the ARM64 work is recorded in `docs/N5_ARM64.md`. That commit is an evidence artifact; the canonical, portable provenance is the upstream commit plus patch in this repository.

## Upstream Base

```text
Repository: https://github.com/XTLS/Xray-core.git
Version: v26.5.3
Commit: 228f1e13aa22739b0d6b9adbdb2b600f1e2018e1
```

## Canonical Delta

```text
Patch: runtime/custom-xray/patches/N5-CUSTOM-XRAY-26.5.3.patch
Patch SHA256: 7cc712282480784706818a3dc5ab52020fb133c83250921228497fc51fc02b77
Changed paths: 17
Functional Go files: 15
```

All 15 modified Go files are `FUNCTIONAL_CUSTOM`. They add and propagate `AccessMessage.InboundTag` and render it as `[inbound-tag]` in access logs:

```text
common/log/access.go
common/mux/server.go
proxy/dokodemo/dokodemo.go
proxy/http/server.go
proxy/hysteria/server.go
proxy/shadowsocks/server.go
proxy/shadowsocks_2022/inbound.go
proxy/shadowsocks_2022/inbound_multi.go
proxy/shadowsocks_2022/inbound_relay.go
proxy/socks/server.go
proxy/trojan/server.go
proxy/tun/handler.go
proxy/vless/inbound/inbound.go
proxy/vmess/inbound/inbound.go
proxy/wireguard/server.go
```

The other two changes are deletions from the recovered snapshot:

| Path | Classification | Runtime impact |
|---|---|---|
| `common/buf/data/test_MultiBufferReadAllToByte.dat` | `GENERATED` / test fixture | None in the runtime binary |
| `main/distro/debug/debug.go` | `BUILD_ONLY` | Removes the optional debug distro entry point; the normal `./main` build is unaffected |

There are no dependency changes: `go.mod` and `go.sum` match upstream v26.5.3.

## Functional Contract

The custom data path is:

```text
client connection
  -> inbound handler reads session.InboundFromContext(ctx).Tag
  -> AccessMessage.InboundTag
  -> AccessMessage.String() appends [inbound-tag]
  -> /var/log/xray/access.log
  -> N5 AccessIPService parser
  -> access_ip_records
  -> /xui/access-ip/list and Access IP UI
```

N5 creates inbound tags as `inbound-<port>`. The panel parser intentionally extracts the port from that exact marker. The mux server copies the outer inbound tag into each nested accepted access message, preventing multiplexed streams from losing the panel's inbound identity.

## Golden AMD64 Runtime

The existing AMD64 runtime is immutable and is not rebuilt or replaced by the ARM64 work:

```text
Xray binary SHA256: 128f9c34811ee74b3770eef7010d011e3946e85dfab28f2ed1804e380461b05e
Runtime ZIP SHA256: 98e1cfe7b8a85d833edcd5101530f2d67609505d832eac15ac2a236f3374bbbe
Go: 1.26.0
CGO: enabled
VCS base: 228f1e13aa22739b0d6b9adbdb2b600f1e2018e1
VCS modified: true
```

The official upstream ARM64 asset is not equivalent to this custom source and must not be used as an N5 runtime.

