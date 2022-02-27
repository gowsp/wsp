package client

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/gowsp/wsp/pkg/msg"
	"github.com/segmentio/ksuid"
)

// Socks5Proxy implement DynamicProxy
type Socks5Proxy struct {
	conf *msg.WspConfig
	wspc *Wspc
}

func (p *Socks5Proxy) Listen() {
	address := p.conf.Address()
	log.Println("listen socks5 proxy", address)
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

func (p *Socks5Proxy) ServeConn(conn net.Conn) {
	reader := bufio.NewReader(conn)
	if err := p.auth(reader, conn); err != nil {
		conn.Close()
		log.Println("auth error:", err)
		return
	}
	addr, err := p.readRequest(reader)
	if err != nil {
		conn.Close()
		log.Println("connect error:", err)
		return
	}
	p.replies(addr, reader, conn)
}

func (p *Socks5Proxy) checkVer(reader *bufio.Reader) error {
	ver, err := reader.ReadByte()
	if err != nil {
		return err
	}
	if ver != 0x05 {
		return fmt.Errorf("unsupported socks version %d", ver)
	}
	return nil
}
func (p *Socks5Proxy) auth(reader *bufio.Reader, writer io.Writer) error {
	// +----+----------+----------+
	// |VER | NMETHODS | METHODS  |
	// +----+----------+----------+
	// | 1  |    1     | 1 to 255 |
	// +----+----------+----------+
	if err := p.checkVer(reader); err != nil {
		return err
	}
	nmethods, err := reader.ReadByte()
	if err != nil {
		return err
	}
	methods := make([]byte, nmethods)
	_, err = io.ReadFull(reader, methods)
	if err != nil {
		return err
	}
	// +----+--------+
	// |VER | METHOD |
	// +----+--------+
	// | 1  |   1    |
	// +----+--------+
	_, err = writer.Write([]byte{0x05, 0x00})
	if err != nil {
		return err
	}
	return nil
}

func (p *Socks5Proxy) readRequest(conn *bufio.Reader) (addr string, err error) {
	// +----+-----+-------+------+----------+----------+
	// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+
	if err = p.checkVer(conn); err != nil {
		return "", err
	}
	buf := make([]byte, 3)
	if _, err = io.ReadFull(conn, buf); err != nil {
		return "", err
	}
	var host string
	addrType := buf[2]
	switch addrType {
	case 1:
		host, err = p.readIP(conn, net.IPv4len)
		if err != nil {
			return "", err
		}
	case 4:
		host, err = p.readIP(conn, net.IPv6len)
		if err != nil {
			return "", err
		}
	case 3:
		addrLen, err := conn.ReadByte()
		if err != nil {
			return "", err
		}
		hostName := make([]byte, addrLen)
		if _, err := io.ReadFull(conn, hostName); err != nil {
			return "", err
		}
		host = string(hostName)
	default:
		return "", fmt.Errorf("unrecognized address type")
	}

	portd := []byte{0, 0}
	if _, err := io.ReadFull(conn, portd); err != nil {
		return "", err
	}
	port := binary.BigEndian.Uint16(portd)
	return net.JoinHostPort(host, fmt.Sprintf("%d", port)), nil
}
func (p *Socks5Proxy) readIP(conn *bufio.Reader, len byte) (string, error) {
	addr := make([]byte, len)
	if _, err := io.ReadFull(conn, addr); err != nil {
		return "", err
	}
	return net.IP(addr).String(), nil
}
func (p *Socks5Proxy) replies(addr string, reader io.Reader, conn net.Conn) {
	// +----+-----+-------+------+----------+----------+
	// |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+
	c := p.wspc
	conf := p.conf.DynamicAddr(addr)
	log.Println("open socks5 proxy", addr)
	id := ksuid.New().String()

	trans := func(data *msg.Data, message *msg.WspResponse) {
		if message.Code == msg.WspCode_FAILED {
			conn.Write([]byte{0x05, 0x05, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
			log.Printf("close socks5 proxy %s, %s\n", addr, message.Data)
			conn.Close()
			return
		}
		_, repeater := c.wan.NewTCPChannel(id, conn)
		conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		repeater.CopyBy(reader)
		log.Println("close socks5 proxy", addr)
	}
	if err := c.wan.Dail(id, conf, trans); err != nil {
		conn.Write([]byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		log.Printf("close socks5 proxy %s, %s\n", addr, err.Error())
		conn.Close()
	}
}
