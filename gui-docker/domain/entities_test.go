package domain

import "testing"

func TestNormalizeStoredStatus(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"running", "running"},
		{"start", "running"},
		{"Stop", "stopped"},
		{"PAUSE", "paused"},
		{"unknown", "unknown"},
	}

	for _, tc := range cases {
		if got := NormalizeStoredStatus(tc.input); got != tc.expected {
			t.Fatalf("NormalizeStoredStatus(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestParseStatusForUpdate(t *testing.T) {
	if got, ok := ParseStatusForUpdate("  stop "); !ok || got != "stopped" {
		t.Fatalf("ParseStatusForUpdate(\" stop \") = %v, %v", got, ok)
	}

	if _, ok := ParseStatusForUpdate("invalid"); ok {
		t.Fatal("expected invalid status to be rejected")
	}
}

func TestFormatStatusLabel(t *testing.T) {
	if got := FormatStatusLabel("running"); got != "Running" {
		t.Fatalf("FormatStatusLabel(\"running\") = %q", got)
	}

	if got := FormatStatusLabel("  PaUsEd "); got != "Paused" {
		t.Fatalf("FormatStatusLabel(\"  PaUsEd \") = %q", got)
	}
}
