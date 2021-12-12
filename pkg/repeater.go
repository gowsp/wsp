package pkg

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
	conn      net.Conn
	wan       io.WriteCloser
	interrupt uint32
}

func (r *NetRepeater) Copy() error {
	_, err := io.Copy(r.wan, r.conn)
	if err != nil {
		log.Println(err)
	}
	return r.Close()
}
func (r *NetRepeater) Relay(data *msg.Data) error {
	_, err := r.conn.Write(data.Payload())
	return err
}
func (r *NetRepeater) Interrupt() error {
	atomic.AddUint32(&r.interrupt, 1)
	return r.conn.Close()
}
func (r *NetRepeater) Close() error {
	if atomic.LoadUint32(&r.interrupt) == 0 {
		r.wan.Close()
	}
	return r.conn.Close()
}

func NewWsRepeater(id string, output io.Writer) Repeater {
	return &WsRepeater{id, output}
}

type WsRepeater struct {
	id     string
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
	data := Wrap(r.id, msg.WspCmd_CLOSE, []byte{})
	_, err := r.output.Write(data)
	return err
}
