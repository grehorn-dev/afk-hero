package appmeta

import (
	"afk-hero/internal/domain"
	"testing"
)

func TestWindowHeight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		settings domain.Settings
		want     int
	}{
		{
			name: "base only",
			settings: domain.Settings{
				Advanced:          false,
				ActivationEnabled: false,
			},
			want: WindowBaseHeight,
		},
		{
			name: "advanced only",
			settings: domain.Settings{
				Advanced:          true,
				ActivationEnabled: false,
			},
			want: WindowBaseHeight + WindowAdvancedHeight,
		},
		{
			name: "activation only",
			settings: domain.Settings{
				Advanced:          false,
				ActivationEnabled: true,
			},
			want: WindowBaseHeight + WindowActivationHeight,
		},
		{
			name: "advanced and activation",
			settings: domain.Settings{
				Advanced:          true,
				ActivationEnabled: true,
			},
			want: WindowBaseHeight + WindowAdvancedHeight + WindowActivationHeight,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := WindowHeight(tt.settings); got != tt.want {
				t.Fatalf("WindowHeight() = %d, want %d", got, tt.want)
			}
		})
	}
}
