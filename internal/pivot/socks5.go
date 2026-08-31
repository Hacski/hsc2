package pivot

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"time"
)

const (
	socks5Ver  = 0x05
	cmdConnect = 0x01
	cmdBind    = 0x02
	cmdUDP     = 0x03
	atypIPv4   = 0x01
	atypDomain = 0x03
	atypIPv6   = 0x04
	repSuccess = 0x00
	repFailure = 0x01
	repConnRef = 0x05
)

type SOCKS5Server struct {
	table *Table
}

func NewSOCKS5(table *Table) *SOCKS5Server {
	return &SOCKS5Server{table: table}
}

func (s *SOCKS5Server) Serve(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		go s.handle(conn)
	}
}

func (s *SOCKS5Server) handle(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(30 * time.Second))
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return
	}
	if buf[0] != socks5Ver {
		return
	}
	nMethods := int(buf[1])
	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return
	}
	// no-auth (0x00)
	conn.Write([]byte{socks5Ver, 0x00})

	head := make([]byte, 4)
	if _, err := io.ReadFull(conn, head); err != nil {
		return
	}
	if head[0] != socks5Ver {
		return
	}
	cmd := head[1]
	atyp := head[3]
	var target string
	switch atyp {
	case atypIPv4:
		ip := make([]byte, 4)
		io.ReadFull(conn, ip)
		pb := make([]byte, 2)
		io.ReadFull(conn, pb)
		target = net.IP(ip).String() + ":" + itoa(uint16(binary.BigEndian.Uint16(pb)))
	case atypIPv6:
		ip := make([]byte, 16)
		io.ReadFull(conn, ip)
		pb := make([]byte, 2)
		io.ReadFull(conn, pb)
		target = "[" + net.IP(ip).String() + "]:" + itoa(uint16(binary.BigEndian.Uint16(pb)))
	case atypDomain:
		lb := make([]byte, 1)
		io.ReadFull(conn, lb)
		dom := make([]byte, lb[0])
		io.ReadFull(conn, dom)
		pb := make([]byte, 2)
		io.ReadFull(conn, pb)
		target = string(dom) + ":" + itoa(uint16(binary.BigEndian.Uint16(pb)))
	default:
		conn.Write([]byte{socks5Ver, repFailure, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}
	if cmd == cmdUDP {
		conn.Write([]byte{socks5Ver, repFailure, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}
	if cmd != cmdConnect {
		conn.Write([]byte{socks5Ver, repFailure, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}

	dialer, _, ok := s.table.Resolve(target)
	if !ok {
		conn.Write([]byte{socks5Ver, repConnRef, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}
	up, err := dialer.Dial(context.Background(), target)
	if err != nil {
		conn.Write([]byte{socks5Ver, repFailure, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}
	defer up.Close()
	// success reply
	conn.Write([]byte{socks5Ver, repSuccess, 0, 1, 0, 0, 0, 0, 0, 0})
	conn.SetDeadline(time.Time{})
	relay(conn, up)
}

func relay(a, b net.Conn) {
	done := make(chan struct{}, 2)
	cp := func(dst, src net.Conn) {
		io.Copy(dst, src)
		if c, ok := dst.(interface{ CloseWrite() error }); ok {
			c.CloseWrite()
		}
		done <- struct{}{}
	}
	go cp(b, a)
	go cp(a, b)
	<-done
}

func itoa(v uint16) string {
	if v == 0 {
		return "0"
	}
	var b [5]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}
