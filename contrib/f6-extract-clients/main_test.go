package main

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestConvert(t *testing.T) {
	msg := "Intro line, contact us later\r\n" +
		"- **Contact** : Jane Doe\n" +
		"* TÉL ; 01 02 03 04 05  \n" +
		"Adresse   e-mail: jane@acme.fr\n" +
		"1. Dâtes:: 2026-01-01\n" +
		"Format:\n" +
		"Pricing 1000€\n" +
		"Telephone: ignored, first match wins\n"

	src := excelize.NewFile()
	header := []any{"id", "title", "receiver", "message"}
	src.SetSheetRow("Sheet1", "A1", &header)
	rows := [][]any{
		{"1", "Acme", "Sales/Leads", msg},
		{"2", "Foo", "Other", "email: foo@bar.io"},
		{"3", "Bar", "Sales/Leads", ""},
	}
	for i, r := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		src.SetSheetRow("Sheet1", cell, &r)
	}

	dst, err := convert(src)
	if err != nil {
		t.Fatal(err)
	}
	if got := dst.GetSheetList(); len(got) != 2 || got[0] != "Sales_Leads" || got[1] != "Other" {
		t.Fatalf("sheets: %v", got)
	}
	got, _ := dst.GetRows("Sales_Leads")
	want := []string{"Acme", "Jane Doe", "", "01 02 03 04 05", "jane@acme.fr", "", "2026-01-01", "1000€"}
	for i, w := range want {
		if i >= len(got[1]) && w == "" {
			continue
		}
		if got[1][i] != w {
			t.Errorf("%s: got %q, want %q", fields[i].header, got[1][i], w)
		}
	}
	if len(got) != 3 || got[2][0] != "Bar" {
		t.Errorf("second Sales_Leads row: %v", got)
	}
	other, _ := dst.GetRows("Other")
	if other[1][4] != "foo@bar.io" {
		t.Errorf("Other email: %v", other[1])
	}
}
