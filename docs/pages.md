# Documentation & GitHub Pages Deployment

The canonical documentation lives in this repository under `/docs` (Markdown only). GitHub Pages should publish the `/docs` folder so users always see docs that match the current codebase.

## Folder structure
- `/docs/index.md`: overview, philosophy, setup (landing page for Pages).
- `/docs/concepts.md`: core concepts.
- `/docs/workflows.md`: end-to-end workflow.
- `/docs/cli.md`: command reference.
- `/docs/agents.md`: agent integration.
- `/docs/collaboration.md`: git/CI model.
- `/docs/extensibility.md`: schema/room/playbook evolution.
- `/docs/pages.md`: this deployment guide.

## Recommended Pages configuration
- **Source**: `/docs` folder in the repository (GitHub Pages “Deploy from a branch”, branch `master`, folder `/docs`).
- **Format**: Plain Markdown; no JS frameworks required.
- **Theme**: Optional GitHub Pages theme (or none) since content is Markdown-only.

## Deployment triggers (implemented)
- GitHub Actions workflow: `.github/workflows/docs.yml`.
- Triggers:
  - Push to `master` when `docs/**` or `README.md` changes.
  - Manual `workflow_dispatch`.
- Steps: checkout → configure-pages → upload-pages-artifact (`path: docs`) → deploy-pages.
- Permissions: `contents: read`, `pages: write`, `id-token: write`.
- Concurrency: group `pages` with `cancel-in-progress: true` to avoid overlapping deploys.

## Release approvals (environment gate)
- Releases use `.github/workflows/release.yml` with `environment: release`.
- Create a GitHub Environment named `release` and enable “Required reviewers” to enforce manual approval before release jobs proceed.
- The release workflow ignores docs-only changes (`docs/**`, `README.md`) so documentation updates do not trigger releases.

## Contributor guidance
- Update `/docs` and/or `README.md` when behavior changes. Keep docs aligned with the current CLI contracts.
- Do not place generated artifacts (`.palace/index`, `.palace/outputs`) in `/docs`.
- Keep examples deterministic (avoid timestamps) to prevent noisy diffs.
