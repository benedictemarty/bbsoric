package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestTLSServe(t *testing.T) {
	cert, err := SelfSignedCert("test")
	if err != nil {
		t.Fatalf("cert: %v", err)
	}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := New(Config{IdleTimeout: 2 * time.Second}, echoHandler{}, discardLogger())

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); srv.Serve(ctx, ln) }()
	defer func() { cancel(); _ = ln.Close(); wg.Wait() }()

	conn, err := tls.Dial("tcp", ln.Addr().String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("dial TLS: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("hi\r\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	r := bufio.NewReader(conn)
	line, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got := strings.TrimSpace(line); got != "echo:hi" {
		t.Fatalf("réponse TLS inattendue: %q", got)
	}
}
