package client

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"sync/atomic"

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
			go socks5.process(conn)
		}
	})
}

func (s *Socks5Listener) process(client net.Conn) {
	defer client.Close()
	if err := s.auth(client); err != nil {
		log.Println("auth error:", err)
		return
	}
	addr, err := s.readAddr(client)
	if err != nil {
		log.Println("connect error:", err)
		return
	}
	s.c.Socks5Conn(client, addr)
}

func (s *Socks5Listener) auth(r net.Conn) (err error) {
	header := make([]byte, 3)
	if _, err := io.ReadAtLeast(r, header, 3); err != nil {
		return fmt.Errorf("failed to get command version: %v", err)
	}
	_, err = r.Write([]byte{0x05, 0x00})
	return err
}

func (s *Socks5Listener) readAddr(client net.Conn) (string, error) {
	header := make([]byte, 4)
	if _, err := io.ReadAtLeast(client, header, 4); err != nil {
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
		if _, err := client.Read(addr); err != nil {
			return "", err
		}
		host = net.IP(addr).String()
	case 4:
		addr := make([]byte, 16)
		if _, err := client.Read(addr); err != nil {
			return "", err
		}
		host = net.IP(addr).String()
	case 3:
		addrLen := []byte{0}
		if _, err := client.Read(addrLen); err != nil {
			return "", err
		}
		hostName := make([]byte, int(addrLen[0]))
		if _, err := client.Read(hostName); err != nil {
			return "", err
		}
		host = string(hostName)
	default:
		return "", fmt.Errorf("unrecognized address type")
	}

	portd := []byte{0, 0}
	if _, err := client.Read(portd); err != nil {
		return "", err
	}
	port := binary.BigEndian.Uint16(portd[:2])
	return net.JoinHostPort(host, strconv.Itoa(int(port))), nil
}

func (c *Wspc) Socks5Conn(out net.Conn, addr string) {
	log.Println("open proxy", addr)
	id := ksuid.New().String()

	in := c.wan.NewWriter(id)
	bridge := NewSocke5Bridge(in, out)
	c.lan.Store(id, bridge)
	defer c.lan.Delete(id)

	if err := c.wan.Dail(id, msg.WspType_SOCKS5, addr); err != nil {
		log.Println(err)
		return
	}
	bridge.Forward()
	log.Println("close connect", addr)
}

type Socke5Bridge struct {
	in      io.WriteCloser
	out     io.ReadWriteCloser
	closed  uint32
	closeIn chan bool
}

func NewSocke5Bridge(in io.WriteCloser, out io.ReadWriteCloser) pkg.Bridge {
	return &Socke5Bridge{
		in:      in,
		out:     out,
		closeIn: make(chan bool),
	}
}
func (b *Socke5Bridge) Read(message *msg.WspMessage) bool {
	if b.IsClosed() {
		return false
	}
	switch message.Cmd {
	case msg.WspCmd_CONN_REP:
		if message.Data[0] == 0 {
			b.out.Write([]byte{0x05, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
			b.closeIn <- false
			return true
		}
		b.out.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		go func() {
			io.Copy(b.in, b.out)
			b.closeIn <- true
		}()
	case msg.WspCmd_FORWARD:
		_, err := b.out.Write(message.Data)
		if err != nil {
			b.closeIn <- true
		}
	case msg.WspCmd_CLOSE:
		b.closeIn <- false
	}
	return true
}
func (b *Socke5Bridge) IsClosed() bool {
	return atomic.LoadUint32(&b.closed) > 0
}
func (b *Socke5Bridge) Close() {
	b.closeIn <- false
}
func (b *Socke5Bridge) Forward() {
	closeIn := <-b.closeIn
	atomic.AddUint32(&b.closed, 1)
	if closeIn {
		b.in.Close()
	}
}
