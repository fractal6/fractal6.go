# Markdown Rendering

## Overview

Markdown rendering in fractal6.go is used to convert user-submitted markdown text into sanitized HTML for **email notifications**. The frontend handles its own markdown rendering independently.

## Pipeline

The rendering follows a two-step process:

1. **Markdown → HTML**: [goldmark](https://github.com/yuin/goldmark) v1.7.2 converts markdown to HTML
2. **HTML sanitization**: [bluemonday](https://github.com/microcosm-cc/bluemonday) strips unsafe tags (XSS prevention)

## Configuration

Defined in `web/email/main.go`. The parser overrides the default inline
parser set to drop `RawHTMLParser` — bare `<tension>` and similar
angle-bracket tokens stay literal (escaped) instead of being silently
swallowed as raw HTML:

```go
var md goldmark.Markdown = goldmark.New(
    goldmark.WithParser(parser.NewParser(
        parser.WithBlockParsers(parser.DefaultBlockParsers()...),
        parser.WithInlineParsers(inlineParsersNoRawHTML...),
        parser.WithParagraphTransformers(parser.DefaultParagraphTransformers()...),
    )),
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
- inline `style` attribute on `<div>`, `<span>`, `<details>` — needed by the
  details extension's wrapper div
- URL-scheme allowlist set to `http`, `https`, `mailto`, `cid` — `cid:` is
  required so the inline-image attachment rewriter can substitute
  `<img src="/file/<id>">` with `<img src="cid:<fid>@<DOMAIN>">` without
  bluemonday dropping the src attribute. `javascript:` and other schemes
  are denied.

## Usage Locations

All in `web/email/main.go`:

- **Tension creation** (~line 310): Converts the tension message body
- **Tension update/comment** (~line 397): Converts comment text
- **Contract notification** (~line 587): Converts contract notification messages

## File-attachment URLs in emails

Markdown bodies may contain `![](/file/<id>)` references (see
`docs/file-storage.md`). For email rendering, these are rewritten in
`web/email/attachments.go` AFTER the goldmark conversion and BEFORE the
bluemonday sanitisation:

- **Inline-image files** (`File.embedded=true` AND a safe image type, within
  the per-email caps) get `<img src="cid:<fid>@<DOMAIN>">` and ride out as
  RFC 2392 inline attachments in the Postal payload — they render in-body
  in the recipient's mail UI regardless of org visibility.
- **Everything else** (non-image, non-embedded, oversized, cap overflow)
  falls back to `<img src="https://<DOMAIN>/file/<id>">`. Public-org images
  may render anonymously; Private/Secret-org ones 404 in the mail client —
  which is when the recipient should click through and authenticate.

Plain (non-image) attachments don't appear in the body at all; they ship
as standard Postal attachments and also as a footer link list. See
`docs/file-storage.md` "Email notifications" for caps and gate semantics.

**Reply direction (inbound).** When a user replies by email, any `cid:<token>`
references reaching `POST /notifications` are resolved against the inbound
attachment list (filename heuristic + document-order fallback) and rewritten
to `/file/<fid>` before the comment is persisted; quoted-back
`cid:<fid>@<DOMAIN>` references from the original notification short-circuit
to the same form without re-uploading the file. Unresolved refs are dropped
from the message. See `docs/file-storage.md` "Inbound email replies".

## Details/Summary Extension

Custom goldmark extension (`web/email/goldmark_details.go`) — parses `<details>` / `<summary>` blocks where the body is regular markdown. The `open` attribute is supported. The sanitizer above is configured to let these elements through.

```markdown
<details open>
<summary>Click to expand</summary>

Body content with **markdown** support.

- List items work
- Code blocks work

</details>
```
