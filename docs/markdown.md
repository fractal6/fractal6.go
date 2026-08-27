# Markdown Rendering

The backend renders markdown only for **email notifications** — the frontend does its
own. Everything lives in `web/email/main.go`.

## Pipeline

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
