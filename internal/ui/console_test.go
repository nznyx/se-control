package ui

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nznyx/se-control/internal/chat"
)

func TestConsole_ReadInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "read simple string (T-UI-03)",
			input:   "hello world\n",
			want:    "hello world",
			wantErr: false,
		},
		{
			name:    "read string with windows newlines",
			input:   "hello windows\r\n",
			want:    "hello windows",
			wantErr: false,
		},
		{
			name:    "read EOF",
			input:   "",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			writer := &bytes.Buffer{}
			console := New(reader, writer)

			got, err := console.ReadInput()
			if tt.wantErr {
				assert.ErrorIs(t, err, io.EOF)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestConsole_DisplayMessage(t *testing.T) {
	fixedTime := time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC)
	msg := chat.Message{
		Sender:    "Alice",
		Text:      "Hello, Bob!",
		Timestamp: fixedTime,
	}

	reader := strings.NewReader("")
	writer := &bytes.Buffer{}
	console := New(reader, writer)

	console.DisplayMessage(msg)

	want := "[2026-03-30 12:00:00] Alice: Hello, Bob!\n"
	assert.Equal(t, want, writer.String(), "T-UI-01 and T-UI-04: message format check")
}

func TestConsole_DisplaySystem(t *testing.T) {
	reader := strings.NewReader("")
	writer := &bytes.Buffer{}
	console := New(reader, writer)

	console.DisplaySystem("Connecting to peer...")

	want := "*** Connecting to peer... ***\n"
	assert.Equal(t, want, writer.String(), "T-UI-02: system message format check")
}
