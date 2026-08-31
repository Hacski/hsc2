package pivot

import (
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

type directDialer struct{ target string }

func (d directDialer) Name() string { return "direct" }

func (d directDialer) Dial(ctx context.Context, target string) (net.Conn, error) {
	return net.Dial("tcp", d.target)
}

func startEcho(t *testing.T) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				io.Copy(c, c)
				c.Close()
			}(c)
		}
	}()
	return ln.Addr().String()
}

func TestRouteTableLongestPrefix(t *testing.T) {
	table := NewTable()
	defaultD := directDialer{target: "default-echo"}
	exactD := directDialer{target: "exact-echo"}
	table.Add("10.0.0.0/8", "sess-0", defaultD, 1)
	table.Add("10.1.2.3", "sess-1", exactD, 1)

	d, sess, ok := table.Resolve("10.1.2.3:80")
	if !ok {
		t.Fatal("expected route")
	}
	if d.Name() != "direct" || sess != "sess-1" {
		t.Fatalf("expected exact route, got %s/%s", d.Name(), sess)
	}

	d, sess, ok = table.Resolve("10.9.9.9:443")
	if !ok || sess != "sess-0" {
		t.Fatalf("expected default route, got %v/%v", ok, sess)
	}
}

func TestSOCKS5Connect(t *testing.T) {
	echoAddr := startEcho(t)
	table := NewTable()
	table.Add("127.0.0.1", "sess-1", directDialer{target: echoAddr}, 1)

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	srv := NewSOCKS5(table)
	go srv.Serve(ln)

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// send version/version identification, no auth
	conn.Write([]byte{0x05, 0x01, 0x00})
	greet := make([]byte, 2)
	if _, err := io.ReadFull(conn, greet); err != nil {
		t.Fatal(err)
	}
	if greet[1] != 0x00 {
		t.Fatalf("expected no-auth, got %d", greet[1])
	}

	// connect to an arbitrary internal target; route forwards to echo
	req := []byte{0x05, 0x01, 0x00, 0x01}
	req = append(req, 127, 0, 0, 1, 0x1F, 0x90)
	conn.Write(req)
	rep := make([]byte, 10)
	if _, err := io.ReadFull(conn, rep); err != nil {
		t.Fatal(err)
	}
	if rep[1] != repSuccess {
		t.Fatalf("expected success, got %d", rep[1])
	}

	conn.Write([]byte("proxy-ping"))
	got := make([]byte, len("proxy-ping"))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != "proxy-ping" {
		t.Fatalf("expected echo, got %q", got)
	}
}

func TestPortForward(t *testing.T) {
	echoAddr := startEcho(t)
	table := NewTable()
	table.Add("203.0.113.0/24", "sess-2", directDialer{target: echoAddr}, 1)

	f := NewForward(table)
	_ = context.Background()
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	local := ln.Addr().String()
	ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := f.Start(ctx, local, "203.0.113.5:8080"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)

	c, err := net.Dial("tcp", local)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Write([]byte("fw-test"))
	got := make([]byte, len("fw-test"))
	if _, err := io.ReadFull(c, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != "fw-test" {
		t.Fatalf("expected fw-test, got %q", got)
	}
}

func TestRelayBidirectional(t *testing.T) {
	clientEnd, serverEnd := net.Pipe()
	upstreamClient, upstreamServer := net.Pipe()

	done := make(chan struct{})
	go func() {
		relay(serverEnd, upstreamClient)
		close(done)
	}()

	go func() {
		buf := make([]byte, 4)
		io.ReadFull(upstreamServer, buf)
		upstreamServer.Write(buf)
		upstreamServer.Close()
	}()

	clientEnd.Write([]byte("abcd"))
	got := make([]byte, 4)
	if _, err := io.ReadFull(clientEnd, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != "abcd" {
		t.Fatalf("expected abcd, got %q", got)
	}
	clientEnd.Close()
	<-done
}

var _ = fmt.Sprintf
