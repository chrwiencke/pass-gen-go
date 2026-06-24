package shortcut

import "testing"

func TestParseAllowsShiftModifier(t *testing.T) {
	parsed, err := Parse("control shift p")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if !parsed.Control || !parsed.Shift || parsed.Key != "P" {
		t.Fatalf("Parse() = %+v, want Ctrl+Shift+P", parsed)
	}
	if got := parsed.String(); got != "Ctrl+Shift+P" {
		t.Fatalf("String() = %q, want %q", got, "Ctrl+Shift+P")
	}
}

func TestParseRequiresModifier(t *testing.T) {
	if _, err := Parse("p"); err == nil {
		t.Fatal("Parse() succeeded without a modifier")
	}
}
