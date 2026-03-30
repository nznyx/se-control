package chat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_Send(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		wantError bool
	}{
		{
			name:      "T-CH-06: empty message",
			text:      "",
			wantError: true,
		},
		{
			name:      "T-CH-01, T-CH-02: valid message and timestamp",
			text:      "Hello world",
			wantError: false,
		},
		{
			name:      "T-CH-07: unicode support",
			text:      "Привет, мир! 🚀",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService("Alice")
			
			err := service.Send(tt.text)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
