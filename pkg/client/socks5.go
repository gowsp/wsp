package client

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/proxy"
	"github.com/segmentio/ksuid"
)

func (c *Wspc) DynamicForward() {
	for _, val := range c.Config.Dynamic {
		conf, err := msg.NewWspConfig(msg.WspType_DYNAMIC, val)
		if err != nil {
			log.Println("forward dynamic error,", err)
			continue
		}
		go c.ListenDynamic(conf)
	}
}

var socks5 sync.Once

type Socks5Listener struct {
	c *Wspc
}

func (c *Wspc) ListenDynamic(conf *msg.WspConfig) {
	socks5.Do(func() {
		addr := conf.Scheme()
		if addr != "socks5" {
			log.Println("Not supported", addr)
			return
		}
		address := conf.Address()
		log.Println("listen socks5", address)
		l, err := net.Listen(conf.Network(), address)
		if err != nil {
			log.Println(err)
			return
		}
		socks5 := Socks5Listener{c}
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Println(err)
				continue
			}
			socks5.process(conn)
		}
	})
}

func (s *Socks5Listener) process(conn net.Conn) {
	if err := s.auth(conn); err != nil {
		conn.Close()
		log.Println("auth error:", err)
		return
	}
	addr, err := s.readAddr(conn)
	if err != nil {
		conn.Close()
		log.Println("connect error:", err)
		return
	}
	s.c.dynamic(conn, addr)
}

func (s *Socks5Listener) auth(conn net.Conn) (err error) {
	header := make([]byte, 3)
	if _, err := io.ReadAtLeast(conn, header, 3); err != nil {
		return fmt.Errorf("failed to get command version: %v", err)
	}
	_, err = conn.Write([]byte{0x05, 0x00})
	return err
}

func (s *Socks5Listener) readAddr(conn net.Conn) (string, error) {
	header := make([]byte, 4)
	if _, err := io.ReadAtLeast(conn, header, 4); err != nil {
		return "", fmt.Errorf("failed to get version: %v", err)
	}
	if header[0] != 5 {
		return "", fmt.Errorf("unsupported version: %d", header[0])
	}

	var host string
	addrType := header[3]
	switch addrType {
	case 1:
		addr := header
		if _, err := conn.Read(addr); err != nil {
			return "", err
		}
		host = net.IP(addr).String()
	case 4:
		addr := make([]byte, 16)
		if _, err := conn.Read(addr); err != nil {
			return "", err
		}
		host = net.IP(addr).String()
	case 3:
		addrLen := []byte{0}
		if _, err := conn.Read(addrLen); err != nil {
			return "", err
		}
		hostName := make([]byte, int(addrLen[0]))
		if _, err := conn.Read(hostName); err != nil {
			return "", err
		}
		host = string(hostName)
	default:
		return "", fmt.Errorf("unrecognized address type")
	}

	portd := []byte{0, 0}
	if _, err := conn.Read(portd); err != nil {
		return "", err
	}
	port := binary.BigEndian.Uint16(portd[:2])
	return net.JoinHostPort(host, fmt.Sprintf("%d", port)), nil
}

func (c *Wspc) dynamic(conn net.Conn, addr string) {
	log.Println("open proxy", addr)
	id := ksuid.New().String()

	c.routing.AddPending(id, &proxy.Pending{OnReponse: func(data *msg.Data, message *msg.WspResponse) {
		if message.Code == msg.WspCode_FAILED {
			conn.Write([]byte{0x05, 0x05, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
			log.Printf("close socks5 proxy %s, %s\n", addr, message.Data)
			conn.Close()
			return
		}
		in := c.wan.NewWriter(id)
		repeater := proxy.NewNetRepeater(in, conn)
		c.routing.AddRepeater(id, repeater)
		defer c.routing.Delete(id)
		conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		repeater.Copy()
		log.Println("close socks5 proxy", addr)
	}})
	config, _ := msg.NewWspConfig(msg.WspType_DYNAMIC, "tcp://"+addr)
	if err := c.wan.Dail(id, config); err != nil {
		conn.Write([]byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		log.Printf("close proxy %s, %s\n", addr, err.Error())
		c.routing.DeleteConn(id)
		conn.Close()
	}
}
