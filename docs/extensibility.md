# Extensibility & Versioning

Mind Palace is schema-first. Extend behavior by adding curated artifacts or evolving schemas carefully.

## Adding rooms
- Create `.palace/rooms/<name>.jsonc` following the embedded `room` schema.
- Define entry points and evidence relevant to the capability you want to collect.
- Reference the room from `palace.jsonc` (`defaultRoom`) or playbooks as needed.
- Keep paths normalized (forward slashes) to align with guardrails and index paths.

## Adding playbooks
- Create `.palace/playbooks/<name>.jsonc` using the embedded `playbook` schema.
- Describe routes through rooms, required evidence, and verification expectations.
- Playbooks are declarative; they do not execute commands. Keep instructions deterministic.

## Extending schemas safely
- Embedded schemas are canonical. To propose changes:
  1. Update the schema under `/schemas/*.schema.json`.
  2. Update any affected models/validation (if code changes are needed).
  3. Export via `palace init`/`palace detect` (or `CopySchemas`) to refresh `.palace/schemas/*`.
  4. Add tests that validate the new contract.
- Treat new required fields as breaking changes; optional fields (with defaults or clear semantics) are preferred for compatibility.

## Versioning expectations
- Schemas include `schemaVersion` and `kind` fields; bump versions when making breaking changes.
- Context-dependent artifacts (context-pack, change-signal, scan summary) must remain reproducible from input state; avoid non-deterministic fields.
- Align CLI flags and behavior with schema changes; never silently widen scope or relax validation.

## What counts as breaking
- Removing or renaming existing required fields.
- Changing semantics of guardrails (e.g., dropping defaults) or diff strictness.
- Introducing non-deterministic behavior in scan, collect, or verify.
- Changing file layout under `.palace/` without migration guidance.
