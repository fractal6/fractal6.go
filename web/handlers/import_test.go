package handlers

import (
	"strings"
	"testing"

	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/tools"
)

func TestHtmlToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain text",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "paragraph",
			input:    "<p>hello world</p>",
			expected: "hello world",
		},
		{
			name:     "bold",
			input:    "<p><strong>bold text</strong></p>",
			expected: "**bold text**",
		},
		{
			name:     "italic",
			input:    "<p><em>italic text</em></p>",
			expected: "*italic text*",
		},
		{
			name:     "link",
			input:    `<a href="https://example.com">click here</a>`,
			expected: "[click here](https://example.com)",
		},
		{
			name:     "data-mention link stripped",
			input:    `<a data-mention="60993e048b3bea14f250e5f4|role">Outils numériques</a>`,
			expected: "Outils numériques",
		},
		{
			name:     "unordered list",
			input:    "<ul><li>item 1</li><li>item 2</li></ul>",
			expected: "- item 1\n- item 2",
		},
		{
			name:     "br becomes line break",
			input:    "line1<br />line2",
			expected: "line1  \nline2",
		},
		{
			name:     "nested bold in paragraph",
			input:    "<p>This is <strong>important</strong> text</p>",
			expected: "This is **important** text",
		},
		{
			name:     "inline code",
			input:    "<p>Use <code>fmt.Println</code> here</p>",
			expected: "Use `fmt.Println` here",
		},
		{
			name:     "fenced code block",
			input:    "<pre><code>func main() {\n  fmt.Println(\"hi\")\n}</code></pre>",
			expected: "```\nfunc main() {\n  fmt.Println(\"hi\")\n}\n```",
		},
		{
			name:     "strikethrough del",
			input:    "<p><del>removed</del></p>",
			expected: "~~removed~~",
		},
		{
			name:     "strikethrough s",
			input:    "<p><s>struck</s></p>",
			expected: "~~struck~~",
		},
		{
			name:     "blockquote",
			input:    "<blockquote><p>quoted text</p></blockquote>",
			expected: "> quoted text",
		},
		{
			name:     "horizontal rule",
			input:    "<p>above</p><hr><p>below</p>",
			expected: "above\n\n---\n\nbelow",
		},
		{
			name:     "image",
			input:    `<img src="https://example.com/img.png" alt="logo">`,
			expected: "![logo](https://example.com/img.png)",
		},
		{
			name:     "h5",
			input:    "<h5>Title Five</h5>",
			expected: "##### Title Five",
		},
		{
			name:     "h6",
			input:    "<h6>Title Six</h6>",
			expected: "###### Title Six",
		},
		{
			name:     "ordered list",
			input:    "<ol><li>first</li><li>second</li></ol>",
			expected: "1. first\n2. second",
		},
		{
			name:     "nested unordered list",
			input:    "<ul><li>a<ul><li>a1</li><li>a2</li></ul></li><li>b</li></ul>",
			expected: "- a\n  - a1\n  - a2\n- b",
		},
		{
			name:     "simple table",
			input:    "<table><thead><tr><th>Name</th><th>Age</th></tr></thead><tbody><tr><td>Alice</td><td>30</td></tr></tbody></table>",
			expected: "| Name | Age |\n| --- | --- |\n| Alice | 30 |",
		},
		{
			name:     "details with summary",
			input:    "<details><summary>More info</summary><p>Hidden content</p></details>",
			expected: "<details>\n<summary>More info</summary>\n\nHidden content\n\n</details>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tools.HTMLToMarkdown(tt.input)
			if err != nil {
				t.Fatalf("tools.HTMLToMarkdown(%q): unexpected error: %v", tt.input, err)
			}
			if got != tt.expected {
				t.Errorf("tools.HTMLToMarkdown(%q):\n  got:  %q\n  want: %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseHolaSpirit(t *testing.T) {
	// In HolaSpirit exports, each circle has two different IDs:
	//   - roleID: in the isCircle=TRUE row declaration
	//   - circleID: used when referenced as parent in child rows
	// The root circle has an empty circleID (no parent).
	sheets := map[string][][]string{
		"Circles & Roles": {
			// Header
			{"Circle ID", "Circle", "Role ID", "Role", "Template", "Hiring", "HasAssignation", "Created", "TimeSpent", "IsCircle", "Purpose", "Domains", "Accountabilities", "Stratégie"},
			// Root circle (empty circleID = no parent)
			{"", "", "root-role-id", "MyOrg", "", "", "FALSE", "2020-01-01 10:00:00.000000", "0", "TRUE", "<p>Root org purpose</p>", "", "", ""},
			// Sub-circle: Engineering (circleID "eng-circle-ref" != roleID "eng-role-id")
			// Its parent ref uses "org-circle-ref" which maps to "MyOrg" via name
			{"org-circle-ref", "MyOrg", "eng-role-id", "Engineering", "", "", "FALSE", "2021-01-15 10:00:00.000000", "0", "TRUE", "<p>Build great software</p>", "Code repositories", "<p>Review PRs</p>", ""},
			// Sub-circle: Backend under Engineering (uses "eng-circle-ref" as parent)
			{"eng-circle-ref", "Engineering", "back-role-id", "Backend", "", "", "FALSE", "2021-06-01 10:00:00.000000", "0", "TRUE", "<p>Backend services</p>", "", "", "<p>Scale to 1M users</p>"},
			// Role under Engineering (uses "eng-circle-ref" as parent)
			{"eng-circle-ref", "Engineering", "role1", "Lead", "TRUE", "", "TRUE", "2021-03-01 10:00:00.000000", "0", "FALSE", "<p>Lead the team</p>", "", "<p>Coordinate work</p>", ""},
			// Role under Backend (uses "back-circle-ref" as parent, maps to Backend via name)
			{"back-circle-ref", "Backend", "role2", "Lead", "TRUE", "", "TRUE", "2021-07-01 10:00:00.000000", "0", "FALSE", "<p>Lead the team</p>", "", "<p>Coordinate work</p>", ""},
			// Unique role under Backend
			{"back-circle-ref", "Backend", "role3", "Developer", "", "", "TRUE", "2021-08-01 10:00:00.000000", "0", "FALSE", "<p>Write code</p>", "", "", ""},
		},
		"Policies": {
			{"Circle ID", "Circle", "Role ID", "Role", "Domain", "Policy", "Description"},
			// Policy on Engineering (uses circleID ref, not roleID)
			{"eng-circle-ref", "Engineering", "", "", "All functions", "Code Review Policy", "<p>All PRs need <strong>two</strong> approvals</p>"},
		},
	}

	tree, err := parseHolaSpirit(sheets)
	if err != nil {
		t.Fatalf("parseHolaSpirit() error: %v", err)
	}

	// Root should be the circle with empty circleID: "MyOrg"
	if tree.Name != "MyOrg" {
		t.Errorf("root name = %q, want %q", tree.Name, "MyOrg")
	}
	if tree.Type != model.NodeTypeCircle {
		t.Errorf("root type = %v, want Circle", tree.Type)
	}
	if tree.Purpose != "Root org purpose" {
		t.Errorf("root purpose = %q, want %q", tree.Purpose, "Root org purpose")
	}

	// Root should have one child circle: Engineering
	var engineering *ImportNode
	for _, c := range tree.Children {
		if c.Name == "Engineering" {
			engineering = c
			break
		}
	}
	if engineering == nil {
		t.Fatal("missing child circle 'Engineering'")
	}
	if engineering.Type != model.NodeTypeCircle {
		t.Errorf("Engineering type = %v, want Circle", engineering.Type)
	}
	if engineering.Purpose != "Build great software" {
		t.Errorf("Engineering purpose = %q", engineering.Purpose)
	}

	// Engineering should have policies merged (via circleID mapping)
	if engineering.Policies == "" {
		t.Error("Engineering should have policies")
	}
	if expected := "- **Code Review Policy**"; !contains(engineering.Policies, expected) {
		t.Errorf("Engineering policies should contain %q, got %q", expected, engineering.Policies)
	}

	// Engineering should have Backend circle and Lead role as children
	var backend *ImportNode
	var engLead *ImportNode
	for _, c := range engineering.Children {
		switch c.Name {
		case "Backend":
			backend = c
		case "Lead":
			engLead = c
		}
	}
	if backend == nil {
		t.Fatal("missing child circle 'Backend'")
	}
	if engLead == nil {
		t.Fatal("missing child role 'Lead' under Engineering")
	}
	if engLead.Type != model.NodeTypeRole {
		t.Errorf("Lead type = %v, want Role", engLead.Type)
	}

	// Backend should have strategy in purpose
	if !contains(backend.Purpose, "### Strategy") {
		t.Errorf("Backend purpose should contain strategy section, got %q", backend.Purpose)
	}

	// Backend should have Lead and Developer roles
	var backLead, developer *ImportNode
	for _, c := range backend.Children {
		switch c.Name {
		case "Lead":
			backLead = c
		case "Developer":
			developer = c
		}
	}
	if backLead == nil {
		t.Fatal("missing role 'Lead' under Backend")
	}
	if developer == nil {
		t.Fatal("missing role 'Developer' under Backend")
	}

	// "Lead" has Template=TRUE -> should create RoleExt template
	if len(tree.RoleExtTemplates) == 0 {
		t.Fatal("expected RoleExt templates")
	}
	var leadTemplate *ImportRoleExt
	for _, re := range tree.RoleExtTemplates {
		if re.Name == "Lead" {
			leadTemplate = re
			break
		}
	}
	if leadTemplate == nil {
		t.Fatal("missing RoleExt template for 'Lead'")
	}
	if leadTemplate.RoleType != model.RoleTypeCoordinator {
		t.Errorf("Lead template RoleType = %v, want Coordinator", leadTemplate.RoleType)
	}

	// Both Lead roles should reference the template
	if engLead.RoleExtRef != "Lead" {
		t.Errorf("Engineering Lead RoleExtRef = %q, want 'Lead'", engLead.RoleExtRef)
	}
	if backLead.RoleExtRef != "Lead" {
		t.Errorf("Backend Lead RoleExtRef = %q, want 'Lead'", backLead.RoleExtRef)
	}
}

func TestParseHolaSpiritSortByCreated(t *testing.T) {
	sheets := map[string][][]string{
		"Circles & Roles": {
			{"Circle ID", "Circle", "Role ID", "Role", "Template", "Hiring", "HasAssignation", "Created", "TimeSpent", "IsCircle", "Purpose", "Domains", "Accountabilities", "Stratégie"},
			// Root
			{"", "", "root-id", "Root", "", "", "", "2020-01-01 10:00:00.000000", "", "TRUE", "", "", "", ""},
			// Roles added out of order by date
			{"root-ref", "Root", "r2", "Newer Role", "", "", "", "2022-06-01 10:00:00.000000", "", "FALSE", "", "", "", ""},
			{"root-ref", "Root", "r1", "Older Role", "", "", "", "2021-01-01 10:00:00.000000", "", "FALSE", "", "", "", ""},
		},
	}

	tree, err := parseHolaSpirit(sheets)
	if err != nil {
		t.Fatalf("parseHolaSpirit() error: %v", err)
	}

	// Children should be in creation-time order: Older Role before Newer Role
	if len(tree.Children) < 2 {
		t.Fatalf("expected at least 2 children, got %d", len(tree.Children))
	}
	if tree.Children[0].Name != "Older Role" {
		t.Errorf("first child = %q, want 'Older Role'", tree.Children[0].Name)
	}
	if tree.Children[1].Name != "Newer Role" {
		t.Errorf("second child = %q, want 'Newer Role'", tree.Children[1].Name)
	}
}

func TestBuildMandateRefNilForEmpty(t *testing.T) {
	// Empty node -> nil mandate
	node := &ImportNode{}
	if got := buildMandateRef(node); got != nil {
		t.Error("expected nil mandate for empty node")
	}

	// Node with only purpose -> other fields should be nil
	node = &ImportNode{Purpose: "test purpose"}
	got := buildMandateRef(node)
	if got == nil {
		t.Fatal("expected non-nil mandate")
	}
	if got.Purpose == nil || *got.Purpose != "test purpose" {
		t.Errorf("mandate purpose = %v, want 'test purpose'", got.Purpose)
	}
	if got.Domains != nil {
		t.Errorf("mandate domains should be nil for empty field, got %q", *got.Domains)
	}
	if got.Responsabilities != nil {
		t.Errorf("mandate responsabilities should be nil for empty field, got %q", *got.Responsabilities)
	}
	if got.Policies != nil {
		t.Errorf("mandate policies should be nil for empty field, got %q", *got.Policies)
	}
}

func TestReadCSV(t *testing.T) {
	csvData := "Circle ID,Circle,Role ID,Role,IsCircle,Purpose\norg1,MyOrg,circle1,Engineering,TRUE,Build software\ncircle1,Engineering,role1,Developer,FALSE,Write code\n"

	sheets, err := readCSV(strings.NewReader(csvData), "Circles & Roles.csv")
	if err != nil {
		t.Fatalf("readCSV() error: %v", err)
	}

	rows, ok := sheets["Circles & Roles"]
	if !ok {
		t.Fatal("expected sheet named 'Circles & Roles'")
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows (1 header + 2 data), got %d", len(rows))
	}
	if rows[0][0] != "Circle ID" {
		t.Errorf("header[0] = %q, want 'Circle ID'", rows[0][0])
	}
	if rows[1][3] != "Engineering" {
		t.Errorf("row1[3] = %q, want 'Engineering'", rows[1][3])
	}
}

func TestReadSpreadsheetCSV(t *testing.T) {
	csvData := "A,B,C\n1,2,3\n"

	sheets, err := readSpreadsheet(strings.NewReader(csvData), "mysheet.csv")
	if err != nil {
		t.Fatalf("readSpreadsheet(.csv) error: %v", err)
	}
	if _, ok := sheets["mysheet"]; !ok {
		t.Error("expected sheet named 'mysheet'")
	}
}

func TestReadSpreadsheetUnsupported(t *testing.T) {
	_, err := readSpreadsheet(strings.NewReader(""), "file.txt")
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestDetectSourceFormat(t *testing.T) {
	sheets := map[string][][]string{
		"Circles & Roles": {},
		"Policies":        {},
	}
	if got := detectSourceFormat(sheets); got != "holaspirit" {
		t.Errorf("detectSourceFormat() = %q, want 'holaspirit'", got)
	}

	empty := map[string][][]string{}
	if got := detectSourceFormat(empty); got != "" {
		t.Errorf("detectSourceFormat(empty) = %q, want ''", got)
	}
}

func TestMapHolaRoleType(t *testing.T) {
	tests := []struct {
		name     string
		expected model.RoleType
	}{
		{"Lead", model.RoleTypeCoordinator},
		{"Leader de Cercle", model.RoleTypeCoordinator},
		{"Facilitateur", model.RoleTypeCoordinator},
		{"Facilitatrice", model.RoleTypeCoordinator},
		{"1er lien", model.RoleTypeCoordinator},
		{"Developer", model.RoleTypePeer},
		{"Secretary", model.RoleTypePeer},
		{"Com&Médias", model.RoleTypePeer},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapHolaRoleType(tt.name)
			if got != tt.expected {
				t.Errorf("mapHolaRoleType(%q) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
