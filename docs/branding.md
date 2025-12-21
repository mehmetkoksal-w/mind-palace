---
layout: default
title: Branding
nav_order: 11
---

# Mind Palace Branding Guidelines

This document defines the visual identity for the Mind Palace ecosystem.

## Logo

The Mind Palace logo is a **brain with structured internal pathways** - representing the concept of organized knowledge and memory (the "palace" within the "mind").

**Logo files** (in `/assets/logo/`):

| File | Purpose |
|------|---------|
| `logo.svg` | Primary logo with gradient |
| `logo-dark.svg` | For light backgrounds (darker gradient) |
| `logo-light.svg` | For dark backgrounds (lighter gradient) |
| `logo-mono.svg` | Monochrome (uses currentColor) |
| `icon-128.png` | Extension icon |
| `icon-256.png` | README badges |
| `icon-512.png` | Marketing materials |
| `favicon-16.png`, `favicon-32.png` | Favicons |

## Colors

### Primary Palette

The purple/blue gradient was chosen to be **distinctive** (most dev tools use green or orange) and **semantically appropriate** (purple = wisdom/knowledge, blue = structure/trust).

| Name | Hex | Usage |
|------|-----|-------|
| Palace Purple | `#6B5B95` | Primary brand color, headers, links, graph room nodes |
| Memory Blue | `#4A90D9` | Secondary accent, interactive elements, gradient end |
| Archive Gray | `#2D3748` | Text, backgrounds (dark mode) |
| Parchment | `#F7F5F2` | Backgrounds (light mode) |

### Status Colors

Universal status colors for clarity across all contexts:

| State | Hex | Usage |
|-------|-----|-------|
| Fresh/Synced | `#10B981` | Green - index is current |
| Stale | `#EF4444` | Red - needs healing |
| Scanning | `#F59E0B` | Amber - operation in progress |
| Error | `#DC2626` | Red (darker) - failure state |

### Extension Color Strategy

The VS Code extension uses a **hybrid approach**:

| Element | Color Source | Rationale |
|---------|--------------|-----------|
| Status bar states | VS Code theme | Respects user's theme preference |
| Sidebar background | VS Code theme | Blends with editor |
| Logo/icon | Brand colors | Consistent identity |
| Graph room nodes | Palace Purple | Recognizable, branded |
| Graph file nodes | Theme foreground | Readable in any theme |

```json
{
  "mindPalace.staleBackground": "statusBarItem.errorBackground",
  "mindPalace.freshBackground": "statusBarItem.prominentBackground"
}
```

This ensures the extension feels native while remaining identifiable.

## Typography

### Documentation

- **Headings**: System sans-serif (via just-the-docs theme)
- **Body**: System sans-serif
- **Code**: System monospace

### CLI Output

- Monospace terminal font (user's terminal default)
- ANSI colors for status indicators

## Naming Conventions

The naming follows the **palace metaphor** consistently:

| Component | Full Name | Short Name | Metaphor |
|-----------|-----------|------------|----------|
| CLI | Mind Palace CLI | `palace` | The palace itself |
| Extension | Mind Palace Observer | Observer | Watches over the palace |
| Search Engine | Butler | Butler | Servant who helps you find things |
| Workspace Structure | Room | Room | A place in the palace |
| Task Template | Playbook | Playbook | Instructions for tasks |
| Multi-repo Link | Corridor | Corridor | Connects rooms/palaces |
| Index Database | Tier-0 Index | Index | The palace's catalog |

## Voice & Tone

- **Technical but approachable**: Clear explanations without jargon overload
- **Deterministic**: Emphasize reliability, predictability, contracts
- **Agent-friendly**: Speak to both humans and AI agents as users

## Asset Locations

```
mind-palace/
├── assets/
│   ├── logo/           # Logo files
│   └── screenshots/    # Documentation screenshots
└── docs/
    └── branding.md     # This file
```

## Usage Examples

### README Badge

```markdown
![Mind Palace](https://img.shields.io/badge/Mind%20Palace-v0.0.1-6B5B95)
```

### Extension Marketplace

- Icon: 128x128 PNG with transparent background
- Banner: 1280x640 with Palace Purple background

### Documentation Header

The just-the-docs theme handles styling. Override in `_config.yml` if needed.
