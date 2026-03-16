package utils

import "testing"

func TestVariantHelpersReturnClasses(t *testing.T) {
	buttonVariants := []string{"default", "primary", "destructive", "outline", "secondary", "ghost", "link"}
	for _, variant := range buttonVariants {
		if got := ButtonStyleVariant(variant); got == "" {
			t.Fatalf("ButtonStyleVariant(%q) returned empty classes", variant)
		}
	}
	buttonSizes := []string{"xs", "sm", "default", "md", "lg", "xl", "icon"}
	for _, size := range buttonSizes {
		if got := ButtonSizeVariant(size); got == "" {
			t.Fatalf("ButtonSizeVariant(%q) returned empty classes", size)
		}
	}
	badgeVariants := []string{"default", "secondary", "destructive", "outline", "success", "warning", "info"}
	for _, variant := range badgeVariants {
		if got := BadgeStyleVariant(variant); got == "" {
			t.Fatalf("BadgeStyleVariant(%q) returned empty classes", variant)
		}
	}
	badgeSizes := []string{"xs", "sm", "default", "lg"}
	for _, size := range badgeSizes {
		if got := BadgeSizeVariant(size); got == "" {
			t.Fatalf("BadgeSizeVariant(%q) returned empty classes", size)
		}
	}
	if got := TypographyClasses("sm", "medium", "6", "tight", "muted-foreground", "left", true); got == "" {
		t.Fatal("TypographyClasses returned empty classes")
	}
	fieldVariants := []string{"default", "outline", "ghost"}
	for _, variant := range fieldVariants {
		if got := FieldVariant(variant); got == "" {
			t.Fatalf("FieldVariant(%q) returned empty classes", variant)
		}
	}
	fieldSizes := []string{"xs", "sm", "default", "md", "lg"}
	for _, size := range fieldSizes {
		if got := FieldSizeVariant(size); got == "" {
			t.Fatalf("FieldSizeVariant(%q) returned empty classes", size)
		}
	}
	statuses := []string{"running", "stopped", "paused", "created", "deploying", "error", "unknown"}
	for _, status := range statuses {
		if got := StatusBadgeClass(status); got == "" {
			t.Fatalf("StatusBadgeClass(%q) returned empty classes", status)
		}
	}
}
