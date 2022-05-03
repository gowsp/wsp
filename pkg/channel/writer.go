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

func (c *Channel) NewWriter(id string) io.Writer {
	return &writer{id: id, channel: c}
}
func (c *Channel) NewWsWriter(id string) Writer {
	return &WsWriter{id: id, output: c}
}
func (w *writer) Write(p []byte) (n int, err error) {
	data := encode(w.id, msg.WspCmd_TRANSFER, p)
	_, err = w.channel.Write(data)
	return len(p), err
}

type Writer interface {
	Transport(data *msg.Data) error

	io.WriteCloser
}

type TCPWriter struct {
	conn net.Conn
	ws   io.Writer
}

func NewTcpWriter(conn net.Conn, ws io.Writer) Writer {
	return &TCPWriter{conn: conn, ws: ws}
}
func (w *TCPWriter) Transport(data *msg.Data) error {
	_, err := w.conn.Write(data.Payload())
	return err
}
func (l *TCPWriter) Write(p []byte) (n int, err error) {
	return l.ws.Write(p)
}
func (l *TCPWriter) Close() error {
	return l.conn.Close()
}

type WsWriter struct {
	id     string
	output *Channel
}

func (w *WsWriter) Transport(data *msg.Data) error {
	_, err := w.output.Write(*data.Raw)
	return err
}
func (w *WsWriter) Write(p []byte) (n int, err error) {
	return w.output.Write(p)
}
func (w *WsWriter) Close() error {
	w.output.session.Delete(w.id)
	data := encode(w.id, msg.WspCmd_INTERRUPT, []byte{})
	_, err := w.output.Write(data)
	return err
}
