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
		c.ListenDynamic(conf)
	}
}

var dynamic sync.Once

func (c *Wspc) ListenDynamic(conf *msg.WspConfig) {
	dynamic.Do(func() {
		switch conf.Scheme() {
		case "socks5":
			go c.ListenSocks5Proxy(conf)
		case "http":
			go c.ListenHttpProxy(conf)
		default:
			log.Println("Not supported", conf.Scheme())
			return
		}
	})
}

func (c *Wspc) ListenSocks5Proxy(conf *msg.WspConfig) {
	address := conf.Address()
	log.Println("listen socks5 proxy", address)
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
		go c.NewSocks5Conn(conf, conn)
	}
}
func (s *Wspc) NewSocks5Conn(conf *msg.WspConfig, conn net.Conn) {
	if err := s.auth(conn); err != nil {
		conn.Close()
		log.Println("auth error:", err)
		return
	}
	addr, err := s.readSocks5Addr(conn)
	if err != nil {
		conn.Close()
		log.Println("connect error:", err)
		return
	}
	connConf := conf.DynamicAddr(addr)
	s.dynamic(connConf, conn)
}

func (s *Wspc) auth(conn net.Conn) (err error) {
	header := make([]byte, 3)
	if _, err := io.ReadAtLeast(conn, header, 3); err != nil {
		return fmt.Errorf("failed to get command version: %v", err)
	}
	_, err = conn.Write([]byte{0x05, 0x00})
	return err
}

func (s *Wspc) readSocks5Addr(conn net.Conn) (string, error) {
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

func (c *Wspc) dynamic(conf *msg.WspConfig, conn net.Conn) {
	addr := conf.Address()
	log.Println("open socks5 proxy", addr)
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
	if err := c.wan.Dail(id, conf); err != nil {
		conn.Write([]byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		log.Printf("close socks5 proxy %s, %s\n", addr, err.Error())
		c.routing.DeleteConn(id)
		conn.Close()
	}
}
