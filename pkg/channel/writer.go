package channel

import (
	"io"
	"net"

	"github.com/gowsp/wsp/pkg/msg"
)

type writer struct {
	id      string
	channel *Channel
}

func (w *writer) Write(p []byte) (n int, err error) {
	data := encode(w.id, msg.WspCmd_TRANSFER, p)
	_, err = w.channel.Write(data)
	return len(p), err
}

type Writer interface {
	Transport(data *msg.Data) error

	io.Closer
}

type TCPWriter struct {
	conn net.Conn
}

func (w *TCPWriter) Transport(data *msg.Data) error {
	_, err := w.conn.Write(data.Payload())
	return err
}
func (l *TCPWriter) Close() error {
	return l.conn.Close()
}

type WsWriter struct {
	id      string
	channel *Channel
}

func (w *WsWriter) Transport(data *msg.Data) error {
	_, err := w.channel.Write(*data.Raw)
	return err
}
func (w *WsWriter) Close() error {
	return nil
}
