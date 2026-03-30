package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_IsServer(t *testing.T) {
	tests := []struct {
		name        string
		peerAddress string
		want        bool
	}{
		{
			name:        "empty address means server mode (T-APP-01)",
			peerAddress: "",
			want:        true,
		},
		{
			name:        "with address means client mode (T-APP-02)",
			peerAddress: "localhost:50051",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{PeerAddress: tt.peerAddress}
			assert.Equal(t, tt.want, cfg.IsServer())
		})
	}
}
