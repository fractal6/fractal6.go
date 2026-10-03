# Markdown Rendering

The frontend renders its own markdown. The backend converts in both directions for email:

- **Outbound** (notifications): markdown -> HTML, `web/email/main.go`.
- **Inbound** (email replies, Holaspirit import): HTML -> markdown, `internal/tools/markdown.go`.

## Outbound pipeline

goldmark converts markdown to HTML, then bluemonday sanitises it.

The parser drops `RawHTMLParser` from the default inline set, so bare `<tension>`-like
tokens stay literal instead of being swallowed as raw HTML. Extensions: GFM, plus a
custom `<details>`/`<summary>` extension (`web/email/goldmark_details.go`) whose body
is regular markdown. Soft line breaks become `<br>` (`WithHardWraps`).

The sanitizer is `UGCPolicy()` extended with `<details>`/`<summary>`, an inline
`style` attribute (needed by the details wrapper div), and a URL-scheme allowlist of
`http`, `https`, `mailto`, `cid`. `cid:` is required by the inline-attachment
rewriter; `javascript:` and everything else is denied.

## File attachments

Bodies may contain `![](/file/<id>)` references. They are rewritten in
`web/email/attachments.go` *after* the goldmark conversion and *before* the
sanitisation: embedded images within the caps become `cid:` inline attachments,
everything else falls back to an absolute `/file/<id>` URL that only resolves
anonymously for Public orgs. Plain attachments never appear in the body — they ship as
Postal attachments plus a footer link list.

The inbound direction (email replies carrying `cid:` refs) is resolved in
`web/handlers/cid.go`. See [file storage](file-storage.md) for both flows.

## Inbound pipeline

`decodeInboundEmail` (`web/handlers/mailer.go`) prefers Postal's `html_body` over the
wrapped plain body, then:

1. `HTMLToMarkdown`: bluemonday allowlist, then a walk of the HTML tree. Whitespace is
   collapsed like a browser (outside `<pre>`), so client source indentation never becomes
   an indented code block.
2. `StripEmailQuote` (`internal/tools/string.go`): drops the quote header, the Fractale
   footer and the signature.
