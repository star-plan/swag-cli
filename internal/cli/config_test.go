package cli

import (
	"bufio"
	"strings"
	"testing"

	"swag-cli/internal/config"
)

func TestFormatConfigDiff(t *testing.T) {
	t.Parallel()

	oldCfg := config.Config{
		SwagDir:       "/old/swag",
		SwagContainer: "swag",
		Network:       "swag-net",
	}
	newCfg := config.Config{
		SwagDir:       "/new/swag",
		SwagContainer: "swag",
		Network:       "prod-net",
	}

	got := formatConfigDiff(oldCfg, newCfg)
	if !strings.Contains(got, "- swag-dir: /old/swag -> /new/swag") {
		t.Fatalf("formatConfigDiff() missing swag-dir diff: %q", got)
	}
	if !strings.Contains(got, "- network: swag-net -> prod-net") {
		t.Fatalf("formatConfigDiff() missing network diff: %q", got)
	}
	if strings.Contains(got, "swag-container") {
		t.Fatalf("formatConfigDiff() should omit unchanged keys: %q", got)
	}
}

func TestReadConfirmation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "yes", input: "yes\n", want: true},
		{name: "y", input: "y\n", want: true},
		{name: "no default", input: "\n", want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := readConfirmation(bufio.NewReader(strings.NewReader(tc.input)))
			if err != nil {
				t.Fatalf("readConfirmation() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("readConfirmation() = %v, want %v", got, tc.want)
			}
		})
	}
}
