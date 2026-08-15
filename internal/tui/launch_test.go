package tui_test

import (
	"testing"

	"github.com/chasebank87/PourOver/internal/tui"
)

func TestShouldAutoLaunch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  tui.LaunchEnv
		want bool
	}{
		{
			name: "non-interactive",
			env:  tui.LaunchEnv{Interactive: false, Args: nil, CI: false},
			want: false,
		},
		{
			name: "interactive no args CI false",
			env:  tui.LaunchEnv{Interactive: true, Args: nil, CI: false},
			want: true,
		},
		{
			name: "interactive empty args CI false",
			env:  tui.LaunchEnv{Interactive: true, Args: []string{}, CI: false},
			want: true,
		},
		{
			name: "interactive with plan subcommand",
			env:  tui.LaunchEnv{Interactive: true, Args: []string{"plan"}, CI: false},
			want: false,
		},
		{
			name: "interactive no args CI true",
			env:  tui.LaunchEnv{Interactive: true, Args: nil, CI: true},
			want: false,
		},
		{
			name: "interactive args tui is not auto-launch",
			env:  tui.LaunchEnv{Interactive: true, Args: []string{"tui"}, CI: false},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tui.ShouldAutoLaunch(tt.env)
			if got != tt.want {
				t.Fatalf("ShouldAutoLaunch(%+v) = %v, want %v", tt.env, got, tt.want)
			}
		})
	}
}
