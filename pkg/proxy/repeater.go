package proxy

import (
	"io"
	"log"
	"net"
	"sync/atomic"

	"github.com/gowsp/wsp/pkg/msg"
)

type Repeater interface {
	Relay(data *msg.Data) error
	Interrupt() error
	io.Closer
}

func NewNetRepeater(wan io.WriteCloser, conn net.Conn) *NetRepeater {
	return &NetRepeater{conn, wan, 0}
}

type NetRepeater struct {
	conn   net.Conn
	wan    io.WriteCloser
	closed uint32
}

func (r *NetRepeater) Copy() error {
	buf := bytePool.Get().(*[]byte)
	defer bytePool.Put(buf)

	_, err := io.CopyBuffer(r.wan, r.conn, *buf)
	if r.IsClosed() {
		return nil
	}
	if err != nil {
		log.Println(err)
	}
	return r.Close()
}
func (r *NetRepeater) Relay(data *msg.Data) error {
	_, err := r.conn.Write(data.Payload())
	if err != nil {
		log.Println(err)
		return r.Close()
	}
	return err
}
func (r *NetRepeater) Interrupt() error {
	atomic.AddUint32(&r.closed, 1)
	return r.Close()
}
func (r *NetRepeater) IsClosed() bool {
	return atomic.LoadUint32(&r.closed) > 0
}
func (r *NetRepeater) NotClosed() bool {
	return atomic.LoadUint32(&r.closed) == 0
}
func (r *NetRepeater) Close() error {
	if r.NotClosed() {
		atomic.AddUint32(&r.closed, 1)
		r.wan.Close()
	}
	return r.conn.Close()
}

func NewWsRepeater(id string, output io.Writer) Repeater {
	return &WsRepeater{id: id, output: output}
}

type WsRepeater struct {
	id     string
	input  io.Writer
	output io.Writer
}

func (r *WsRepeater) Relay(data *msg.Data) error {
	_, err := r.output.Write(*data.Raw)
	return err
}
func (r *WsRepeater) Interrupt() error {
	return r.Close()
}
func (r *WsRepeater) Close() error {
	data := Wrap(r.id, msg.WspCmd_INTERRUPT, []byte{})
	_, err := r.output.Write(data)
	return err
}
