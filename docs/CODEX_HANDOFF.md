# N5-UI Development Handoff

This is the first document a new Codex agent should read after cloning the
public repository.

Start from a fresh clone:

```bash
git clone https://github.com/torr9522/n5-ui.git
cd n5-ui
git fetch --tags
```

Read these files in order:

1. `docs/N5_UI_PROJECT_RULES.md`
2. `docs/N5_UI_DEVELOPMENT_ARCHITECTURE.md`
3. `docs/N5_UI_SOURCE_TREE.md`
4. `docs/N5_UI_DATABASE.md`
5. `docs/N5_UI_RUNTIME.md`
6. `docs/N5_UI_SIMPLE_MODE_DESIGN.md`
7. `docs/DEVELOPMENT_SKILL_TREE.md`
8. `docs/history/README.md`
9. `docs/history/07_ARCHITECTURE_DECISIONS.md`
10. `docs/history/10_CONTINUATION_GUIDE.md`

## Current Stable Baseline

- Stable tag: `v0.3.0`
- Release commit: `d3485173b11b80c29acddfd800f377392d8b68ef`
- Release tree: `56e5d69876fef102f3d8289c141c52935d723f90`
- Annotated tag object: `aa0ee94880c8d085d889790547697ac3dcbbc9bd`
- Custom Xray baseline: `26.5.3`, amd64 and arm64
- AMD64 Golden Xray SHA256:
  `128f9c34811ee74b3770eef7010d011e3946e85dfab28f2ed1804e380461b05e`
- ARM64 Custom Xray SHA256:
  `2f59c045ff47d588edd16738d9761fe706b129a373088b4a7db0b9a936dd41ed`
- Full Stable acceptance: `docs/history/N5_V0.3.0_ARM64_STABLE.md`

## Release Policy

- Clean Install: supported for the validated Stable path.
- In-place Upgrade: not currently guaranteed.
- Official Stable runtime packages: Linux amd64 and arm64 with N5 Custom Xray
  26.5.3.
- Formal builders: Debian 11, Go 1.26.0, CGO enabled.
- ARM64 clean install baseline: Debian 12, source and package modes accepted.

## Operating Principles

1. Continue development from current `main`; use `v0.3.0` to reproduce release
   code and assets.
2. Treat any deployment host as TEST/UAT only unless a separate source-of-truth
   migration is explicitly performed.
3. Change business source only in the development repository.
4. Never let a test server become the only Source of Truth.
5. For major work: develop, test, create a local commit, create a checkpoint,
   then deploy to UAT.
6. Before the next Stable release, repeat clean install, browser, external TCP,
   external UDP, updater, reboot, runtime, and DB consistency gates.
7. Never move or recreate an existing Stable tag.
8. If a published Stable has a bug, fix it in a new version such as `v0.3.1`.

## Fast Checks

```bash
git checkout v0.3.0
git rev-list -n 1 v0.3.0
git cat-file -t v0.3.0
go test ./...
go build ./...
go vet ./...
git diff --check
go test -race ./web/service/n5/... ./web/controller/n5/...
```

Expected Stable commit:

```text
d3485173b11b80c29acddfd800f377392d8b68ef
```

For the full historical map, read `docs/history/09_CHECKPOINT_MAP.md` and
`docs/history/checkpoints.json`.
