package stream

import (
	"io"
	"net"
	"sync"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

func NewReadWriter(conn net.Conn) io.ReadWriter {
	return &wsReadWriter{
		conn: conn,
	}
}

type wsReadWriter struct {
	lock sync.Mutex
	conn net.Conn
}

func (w *wsReadWriter) Read(p []byte) (n int, err error) {
	val, err := wsutil.ReadClientBinary(w.conn)
	if err != nil {
		return len(val), err
	}
	copy(p, val)
	return len(val), err
}

func (w *wsReadWriter) Write(p []byte) (n int, err error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	err = wsutil.WriteMessage(w.conn, ws.StateServerSide, ws.OpBinary, p)
	if err == nil {
		n = len(p)
	}
	return
}
