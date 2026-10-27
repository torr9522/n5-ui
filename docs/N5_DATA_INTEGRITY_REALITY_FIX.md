# N5 data integrity and REALITY export compatibility

## Scope

This maintenance change keeps the database schema, N5 routing semantics, and
Custom Xray runtime unchanged. It addresses two compatibility failures:

- Browser-generated REALITY VLESS links now select the first trimmed non-empty
  `serverNames` entry, matching backend URL, Base64, Clash, and Mihomo exports.
- Xray routing generation tolerates a restored traffic policy binding whose
  `inbound_id` no longer exists.

## Restored database behavior

Old database backups can contain rows in `n5_traffic_policy_bindings` that do
not resolve to an `inbounds.id`. The tables do not define a foreign key for this
relationship, and startup migrations intentionally do not delete user data.

During config generation, only `gorm.ErrRecordNotFound` from the inbound lookup
is treated as a historical dangling reference. The binding is skipped and a
warning records its binding ID, inbound ID, and policy ID. Other database errors
still abort config generation. No startup self-heal or automatic deletion is
performed.

The inbound deletion service already removes related bindings inside the same
database transaction before deleting the inbound. Regression coverage now also
checks multiple historical bindings, unrelated binding preservation, and that
an inbound update preserves its ID and binding.

Diagnostic SQL:

```sql
SELECT b.id AS binding_id, b.inbound_id, b.policy_id
FROM n5_traffic_policy_bindings AS b
LEFT JOIN inbounds AS i ON i.id = b.inbound_id
WHERE i.id IS NULL;
```

## Regression coverage

- `node scripts/test-reality-share.js`
- `go test ./...`
- `go vet ./...`

The database restore tests cover normal, single-orphan, multiple-orphan, mixed,
and all-orphan binding sets. The frontend harness covers normal, empty-first,
multiple-empty, all-empty, whitespace-only, address fallback, and input object
immutability cases.
