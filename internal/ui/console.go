// Package ui — консольный пользовательский интерфейс.
// Отвечает за чтение ввода и форматированный вывод сообщений.
package ui

import (
	"bufio"
	"fmt"
	"io"
	"time"

	pb "github.com/nznyx/se-control/pkg/proto/chat"
)

// Console — консольный интерфейс для взаимодействия с пользователем.
type Console struct {
	reader *bufio.Reader
	writer io.Writer
}

// New создаёт новый экземпляр Console.
func New(reader io.Reader, writer io.Writer) *Console {
	return &Console{
		reader: bufio.NewReader(reader),
		writer: writer,
	}
}

// ReadInput читает строку пользовательского ввода из stdin.
func (c *Console) ReadInput() (string, error) {
	// TODO: реализовать чтение ввода.
	return "", nil
}

// DisplayMessage выводит отформатированное сообщение в stdout.
// Формат: [2026-03-30 12:00:00] Alice: Hello
func (c *Console) DisplayMessage(msg *pb.ChatMessage) {
	t := time.Unix(msg.GetTimestamp(), 0)
	fmt.Fprintf(c.writer, "[%s] %s: %s\n",
		t.Format("2006-01-02 15:04:05"),
		msg.GetSender(),
		msg.GetText(),
	)
}

// DisplaySystem выводит системное сообщение.
func (c *Console) DisplaySystem(text string) {
	fmt.Fprintf(c.writer, "*** %s ***\n", text)
}
