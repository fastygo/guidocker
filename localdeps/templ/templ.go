package templ

import (
	"context"
	"html"
	"io"
	"strings"
)

// Component matches the render contract used by templ-generated code.
type Component interface {
	Render(context.Context, io.Writer) error
}

// ComponentFunc adapts a function to a Component.
type ComponentFunc func(context.Context, io.Writer) error

func (f ComponentFunc) Render(ctx context.Context, w io.Writer) error {
	if f == nil {
		return nil
	}
	return f(ctx, w)
}

// Raw renders HTML without escaping.
func Raw(value string) Component {
	return ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, value)
		return err
	})
}

// EscapeString exposes HTML escaping helpers to component code.
func EscapeString(value string) string {
	return html.EscapeString(value)
}

// RenderToString renders a component into a string builder.
func RenderToString(component Component) (string, error) {
	if component == nil {
		return "", nil
	}
	var builder strings.Builder
	if err := component.Render(context.Background(), &builder); err != nil {
		return "", err
	}
	return builder.String(), nil
}
