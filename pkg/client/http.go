package client

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/gowsp/wsp/pkg/channel"
	"github.com/gowsp/wsp/pkg/msg"
	"github.com/segmentio/ksuid"
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

func (p *HTTPProxy) ServeConn(conn net.Conn) error {
	buffer := new(bytes.Buffer)
	reader := bufio.NewReader(io.TeeReader(conn, buffer))
	reqeust, err := http.ReadRequest(reader)
	if err != nil {
		return err
	}
	addr := reqeust.URL.Host
	ssl := reqeust.Method == http.MethodConnect
	if strings.LastIndexByte(reqeust.URL.Host, ':') == -1 {
		addr = addr + ":80"
	}
	conf := p.conf.DynamicAddr(addr)
	log.Println("open http proxy", addr)
	id := ksuid.New().String()
	l := &httpProxyLinker{addr: addr, conn: conn, buff: buffer, ssl: ssl}
	return p.wspc.channel.NewTcpSession(id, conf, l, conn).Syn()
}

type httpProxyLinker struct {
	addr string
	ssl  bool
	conn net.Conn
	buff *bytes.Buffer
}

func (l *httpProxyLinker) InActive(err error) {
	l.conn.Write([]byte("HTTP/1.1 500\r\n\r\n"))
	log.Println("close http proxy", l.addr, err.Error())
}
func (l *httpProxyLinker) Active(session *channel.Session) error {
	go func() {
		if l.ssl {
			l.conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		} else {
			session.Write(l.buff.Bytes())
		}
		session.CopyFrom(l.conn)
		log.Println("close http proxy", l.addr)
	}()
	return nil
}
