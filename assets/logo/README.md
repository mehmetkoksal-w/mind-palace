# Mind Palace Logo Assets

A brain with structured internal pathways - representing the "Mind Palace" concept of organized knowledge and memory.

## Logo Concept

The logo combines:

- **Brain shape**: Represents memory, cognition, and the "mind" aspect
- **Internal corridors/pathways**: Represents the structured "palace" - rooms, connections, organized knowledge
- **Gradient colors**: Palace Purple (#6B5B95) to Memory Blue (#4A90D9) - scholarly yet modern

## Available Files

| File             | Purpose                   | Use Case                           |
| ---------------- | ------------------------- | ---------------------------------- |
| `logo.svg`       | Primary logo              | Documentation, README, marketing   |
| `logo-dark.svg`  | Darker variant            | Light backgrounds                  |
| `logo-light.svg` | Lighter variant           | Dark backgrounds                   |
| `logo-mono.svg`  | Monochrome (currentColor) | Favicons, where color is inherited |

## Color Variants

### Primary (logo.svg)

- Outer shell: Gradient from #6B5B95 (Palace Purple) to #4A90D9 (Memory Blue)
- Inner corridors: White (#FFFFFF) at 95% opacity

### Dark (logo-dark.svg)

- Outer shell: Gradient from #5A4A84 to #3A7BC8 (slightly darker)
- Inner corridors: White (#FFFFFF) at 95% opacity

### Light (logo-light.svg)

- Outer shell: Gradient from #9B8BC5 to #6AB0F9 (lighter, more vibrant)
- Inner corridors: Dark (#1A1A2E) at 90% opacity

### Monochrome (logo-mono.svg)

- Uses `currentColor` - inherits from parent CSS
- Single color, no gradient

## Generating PNG Versions

To generate PNG versions for the extension and other uses:

```bash
# Using Inkscape (recommended)
inkscape logo.svg -w 128 -h 128 -o icon-128.png
inkscape logo.svg -w 256 -h 256 -o icon-256.png
inkscape logo.svg -w 512 -h 512 -o icon-512.png
inkscape logo.svg -w 16 -h 16 -o favicon-16.png
inkscape logo.svg -w 32 -h 32 -o favicon-32.png

# Using ImageMagick
convert -background none logo.svg -resize 128x128 icon-128.png

# Using rsvg-convert (librsvg)
rsvg-convert -w 128 -h 128 logo.svg > icon-128.png
```

## VS Code Extension Icon

Copy the 128x128 PNG to the extension:

```bash
cp icon-128.png ../../mind-palace-vscode/images/icon.png
```

Then update `mind-palace-vscode/package.json`:

```json
{
  "icon": "images/icon.png"
}
```

## Attribution

Original brain SVG from [SVG Repo](https://www.svgrepo.com/) - modified with brand colors and optimized for Mind Palace ecosystem.

## Design Guidelines

When using the logo:

1. **Minimum size**: 16x16px (favicon), 32x32px (recommended minimum)
2. **Clear space**: Maintain padding equal to 10% of logo width on all sides
3. **Background**: Use appropriate variant for background color
4. **Don't stretch**: Always maintain 1:1 aspect ratio
5. **Don't modify**: Use provided variants, don't alter colors or proportions
