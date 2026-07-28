# N3-UI Initialization Report

## Source Location

- n3-ui source path: `/root/n3-ui`
- Based on n2-ui source path: `/tmp/cert-audit-1785225391/n2-ui`
- n2-ui base commit: `654aa2c0a5cc02823922ad5a0f5d9d61a3c2461d`
- Initialization time: `2026-07-28T08:57:43Z`

## Compatibility Kept

- Service name remains `x-ui.service`.
- CLI command remains `x-ui`.
- Runtime install path family remains `/usr/local/x-ui`.
- Config directory remains `/etc/x-ui`.
- Database path remains `/etc/x-ui/x-ui.db`.
- Go module remains `module x-ui`.
- API prefix remains `/xui/`.
- Frontend internal `x-ui` naming was not mass-renamed.

## Added Project Files

- `docs/N3_UI_PROJECT_RULES.md`
- `docs/N3_UI_BASELINE.md`
- `N3_UI_INITIALIZATION_REPORT.md`
- `.gitignore`

## Baseline Notes

- n3-ui is initialized as a source-level fork of n2-ui.
- The copied source includes the current n2-ui working state, including the prior Trojan Web visibility restore.
- No certificate-system development was started.
- No backend protocol code was changed during initialization.
- No UI redesign or dependency upgrade was performed.

## Git Commit

- Commit message: `chore: initialize n3-ui based on n2-ui`
- Commit hash: run `git rev-parse HEAD` in `/root/n3-ui` to read the current immutable commit id.

## Verification Plan

- Check source file inventory.
- Run `go build` if the local Go environment allows it.
- Check Git status after commit.
- Confirm compatibility-sensitive identifiers remain unchanged.
