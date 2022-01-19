package client

import (
	"bufio"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/proxy"
	"github.com/segmentio/ksuid"
)

func (c *Wspc) ListenHttpProxy(conf *msg.WspConfig) {
	address := conf.Address()
	log.Println("listen http proxy", address)
	l, err := net.Listen(conf.Network(), address)
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
		go c.NewHttpProxyConn(conf, conn)
	}
}
func (c *Wspc) NewHttpProxyConn(conf *msg.WspConfig, conn net.Conn) {
	buffer := proxy.GetBuffer()
	reader := bufio.NewReader(io.TeeReader(conn, buffer))
	reqeust, err := http.ReadRequest(reader)
	if err != nil {
		log.Println(err)
		return
	}
	addr := reqeust.URL.Host
	isConnect := reqeust.Method == http.MethodConnect
	if strings.LastIndexByte(reqeust.URL.Host, ':') == -1 {
		addr = addr + ":80"
	}
	log.Println("open http proxy", addr)
	id := ksuid.New().String()

	c.routing.AddPending(id, &proxy.Pending{OnReponse: func(data *msg.Data, message *msg.WspResponse) {
		if message.Code == msg.WspCode_FAILED {
			proxy.PutBuffer(buffer)
			conn.Write([]byte("HTTP/1.1 500\r\n\r\n"))
			log.Printf("close http proxy %s, %s\n", addr, message.Data)
			conn.Close()
			return
		}
		in := c.wan.NewWriter(id)
		repeater := proxy.NewNetRepeater(in, conn)
		c.routing.AddRepeater(id, repeater)
		defer c.routing.Delete(id)
		if isConnect {
			conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		} else {
			in.Write(buffer.Bytes())
		}
		proxy.PutBuffer(buffer)
		repeater.Copy()
		log.Println("close http proxy", addr)
	}})
	config := conf.DynamicAddr(addr)
	if err := c.wan.Dail(id, config); err != nil {
		proxy.PutBuffer(buffer)
		conn.Write([]byte("HTTP/1.1 500\r\n\r\n"))
		log.Printf("close http proxy %s, %s\n", addr, err.Error())
		c.routing.DeleteConn(id)
		conn.Close()
	}
}
