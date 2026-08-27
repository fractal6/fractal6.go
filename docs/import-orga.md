# Import Organisation from Spreadsheet

Create an organisation from a spreadsheet export of another platform (HolaSpirit,
Glassfrog).

`POST /auth/createorga/spreadsheet` takes a multipart form with the file plus the
usual org creation fields (name, nameid, visibility, about) and an optional `format`
override — the source platform is otherwise auto-detected from the sheet names. Needs
a valid JWT cookie. Handler signature and validation live in `web/handlers/import.go`.

`.xlsx` (multi-sheet) and `.csv` (single sheet, named after the file) are accepted,
capped at 10 MB and 10k rows per sheet.

## Where things live

```
web/handlers/import.go             ImportNode types, HTTP handler, org builder
web/handlers/import_readers.go     xlsx/csv readers + HTML-to-markdown converter
web/handlers/import_holaspirit.go  HolaSpirit adapter: sheets -> ImportNode tree
```

Dependencies: `excelize/v2` (xlsx), `bluemonday` (sanitisation), `x/net/html` (markdown
conversion), stdlib `encoding/csv`.

## Data flow

The handler authenticates and parses the form, the reader turns the file into
`sheet -> rows`, a format adapter parses that into an `ImportNode` tree, and the
builder persists it: root node + owner role (same path as `CreateOrga`), then RoleExt
templates, then circles and roles created recursively with a governance tension each.
Every governance tension stores a complete Node fragment and then links
`Node.source` / `Tension.governed_node` for the Node it just created.

## HolaSpirit specifics

Detected by the "Circles & Roles" sheet; an optional "Policies" sheet is merged in.
The root circle is the row with an empty `Circle ID`.

Circles carry two distinct ids — a `roleID` on their own row and a `circleID` when
referenced as a parent — so the adapter resolves the hierarchy by circle name instead.

Imported: circle hierarchy (creation-ordered), roles with a type mapping (lead /
coordinator-ish names -> Coordinator, everything else -> Peer), and mandate content
(purpose, domains, responsibilities, policies, strategy appended as a sub-section).
HTML cells are sanitised then converted to markdown; names go through `NameidEncoder`,
mirroring the frontend's Elm encoder. Empty fields stay nil.

Roles flagged `Template=TRUE`, and roles that repeat identically across circles, become
RoleExt templates at root level, with every instance linked back.

Members and assignations are not imported.

## Failure behaviour

Parsing and validation complete before persistence, but org creation is multi-step with
no rollback: a failure can leave a partial organisation. Retrying the request is not a
resume and is not idempotent — remove the partial org first, or use a fresh nameid
(which does not clean up the previous attempt).

The governance-link step only writes the Node it just created; it repairs nothing.
Existing orgs are backfilled by the upgrade script in [node governance](node-governance.md).
