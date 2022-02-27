// Package proxy is core logic shared by client and server
package proxy

import (
	"io"
	"log"
	"net"
	"sync/atomic"

	"github.com/gowsp/wsp/pkg/msg"
)

type Channel interface {
	Transport(data *msg.Data) error
	Interrupt() error
	io.Closer
}

func NewTCPChannel(wan io.WriteCloser, conn net.Conn, onClose func()) *TCPChannel {
	return &TCPChannel{conn, wan, onClose, 0}
}

type TCPChannel struct {
	conn    net.Conn
	wan     io.WriteCloser
	onClose func()
	closed  uint32
}

func (r *TCPChannel) Copy() error {
	return r.CopyBy(r.conn)
}

func (r *TCPChannel) CopyBy(reader io.Reader) error {
	buf := bytePool.Get().(*[]byte)
	defer bytePool.Put(buf)

	_, err := io.CopyBuffer(r.wan, reader, *buf)
	if r.IsClosed() {
		return nil
	}
	if err != nil {
		log.Println(err)
	}
	return r.Close()
}
func (r *TCPChannel) Transport(data *msg.Data) error {
	_, err := r.conn.Write(data.Payload())
	if err != nil {
		log.Println(err)
		return r.Close()
	}
	return err
}
func (r *TCPChannel) Interrupt() error {
	atomic.AddUint32(&r.closed, 1)
	return r.Close()
}
func (r *TCPChannel) IsClosed() bool {
	return atomic.LoadUint32(&r.closed) > 0
}
func (r *TCPChannel) NotClosed() bool {
	return atomic.LoadUint32(&r.closed) == 0
}
func (r *TCPChannel) Close() error {
	if r.NotClosed() {
		atomic.AddUint32(&r.closed, 1)
		r.wan.Close()
	}
	r.onClose()
	return r.conn.Close()
}

func NewWsRepeater(id string, output *Wan) Channel {
	return &WsRepeater{id: id, output: output}
}

type WsRepeater struct {
	id     string
	output *Wan
}

func (r *WsRepeater) Transport(data *msg.Data) error {
	_, err := r.output.Write(*data.Raw)
	return err
}
func (r *WsRepeater) Interrupt() error {
	return r.Close()
}
func (r *WsRepeater) Close() error {
	data := Wrap(r.id, msg.WspCmd_INTERRUPT, []byte{})
	_, err := r.output.Write(data)
	r.output.routing.Delete(r.id)
	return err
}
