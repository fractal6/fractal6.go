# Markdown Rendering

## Overview

Markdown rendering in fractal6.go is used to convert user-submitted markdown text into sanitized HTML for **email notifications**. The frontend handles its own markdown rendering independently.

## Pipeline

The rendering follows a two-step process:

1. **Markdown → HTML**: [goldmark](https://github.com/yuin/goldmark) v1.7.2 converts markdown to HTML
2. **HTML sanitization**: [bluemonday](https://github.com/microcosm-cc/bluemonday) strips unsafe tags (XSS prevention)

## Configuration

Defined in `web/email/main.go`:

```go
var md goldmark.Markdown = goldmark.New(
    goldmark.WithExtensions(
        extension.GFM,
        &detailsExtension{},
    ),
    goldmark.WithRendererOptions(
        html.WithHardWraps(),
    ),
)
```

### Extensions

| Extension | Source | Description |
|-----------|--------|-------------|
| GFM | goldmark built-in | GitHub Flavored Markdown (tables, strikethrough, task lists, autolinks) |
| detailsExtension | `web/email/goldmark_details.go` | Collapsible `<details>`/`<summary>` blocks with markdown body |

### Renderer Options

- `html.WithHardWraps()`: Soft line breaks become `<br>` tags

### Sanitizer

A custom `bluemonday.UGCPolicy()` extended with:
- `<details>` element (with `open` attribute allowed)
- `<summary>` element

## Usage Locations

All in `web/email/main.go`:

- **Tension creation** (~line 310): Converts the tension message body
- **Tension update/comment** (~line 397): Converts comment text
- **Contract notification** (~line 587): Converts contract notification messages

## Details/Summary Extension

Custom goldmark extension (`web/email/goldmark_details.go`) that parses collapsible content blocks.

### Markdown Syntax

```markdown
<details>
<summary>Click to expand</summary>

Body content with **markdown** support.

- List items work
- Code blocks work

</details>
```

The `open` attribute is supported:

```markdown
<details open>
<summary>Expanded by default</summary>

This section starts expanded.

</details>
```

### How It Works

1. **Block parser** (priority 650, above HTMLBlock at 600) detects `<details>` opening tags
2. `<summary>` lines are captured as metadata on a Summary AST node
3. Body content between `</summary>` and `</details>` is parsed as regular markdown (paragraphs, lists, code, etc.)
4. **Renderer** wraps the output in proper `<details>` and `<summary>` HTML elements
5. The sanitizer allows these elements through

### Rendered Output

```html
<details>
<summary>Click to expand</summary>
<p>Body content with <strong>markdown</strong> support.</p>
<ul>
<li>List items work</li>
<li>Code blocks work</li>
</ul>
</details>
```
