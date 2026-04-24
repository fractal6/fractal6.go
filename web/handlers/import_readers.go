/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
 *
 * This file is part of Fractale.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Fractale.  If not, see <http://www.gnu.org/licenses/>.
 */

package handlers

import (
	"encoding/csv"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

const maxImportRows = 10000

// readXLSX reads an xlsx file and returns a map of sheet name -> rows (each row is []string).
func readXLSX(file io.Reader) (map[string][][]string, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open xlsx: %w", err)
	}
	defer f.Close()

	sheets := make(map[string][][]string)
	for _, name := range f.GetSheetList() {
		rows, err := f.GetRows(name)
		if err != nil {
			return nil, fmt.Errorf("failed to read sheet %q: %w", name, err)
		}
		if len(rows) > maxImportRows {
			return nil, fmt.Errorf("sheet %q exceeds maximum of %d rows", name, maxImportRows)
		}
		sheets[name] = rows
	}
	return sheets, nil
}

// readCSV reads a CSV file and returns it as a single-sheet map.
// The sheet name is derived from the filename (without extension),
// so "Circles & Roles.csv" becomes sheet "Circles & Roles".
func readCSV(file io.Reader, filename string) (map[string][][]string, error) {
	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}
	if len(rows) > maxImportRows {
		return nil, fmt.Errorf("CSV exceeds maximum of %d rows", maxImportRows)
	}

	sheetName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	return map[string][][]string{sheetName: rows}, nil
}

// readSpreadsheet dispatches to the appropriate reader based on file extension.
func readSpreadsheet(file io.Reader, filename string) (map[string][][]string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".xlsx":
		return readXLSX(file)
	case ".csv":
		return readCSV(file, filename)
	default:
		return nil, fmt.Errorf("unsupported file format %q: only .xlsx and .csv are supported", ext)
	}
}
