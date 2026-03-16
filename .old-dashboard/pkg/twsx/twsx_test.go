package twsx

import (
	"strings"
	"testing"
)

func TestTWSX_MapsKnownClasses(t *testing.T) {
	styles := TWSX("flex", "items-center", "gap-4")
	if styles["display"] != "flex" {
		t.Fatalf("expected display to be flex, got %v", styles["display"])
	}
	if styles["alignItems"] != "center" {
		t.Fatalf("expected alignItems to be center, got %v", styles["alignItems"])
	}
	if styles["gap"] != 16 {
		t.Fatalf("expected gap to be 16, got %v", styles["gap"])
	}
}

func TestTWSX_IgnoresUnknownClasses(t *testing.T) {
	styles := TWSX("flex", "unknown-class")
	if _, ok := styles["unknown-class"]; ok {
		t.Fatalf("did not expect unknown class to be added")
	}
	if _, ok := styles["display"]; !ok {
		t.Fatalf("expected display to be set")
	}
}

func TestStyleRegistry_GenerateCSS(t *testing.T) {
	registry := NewStyleRegistry()
	registry.CLASS("demo", TWSX("bg-white rounded"))
	css := registry.GenerateCSS()

	if !strings.Contains(css, ".demo{") {
		t.Fatalf("expected generated css to include .demo class")
	}
	if !strings.Contains(css, "background-color:white") {
		t.Fatalf("expected background-color style in generated css")
	}
}
