package ui

import (
	"context"
	"fmt"
	"github.com/a-h/templ"
	"html"
	"io"
	"strings"
	"ui8kit/utils"
)

type BoxProps struct {
	utils.UtilityProps
	Class string
	Tag   string
}

type StackProps struct {
	utils.UtilityProps
	Class string
	Tag   string
}

type GroupProps struct {
	utils.UtilityProps
	Class string
	Tag   string
	Grow  bool
}

type ContainerProps struct {
	utils.UtilityProps
	Class string
}

type BlockProps = BoxProps

type ButtonProps struct {
	utils.UtilityProps
	Variant  string
	Size     string
	Href     string
	Class    string
	Type     string
	Disabled bool
	OnClick  string
}

type BadgeProps struct {
	utils.UtilityProps
	Variant string
	Size    string
	Class   string
}

type TextProps struct {
	utils.UtilityProps
	Class         string
	Tag           string
	FontSize      string
	FontWeight    string
	LineHeight    string
	LetterSpacing string
	TextColor     string
	TextAlign     string
	Truncate      bool
}

type TitleProps struct {
	utils.UtilityProps
	Class         string
	Order         int
	FontSize      string
	FontWeight    string
	LineHeight    string
	LetterSpacing string
	TextColor     string
	TextAlign     string
	Truncate      bool
}

type FieldOption struct {
	Value string
	Label string
}

type FieldProps struct {
	utils.UtilityProps
	Class       string
	Variant     string
	Size        string
	Type        string
	Name        string
	ID          string
	Placeholder string
	Value       string
	Rows        int
	Min         string
	Max         string
	Checked     bool
	Disabled    bool
	Component   string
	Options     []FieldOption
}

type IconProps struct {
	Name  string
	Size  string
	Class string
}

func writeString(w io.Writer, value string) error {
	_, err := io.WriteString(w, value)
	return err
}

func renderWrapped(tag, className string, children templ.Component) templ.Component {
	if strings.TrimSpace(tag) == "" {
		tag = "div"
	}
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if err := writeString(w, fmt.Sprintf(`<%s class="%s">`, tag, html.EscapeString(strings.TrimSpace(className)))); err != nil {
			return err
		}
		if children != nil {
			if err := children.Render(ctx, w); err != nil {
				return err
			}
		}
		return writeString(w, fmt.Sprintf("</%s>", tag))
	})
}

func Box(props BoxProps, children templ.Component) templ.Component {
	return renderWrapped(props.Tag, utils.Cn(props.Resolve(), props.Class), children)
}

func Stack(props StackProps, children templ.Component) templ.Component {
	return renderWrapped(props.Tag, utils.Cn("flex flex-col gap-4 items-start justify-start", props.Resolve(), props.Class), children)
}

func Group(props GroupProps, children templ.Component) templ.Component {
	base := "flex gap-4 items-center justify-start min-w-0"
	if props.Grow {
		base = utils.Cn(base, "w-full")
	}
	return renderWrapped(props.Tag, utils.Cn(base, props.Resolve(), props.Class), children)
}

func Container(props ContainerProps, children templ.Component) templ.Component {
	return renderWrapped("div", utils.Cn("max-w-7xl mx-auto px-4", props.Resolve(), props.Class), children)
}

func Block(props BlockProps, children templ.Component) templ.Component {
	return Box(BoxProps(props), children)
}

func Button(props ButtonProps, label string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		className := utils.Cn(utils.ButtonStyleVariant(props.Variant), utils.ButtonSizeVariant(props.Size), props.Resolve(), props.Class)
		if props.Disabled {
			className = utils.Cn(className, "pointer-events-none opacity-50")
		}
		label = html.EscapeString(label)
		if strings.TrimSpace(props.Href) != "" {
			return writeString(w, fmt.Sprintf(`<a href="%s" class="%s">%s</a>`, html.EscapeString(props.Href), html.EscapeString(className), label))
		}
		buttonType := props.Type
		if buttonType == "" {
			buttonType = "button"
		}
		disabled := ""
		if props.Disabled {
			disabled = " disabled"
		}
		return writeString(w, fmt.Sprintf(`<button type="%s" class="%s"%s>%s</button>`, html.EscapeString(buttonType), html.EscapeString(className), disabled, label))
	})
}

