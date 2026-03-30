package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestApp_RunAndShutdown(t *testing.T) {
	t.Parallel()

	app := New(Config{
		Username: "TestUser",
		Port:     0, // random port
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run()
	}()

	// Даем приложению запуститься
	time.Sleep(100 * time.Millisecond)

	// Инициируем остановку
	app.Shutdown()

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("App.Run() did not return after Shutdown()")
	}
}
