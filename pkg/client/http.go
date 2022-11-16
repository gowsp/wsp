package client

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/gowsp/wsp/pkg/msg"
)

// HTTPProxy implement DynamicProxy
type HTTPProxy struct {
	conf *msg.WspConfig
	wspc *Wspc
}

func (p *HTTPProxy) Listen() {
	address := p.conf.Address()
	log.Println("listen http proxy", address)
	l, err := net.Listen(p.conf.Network(), address)
	if err != nil {
		log.Println(err)
		return
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
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
	log.Println("open http proxy", addr)
	config := p.conf.DynamicAddr(addr)
	remote, err := p.wspc.wan.DialTCP(local, config)
	if err != nil {
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
	io.Copy(remote, local)
	log.Println("close http proxy", addr)
	return nil
}
