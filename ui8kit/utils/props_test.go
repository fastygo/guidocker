package utils

import "testing"

func TestUtilityPropsResolve(t *testing.T) {
	tests := []struct {
		name  string
		props UtilityProps
		want  string
	}{
		{name: "empty", props: UtilityProps{}, want: ""},
		{name: "single field", props: UtilityProps{P: "4"}, want: "p-4"},
		{name: "multiple fields", props: UtilityProps{P: "4", Mx: "auto", Bg: "card"}, want: "p-4 mx-auto bg-card"},
		{name: "flex direction", props: UtilityProps{Flex: "col"}, want: "flex flex-col"},
		{name: "gap alias", props: UtilityProps{Gap: "lg"}, want: "gap-6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.props.Resolve(); got != tt.want {
				t.Fatalf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}
