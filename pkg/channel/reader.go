package channel

import (
	"io"
	"net"
)

type Reader interface {
	Copy() error
}

type TCPReader struct {
	output io.Writer
	input  net.Conn
}

func (r *TCPReader) Copy() error {
	io.Copy(r.output, r.input)
	return nil
}
