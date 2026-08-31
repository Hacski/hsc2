package transport

import (
	"bytes"
	"context"
	"encoding/base32"
	"encoding/binary"
	"net"
	"strings"
	"sync"
)

type DNSListener struct {
	name string
	conn *net.UDPConn
	mu   sync.Mutex
}

const maxDNSLabel = 63

func NewDNSListener(name string) *DNSListener {
	return &DNSListener{name: name}
}

func (d *DNSListener) Name() string { return d.name }

func (d *DNSListener) Listen(ctx context.Context, addr string, handler Handler) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	d.conn = conn
	buf := make([]byte, 65535)
	for {
		select {
		case <-ctx.Done():
			return conn.Close()
		default:
		}
		n, raddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		go d.handle(conn, raddr, buf[:n], handler)
	}
}

func (d *DNSListener) handle(conn *net.UDPConn, raddr *net.UDPAddr, packet []byte, handler Handler) {
	if len(packet) < 12 {
		return
	}
	qname := parseQName(packet[12:])
	payload := decodeQNamePayload(qname)
	if payload == nil {
		return
	}
	resp, err := handler.Handle(context.Background(), payload)
	if err != nil {
		return
	}
	reply := encodeTXTResponse(packet[:12], qname, resp)
	conn.WriteToUDP(reply, raddr)
}

func (d *DNSListener) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}

func parseQName(msg []byte) []byte {
	i := 0
	for i < len(msg) && msg[i] != 0 {
		l := int(msg[i])
		i++
		if i+l > len(msg) {
			return nil
		}
		i += l
	}
	return msg[:i]
}

func decodeQNamePayload(qname []byte) []byte {
	if qname == nil || len(qname) == 0 {
		return nil
	}
	var parts []byte
	i := 0
	for i < len(qname) && qname[i] != 0 {
		l := int(qname[i])
		i++
		parts = append(parts, qname[i:i+l]...)
		i += l
	}
	if len(parts) == 0 {
		return nil
	}
	str, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(string(parts)))
	if err != nil {
		return nil
	}
	return str
}

func encodeTXTResponse(header, qname, payload []byte) []byte {
	var buf bytes.Buffer
	buf.Write(header[:2])
	buf.Write([]byte{0x81, 0x80})
	binary.Write(&buf, binary.BigEndian, uint16(1))
	binary.Write(&buf, binary.BigEndian, uint16(1))
	binary.Write(&buf, binary.BigEndian, uint16(0))
	binary.Write(&buf, binary.BigEndian, uint16(0))
	buf.Write(qname)
	buf.WriteByte(0)
	binary.Write(&buf, binary.BigEndian, uint16(16))
	binary.Write(&buf, binary.BigEndian, uint16(1))
	binary.Write(&buf, binary.BigEndian, uint32(60))
	binary.Write(&buf, binary.BigEndian, uint16(len(payload)))
	buf.Write(payload)
	return buf.Bytes()
}
