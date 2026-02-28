# Import Organisation from Spreadsheet

Import an organisation structure from a spreadsheet export (xlsx or csv) of platforms like HolaSpirit or Glassfrog.

## API

**`POST /auth/createorga/spreadsheet`**

Multipart form with the following fields:

| Field        | Type   | Required | Description                                    |
|-------------|--------|----------|------------------------------------------------|
| `file`      | file   | yes      | The spreadsheet file (.xlsx or .csv)           |
| `name`      | string | yes      | Organisation display name                      |
| `nameid`    | string | yes      | Organisation slug (URL-safe identifier)        |
| `format`    | string | no       | Source platform: `holaspirit`, `glassfrog`. Auto-detected if omitted |
| `visibility`| string | no       | `Public`, `Private`, or `Secret`. Default: `Public` |
| `about`     | string | no       | Organisation description                       |

Authentication: requires a valid JWT cookie.

### Response

**Success (200):**
```json
{"nameid": "my-org"}
```

**Error (400/500):**
Plain text error message.

### Example

```bash
curl -X POST http://localhost:8888/auth/createorga/spreadsheet \
  -H "Cookie: jwt=<token>" \
  -F "file=@export.xlsx" \
  -F "name=My Organisation" \
  -F "nameid=my-org" \
  -F "format=holaspirit"
```

## Supported File Formats

| Extension | Description |
|-----------|-------------|
| `.xlsx`   | Excel spreadsheet with multiple sheets |
| `.csv`    | Single CSV file. The filename (without extension) is used as the sheet name, so `Circles & Roles.csv` maps to the `"Circles & Roles"` sheet expected by HolaSpirit. |

## Supported Formats

### HolaSpirit

Auto-detected by the presence of a "Circles & Roles" sheet.

**Required sheets:**
- **Circles & Roles** — columns: Circle ID, Circle, Role ID, Role, Template, IsCircle, Purpose, Domains, Accountabilities, Strategy/Strategie, Created
- **Policies** (optional) — columns: Circle ID, Circle, Role ID, Role, Policy, Description

**HolaSpirit ID mapping:**
In HolaSpirit exports, each circle has two different IDs: a `roleID` (in its own `isCircle=TRUE` row) and a `circleID` (used when referenced as parent in child rows). These are different values. The importer resolves this via circle name matching to build the correct hierarchy.

**Root circle detection:**
The root circle is identified as the one with an empty `Circle ID` column (no parent).

**What gets imported:**
- Circle hierarchy (nested circles), ordered by creation time when available
- Roles within circles (with role type mapping)
- Mandate data: purpose, domains, responsibilities (from Accountabilities), policies
- Strategy content is appended to purpose as a `### Strategy` sub-section
- HTML content in cells is sanitized (bluemonday) then converted to markdown
- Policies from the Policies sheet are formatted as a markdown list under each circle/role
- Empty fields are left unset (nil) rather than set to empty strings
- Node names are sanitized via `NameidEncoder` (mirrors frontend Elm `nameidEncoder`)

**Role type mapping:**
- "Lead", "Leader", "Coordinateur", "Facilitateur", "1er lien" -> Coordinator
- Everything else -> Peer

**Role templates (RoleExt):**
- Roles marked with `Template=TRUE` in the HolaSpirit export are created as RoleExt templates at the root circle level
- Additionally, roles with identical names and content across multiple circles are automatically deduplicated into RoleExt templates
- All role instances referencing a template are linked to the corresponding RoleExt

**Not imported:**
- Members / assignations (deferred)
- The "Members" sheet is ignored

## Architecture

```
web/handlers/import.go             — ImportNode types, HTTP handler, org builder
web/handlers/import_readers.go     — xlsx reader + HTML-to-markdown converter
web/handlers/import_holaspirit.go  — HolaSpirit adapter: sheets -> ImportNode tree
web/handlers/import_test.go        — Unit tests
```

### Data flow

1. HTTP handler authenticates user and parses multipart form
2. Spreadsheet is read into `map[string][][]string` (sheet name -> rows)
3. Source format is detected from sheet names (or `format` field)
4. Format adapter parses sheets into an `ImportNode` tree
5. Builder creates root node + owner role (same as `CreateOrga`)
6. Builder creates RoleExt templates for deduplicated roles
7. Builder recursively creates child circles and roles with governance tensions

**Limits:**
- Maximum file size: 10 MB
- Maximum rows per sheet: 10,000

## Dependencies

- `github.com/xuri/excelize/v2` — xlsx reader
- `github.com/microcosm-cc/bluemonday` — HTML sanitization
- `encoding/csv` — CSV reader (stdlib)
- `golang.org/x/net/html` — HTML parsing for markdown conversion (stdlib)
