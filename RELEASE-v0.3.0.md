# N5-UI v0.3.0 Stable

## Native ARM64 Support

- The panel, installer, updater, and release packages support Linux amd64 and
  arm64 from one source tree.
- ARM64 uses a native Custom Xray 26.5.3 built from the same recovered N5 custom
  delta as the AMD64 Golden runtime.
- The Custom Xray upstream base, canonical patch, reconstructed source identity,
  build recipe, and architecture mapping are documented in the repository.

## Runtime Assets

```text
amd64 -> Xray-linux-64.zip
arm64 -> Xray-linux-arm64.zip
```

The historical AMD64 Golden runtime is unchanged. ARM64 reuses the fixed N5
`geoip.dat` and `geosite.dat` files.

## Installation

The same installer automatically recognizes amd64 and arm64:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/torr9522/n5-ui/main/install.sh)
```

Source and package installation modes are supported. Clean installation remains
recommended; this release does not expand the existing in-place upgrade promise.

## Compatibility

The architecture work does not change the DB schema, routing semantics, API,
web UI behavior, service name, install paths, subscriptions, or port-limit
semantics.
