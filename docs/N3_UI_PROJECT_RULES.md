# N3-UI Project Rules

## Project Identity

- Project name: `n3-ui`
- Source baseline: `n2-ui`
- Development model: source-level fork based on `n2-ui`

## Runtime Compatibility

`n3-ui` must continue to run as an `x-ui` compatible panel. Existing user environments, service names, commands, paths, database files, and API paths must remain compatible.

## Must Keep Unchanged

- Service name: `x-ui.service`
- CLI command name: `x-ui`
- Install/runtime directory family: `/usr/local/x-ui`
- Configuration directory: `/etc/x-ui`
- Database path: `/etc/x-ui/x-ui.db`
- Database schema compatibility with existing `x-ui` installations
- API path prefix: `/xui/`
- Go module name: `module x-ui`
- Existing frontend internal `x-ui` naming and variables unless a narrow feature requires otherwise

## Allowed Development Areas

- New product features
- Protocol support fixes and enhancements
- Certificate system improvements
- Web panel usability enhancements
- CLI command enhancements that preserve `x-ui` command compatibility

## Compatibility Rules

- Do not rename `x-ui.service`.
- Do not rename the installed binary or CLI command away from `x-ui`.
- Do not move the database away from `/etc/x-ui/x-ui.db`.
- Do not introduce database schema changes without explicit migration and rollback planning.
- Do not change existing `/xui/` API paths unless backward compatible aliases remain.
- Do not mass-replace `x-ui` strings only for branding.
- Document every compatibility-sensitive change before implementation.

