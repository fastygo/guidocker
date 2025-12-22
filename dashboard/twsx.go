package main

import (
	"fmt"
	"regexp"
	"strings"
)

// TailwindStyle represents a CSS property mapping
type TailwindStyle map[string]interface{}

// TwMap contains all supported Tailwind class mappings
var TwMap = map[string]TailwindStyle{
	// Display
	"block":        {"display": "block"},
	"inline-block": {"display": "inline-block"},
	"inline":       {"display": "inline"},
	"flex":         {"display": "flex"},
	"inline-flex":  {"display": "inline-flex"},
	"grid":         {"display": "grid"},
	"inline-grid":  {"display": "inline-grid"},
	"hidden":       {"display": "none"},
	"contents":     {"display": "contents"},

	// Flex Direction
	"flex-row":         {"flexDirection": "row"},
	"flex-row-reverse": {"flexDirection": "row-reverse"},
	"flex-col":         {"flexDirection": "column"},
	"flex-col-reverse": {"flexDirection": "column-reverse"},

	// Flex Wrap
	"flex-wrap":         {"flexWrap": "wrap"},
	"flex-wrap-reverse": {"flexWrap": "wrap-reverse"},
	"flex-nowrap":       {"flexWrap": "nowrap"},

	// Flex Grow/Shrink
	"flex-1":       {"flex": "1 1 0%"},
	"flex-auto":    {"flex": "1 1 auto"},
	"flex-initial": {"flex": "0 1 auto"},
	"flex-none":    {"flex": "none"},
	"grow":         {"flexGrow": 1},
	"grow-0":       {"flexGrow": 0},
	"shrink":       {"flexShrink": 1},
	"shrink-0":     {"flexShrink": 0},

	// Align Items
	"items-start":    {"alignItems": "flex-start"},
	"items-end":      {"alignItems": "flex-end"},
	"items-center":   {"alignItems": "center"},
	"items-baseline": {"alignItems": "baseline"},
	"items-stretch":  {"alignItems": "stretch"},

	// Align Self
	"self-auto":     {"alignSelf": "auto"},
	"self-start":    {"alignSelf": "flex-start"},
	"self-end":      {"alignSelf": "flex-end"},
	"self-center":   {"alignSelf": "center"},
	"self-stretch":  {"alignSelf": "stretch"},
	"self-baseline": {"alignSelf": "baseline"},

	// Justify Content
	"justify-start":   {"justifyContent": "flex-start"},
	"justify-end":     {"justifyContent": "flex-end"},
	"justify-center":  {"justifyContent": "center"},
	"justify-between": {"justifyContent": "space-between"},
	"justify-around":  {"justifyContent": "space-around"},
	"justify-evenly":  {"justifyContent": "space-evenly"},

	// Gap
	"gap-0": {"gap": 0},
	"gap-1": {"gap": 4},
	"gap-2": {"gap": 8},
	"gap-3": {"gap": 12},
	"gap-4": {"gap": 16},
	"gap-5": {"gap": 20},
	"gap-6": {"gap": 24},

	// Width
	"w-0":    {"width": 0},
	"w-1":    {"width": 4},
	"w-2":    {"width": 8},
	"w-4":    {"width": 16},
	"w-6":    {"width": 24},
	"w-8":    {"width": 32},
	"w-12":   {"width": 48},
	"w-16":   {"width": 64},
	"w-24":   {"width": 96},
	"w-32":   {"width": 128},
	"w-full": {"width": "100%"},
	"w-auto": {"width": "auto"},

	// Height
	"h-0":    {"height": 0},
	"h-1":    {"height": 4},
	"h-2":    {"height": 8},
	"h-4":    {"height": 16},
	"h-6":    {"height": 24},
	"h-8":    {"height": 32},
	"h-12":   {"height": 48},
	"h-16":   {"height": 64},
	"h-24":   {"height": 96},
	"h-32":   {"height": 128},
	"h-full": {"height": "100%"},
	"h-auto": {"height": "auto"},

	// Padding
	"p-0": {"padding": 0},
	"p-1": {"padding": 4},
	"p-2": {"padding": 8},
	"p-3": {"padding": 12},
	"p-4": {"padding": 16},
	"p-6": {"padding": 24},

	// Margin
	"m-0":    {"margin": 0},
	"m-1":    {"margin": 4},
	"m-2":    {"margin": 8},
	"m-4":    {"margin": 16},
	"m-auto": {"margin": "auto"},

	// Colors (simplified - using CSS variables)
	"bg-transparent": {"backgroundColor": "transparent"},
	"bg-white":       {"backgroundColor": "white"},
	"bg-black":       {"backgroundColor": "black"},
	"text-black":     {"color": "black"},
	"text-white":     {"color": "white"},

	// Border Radius
	"rounded":      {"borderRadius": 4},
	"rounded-md":   {"borderRadius": 6},
	"rounded-lg":   {"borderRadius": 8},
	"rounded-xl":   {"borderRadius": 12},
	"rounded-full": {"borderRadius": 9999},

	// Shadow
	"shadow":    {"boxShadow": "0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1)"},
	"shadow-md": {"boxShadow": "0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)"},
	"shadow-lg": {"boxShadow": "0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1)"},

	// Position
	"relative": {"position": "relative"},
	"absolute": {"position": "absolute"},
	"fixed":    {"position": "fixed"},

	// Text
	"text-xs":     {"fontSize": 12},
	"text-sm":     {"fontSize": 14},
	"text-base":   {"fontSize": 16},
	"text-lg":     {"fontSize": 18},
	"text-xl":     {"fontSize": 20},
	"text-center": {"textAlign": "center"},
	"text-left":   {"textAlign": "left"},
	"text-right":  {"textAlign": "right"},
}

