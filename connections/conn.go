package connections

// Wrap websocket.Conn to interface
// Useful for test
type Connection interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	Close() error
}
