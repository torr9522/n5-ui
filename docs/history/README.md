# N5-UI Public Development History

This directory is the public, redacted handoff archive for the N5-UI path from
the early fork state through `v0.3.1` Stable.

It is intentionally a summary archive. Raw UAT logs, databases, browser traces,
server addresses, credentials, tokens, and private runtime evidence are not
published.

## Reading Order

1. `00_PROJECT_TIMELINE.md`
2. `01_FIX_01_10.md`
3. `02_P1_HARDENING.md`
4. `03_P2_HARDENING.md`
5. `04_RELEASE_CANDIDATES.md`
6. `05_TEST_EVIDENCE.md`
7. `06_RELEASE_V0.2.0.md`
8. `07_ARCHITECTURE_DECISIONS.md`
9. `08_REJECTED_AND_DEFERRED.md`
10. `09_CHECKPOINT_MAP.md`
11. `10_CONTINUATION_GUIDE.md`
12. `11_RELEASE_PROCESS.md`
13. `12_SECURITY_REDACTION.md`
14. `13_SOURCE_INVENTORY.md`
15. `14_NEW_CODEX_RECOVERY_TEST.md`
16. `N5_V0.3.0_ARM64_STABLE.md`
17. `N5_V0.3.1_STABLE.md`

Machine-readable checkpoint data is in `checkpoints.json`.

## Stable Summary

- Current Stable: `v0.3.1`
- Release commit: `d0e782fc31ff726baf0556e8add9333b14522527`
- Release tree: `0e54b2830773bcf3a32c4fc81e6c004812ce81e0`
- Xray runtime: Custom `26.5.3`, amd64 and arm64
- ARM64 acceptance: independent Debian 12 clean install, external TCP/UDP,
  Access IP, updater, browser, DB, routing, and real reboot passed.
- v0.3.1 acceptance: AMD64 clean RC/Stable source and package installs,
  restored historical DB tolerance, REALITY export, routing, Access IP, and
  real reboot passed; dual-architecture packages and frozen runtimes passed.
- Upgrade policy: clean install supported; in-place upgrade not guaranteed.

## Public Checkpoint Tags

The long-term checkpoint tags listed in `09_CHECKPOINT_MAP.md` are safe to push
and fetch from GitHub one by one. Legacy duplicate local tags are documented but
do not need to be public for project reconstruction.
