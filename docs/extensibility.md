---
layout: default
title: Extensibility
nav_order: 9
---

# Extensibility

Mind Palace is schema-first. Extend by adding curated artifacts.

---

## Adding Rooms

Create `.palace/rooms/<name>.jsonc`:

```jsonc
{
  "schemaVersion": "1.0.0",
  "kind": "palace/room",
  "name": "payments",
  "purpose": "Payment processing and billing",
  "entryPoints": [
    "src/payments/processor.ts",
    "src/payments/stripe.ts"
  ],
  "includeGlobs": ["src/payments/**"],
  "excludeGlobs": ["**/*.test.ts"]
}
```

### Guidelines

- Entry points should be the "start here" files
- Use forward slashes in paths
- Keep Rooms focused (one capability each)

---

## Adding Playbooks

Create `.palace/playbooks/<name>.jsonc`:

```jsonc
{
  "schemaVersion": "1.0.0",
  "kind": "palace/playbook",
  "name": "add-api-endpoint",
  "purpose": "Add a new REST API endpoint",
  "rooms": ["api", "validation", "tests"],
  "steps": [
    "Define route in src/routes/",
    "Add request validation schema",
    "Implement handler",
    "Add integration test",
    "Update OpenAPI spec"
  ],
  "verification": [
    "palace verify --strict",
    "npm test"
  ]
}
```

### Guidelines

- Playbooks are declarative (don't run code)
- Steps should be actionable instructions
- Reference relevant Rooms

---

## Schema Versioning

All artifacts have `schemaVersion` and `kind`:

```jsonc
{
  "schemaVersion": "1.0.0",
  "kind": "palace/room",
  // ...
}
```

### Version Bumps

| Change Type | Version Bump | Example |
|-------------|--------------|---------|
| Breaking (remove field) | Major | 1.0.0 → 2.0.0 |
| New optional field | Minor | 1.0.0 → 1.1.0 |
| Bug fix | Patch | 1.0.0 → 1.0.1 |

---

## Modifying Schemas

Schemas live in `/schemas/*.schema.json`. To modify:

1. Edit schema file
2. Update CLI validation code if needed
3. Run `palace init --force` to re-export
4. Bump `schemaVersion` if breaking

### Breaking Changes

Avoid if possible. If necessary:

- Provide migration guidance
- Update all example configs
- Bump major version

---

## What's Deterministic

These must remain deterministic:

| Component | Guarantee |
|-----------|-----------|
| `scan` | Same files → same Index |
| `collect` | Same Index + Rooms → same context pack |
| `signal` | Same diff → same change signal |
| `verify` | Same state → same result |

Non-determinism is a bug.

---

## Future Extension Points

Planned but not implemented:

| Feature | Description |
|---------|-------------|
| Custom rankers | Plug-in scoring for Butler |
| Hooks | Pre/post command scripts |
| Sessions | Persist agent conversations |
| Maps | Visual dependency graphs |