// TWSX converts Tailwind class strings to CSS properties map
func TWSX(classStrings ...string) map[string]interface{} {
	// Filter and join all class strings
	var validStrings []string
	for _, s := range classStrings {
		if s != "" {
			validStrings = append(validStrings, s)
		}
	}

	input := strings.Join(validStrings, " ")
	input = strings.TrimSpace(input)

	if input == "" {
		return map[string]interface{}{}
	}

	// Parse classes
	classes := regexp.MustCompile(`\s+`).Split(input, -1)
	merged := make(map[string]interface{})

	for _, class := range classes {
		if class == "" {
			continue
		}

		if styles, exists := TwMap[class]; exists {
			for key, value := range styles {
				merged[key] = value
			}
		} else {
			fmt.Printf("[twsx] Warning: Unknown class \"%s\"\n", class)
		}
	}

	return merged
}

// TWSXCreate creates reusable style objects
func TWSXCreate(definitions map[string]string) map[string]map[string]interface{} {
	result := make(map[string]map[string]interface{})

	for key, classes := range definitions {
		result[key] = TWSX(classes)
	}

	return result
}

// MergeStyles merges multiple style objects
func MergeStyles(styles ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for _, style := range styles {
		if style != nil {
			for key, value := range style {
				result[key] = value
			}
		}
	}

	return result
}

// ValidateClasses checks if all classes in the string are supported
func ValidateClasses(classString string) []string {
	classes := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(classString), -1)
	var invalidClasses []string

	for _, class := range classes {
		if class != "" {
			if _, exists := TwMap[class]; !exists {
				invalidClasses = append(invalidClasses, class)
			}
		}
	}

	return invalidClasses
}

// GetSupportedClasses returns all supported Tailwind classes
func GetSupportedClasses() []string {
	classes := make([]string, 0, len(TwMap))
	for class := range TwMap {
		classes = append(classes, class)
	}
	return classes
}

// StylesToInlineCSS converts style map to CSS string for HTML style attribute
func StylesToInlineCSS(styles map[string]interface{}) string {
	var cssParts []string
	for key, value := range styles {
		// Convert camelCase to kebab-case
		kebabKey := camelToKebab(key)
		cssParts = append(cssParts, kebabKey+":"+fmt.Sprintf("%v", value))
	}
	return strings.Join(cssParts, ";")
}

func camelToKebab(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '-')
		}
		if r >= 'A' && r <= 'Z' {
			result = append(result, r-'A'+'a')
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}
