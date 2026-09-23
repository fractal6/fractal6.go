# f6-extract-clients

Converts Fractale tensions exports (`.xlsx`) into a contacts workbook: `<name>_converted.xlsx`, written next to each input.

## Usage

- Double-click the binary and pick one or more `.xlsx` files, or
- Drag and drop files onto the binary (Windows, Linux; not macOS), or
- `f6-extract-clients file1.xlsx file2.xlsx`

A loading dialog shows progress; a final popup lists created files and failures.

## Output

One sheet per tension receiver. Columns:

| Column | Source |
|---|---|
| Nom de la société | tension title |
| Nom du contact | `contact[s]` |
| Format | `format[s]` |
| Numéro de téléphone | `téléphone`, `telephone`, `tél`, `tel`, `phone` |
| Adresse email | `[adresse] [e[-]]mail[s]` |
| Commercial | `commercial[s]` |
| Date | `date[s]`, `dâte[s]` |
| Prix | `prix`, `cost`, `pricing` |

Fields are read from the tension message: the first line starting with a key (case insensitive, after optional
list marker or bold) gives the rest of the line, separators (`:`, `;`) stripped. Empty if no match.
Keys are regexps, edit `fields` in `main.go`.

## Build

```sh
make          # all targets into build/
make windows  # build/f6-extract-clients.exe
make mac-arm64 | mac-intel | linux
make test
```

No cgo, cross-compiles from any OS.

## Platform notes

- Windows: SmartScreen may warn, "More info" > "Run anyway".
- macOS: first run, right-click > Open (or `xattr -d com.apple.quarantine <bin>`).
- Linux: requires `zenity` (`apt install zenity`); `chmod +x` after download.
- Existing `_converted.xlsx` files are overwritten.