func Badge(props BadgeProps, label string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		className := utils.Cn(utils.BadgeStyleVariant(props.Variant), utils.BadgeSizeVariant(props.Size), props.Resolve(), props.Class)
		return writeString(w, fmt.Sprintf(`<span class="%s">%s</span>`, html.EscapeString(className), html.EscapeString(label)))
	})
}

func Text(props TextProps, value string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		className := utils.Cn(utils.TypographyClasses(props.FontSize, props.FontWeight, props.LineHeight, props.LetterSpacing, props.TextColor, props.TextAlign, props.Truncate), props.Resolve(), props.Class)
		return writeString(w, fmt.Sprintf(`<p class="%s">%s</p>`, html.EscapeString(className), html.EscapeString(value)))
	})
}

func Title(props TitleProps, value string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		tag := fmt.Sprintf("h%d", props.Order)
		if props.Order < 1 || props.Order > 6 {
			tag = "h2"
		}
		fontSize := props.FontSize
		if fontSize == "" {
			fontSize = "2xl"
		}
		fontWeight := props.FontWeight
		if fontWeight == "" {
			fontWeight = "semibold"
		}
		className := utils.Cn(utils.TypographyClasses(fontSize, fontWeight, props.LineHeight, props.LetterSpacing, props.TextColor, props.TextAlign, props.Truncate), props.Resolve(), props.Class)
		return writeString(w, fmt.Sprintf(`<%s class="%s">%s</%s>`, tag, html.EscapeString(className), html.EscapeString(value), tag))
	})
}

func Field(props FieldProps) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		className := utils.Cn(utils.FieldVariant(props.Variant), utils.FieldSizeVariant(props.Size), props.Resolve(), props.Class)
		disabled := ""
		if props.Disabled {
			disabled = " disabled"
		}
		switch props.Component {
		case "textarea":
			rows := props.Rows
			if rows <= 0 {
				rows = 4
			}
			return writeString(w, fmt.Sprintf(`<textarea id="%s" name="%s" class="%s" rows="%d" placeholder="%s"%s>%s</textarea>`, html.EscapeString(props.ID), html.EscapeString(props.Name), html.EscapeString(className), rows, html.EscapeString(props.Placeholder), disabled, html.EscapeString(props.Value)))
		case "select":
			if err := writeString(w, fmt.Sprintf(`<select id="%s" name="%s" class="%s"%s>`, html.EscapeString(props.ID), html.EscapeString(props.Name), html.EscapeString(className), disabled)); err != nil {
				return err
			}
			for _, option := range props.Options {
				selected := ""
				if option.Value == props.Value {
					selected = " selected"
				}
				if err := writeString(w, fmt.Sprintf(`<option value="%s"%s>%s</option>`, html.EscapeString(option.Value), selected, html.EscapeString(option.Label))); err != nil {
					return err
				}
			}
			return writeString(w, "</select>")
		default:
			inputType := props.Type
			if inputType == "" {
				inputType = "text"
			}
			checked := ""
			if props.Checked {
				checked = " checked"
			}
			return writeString(w, fmt.Sprintf(`<input id="%s" name="%s" type="%s" class="%s" placeholder="%s" value="%s" min="%s" max="%s"%s%s />`, html.EscapeString(props.ID), html.EscapeString(props.Name), html.EscapeString(inputType), html.EscapeString(className), html.EscapeString(props.Placeholder), html.EscapeString(props.Value), html.EscapeString(props.Min), html.EscapeString(props.Max), checked, disabled))
		}
	})
}

func Icon(props IconProps) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		size := props.Size
		switch size {
		case "xs":
			size = "h-3 w-3"
		case "", "sm":
			size = "h-4 w-4"
		case "md":
			size = "h-5 w-5"
		case "lg":
			size = "h-6 w-6"
		}
		className := utils.Cn("latty", "latty-"+props.Name, size, props.Class)
		return writeString(w, fmt.Sprintf(`<span class="%s"></span>`, html.EscapeString(className)))
	})
}
