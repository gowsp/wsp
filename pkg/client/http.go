package client

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/gowsp/wsp/pkg/logger"
	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/stream"
)

// HTTPProxy implement DynamicProxy
type HTTPProxy struct {
	conf *msg.WspConfig
	wspc *Wspc
}

func (p *HTTPProxy) Listen() {
	address := p.conf.Address()
	logger.Info("listen http proxy %s", address)
	l, err := net.Listen(p.conf.Network(), address)
	if err != nil {
		logger.Error("listen http proxy error: %s", err)
		return
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			logger.Error("http proxy accept %s", err)
			continue
		}
		go p.ServeConn(conn)
	}
}

func (p *HTTPProxy) ServeConn(local net.Conn) error {
	buffer := new(bytes.Buffer)
	reader := bufio.NewReader(io.TeeReader(local, buffer))
	reqeust, err := http.ReadRequest(reader)
	if err != nil {
		return err
	}
	addr := reqeust.URL.Host
	ssl := reqeust.Method == http.MethodConnect
	if strings.LastIndexByte(reqeust.URL.Host, ':') == -1 {
		addr = addr + ":80"
	}
	logger.Info("open http proxy %s", addr)
	config := p.conf.DynamicAddr(addr)
	remote, err := p.wspc.wan.DialTCP(local.LocalAddr(), config)
	if err != nil {
		logger.Error("open http proxy %s error:", addr, err)
		local.Write([]byte("HTTP/1.1 500\r\n\r\n"))
		return err
	}
	defer remote.Close()
	if ssl {
		local.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	} else {
		remote.Write(buffer.Bytes())
	}
	buffer.Reset()
	stream.Copy(local, remote)
	logger.Info("close http proxy %s", addr)
	return nil
}
