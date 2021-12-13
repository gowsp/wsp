package client

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"

	"github.com/gowsp/wsp/pkg"
	"github.com/gowsp/wsp/pkg/msg"
	"github.com/segmentio/ksuid"
)

var socks5 sync.Once

type Socks5Listener struct {
	c *Wspc
}

func (c *Wspc) ListenSocks5() {
	socks5.Do(func() {
		l, err := net.Listen("tcp", c.Config.Socks5)
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
	s.c.Socks5Conn(conn, addr)
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
	return net.JoinHostPort(host, strconv.Itoa(int(port))), nil
}

func (c *Wspc) Socks5Conn(conn net.Conn, addr string) {
	log.Println("open proxy", addr)
	id := ksuid.New().String()

	in := c.wan.NewWriter(id)
	repeater := pkg.NewNetRepeater(in, conn)
	c.routing.AddRepeater(id, repeater)

	if err := c.wan.Dail(id, msg.WspType_SOCKS5, addr); err != nil {
		repeater.Interrupt()
		c.routing.Delete(id)
		log.Println(err)
		return
	}
	c.routing.AddPending(id, &pkg.Pending{OnReponse: func(message *msg.Data) {
		defer c.routing.Delete(id)
		defer log.Println("close conn", addr)
		if message.Payload()[0] == 0 {
			conn.Write([]byte{0x05, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
			repeater.Interrupt()
			return
		}
		conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		repeater.Copy()
	}})
}
