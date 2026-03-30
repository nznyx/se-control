// Package client — gRPC клиент для подключения к peer-у.
package client

// Client — gRPC клиент для установки соединения и обмена сообщениями.
type Client struct {
	address string
}

// New создаёт новый экземпляр Client.
func New(address string) *Client {
	return &Client{
		address: address,
	}
}

// Connect устанавливает соединение с peer-ом.
func (c *Client) Connect() error {
	// TODO: реализовать подключение к gRPC серверу.
	return nil
}

// Close корректно закрывает соединение.
func (c *Client) Close() {
	// TODO: реализовать закрытие gRPC соединения.
}
