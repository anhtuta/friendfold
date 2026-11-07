package mail

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSMTPSendRecordsPlainText(t *testing.T) {
	t.Parallel()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	got := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			got <- fmt.Sprintf("accept: %v", err)
			return
		}
		defer conn.Close()
		got <- fakeSMTPSession(conn)
	}()

	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("addr: %v", err)
	}
	port, _ := strconv.Atoi(portStr)
	s := &SMTP{Host: host, Port: port, From: "Friendfold <friendfold@localhost>"}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := s.Send(ctx, Message{
		UserID:  42,
		To:      "minh@example.com",
		Subject: "Verify your Friendfold email",
		Body:    "token=super-secret-token-value\n",
	}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	session := <-got
	if !strings.Contains(session, "MAIL FROM:<friendfold@localhost>") {
		t.Fatalf("from missing: %s", session)
	}
	if !strings.Contains(session, "RCPT TO:<minh@example.com>") {
		t.Fatalf("rcpt missing: %s", session)
	}
	if !strings.Contains(session, "token=super-secret-token-value") {
		t.Fatal("body missing from SMTP DATA")
	}
	if strings.Contains(session, "Subject: Verify your Friendfold email\r\n\r\n") {
		t.Fatal("expected headers before blank line")
	}
}

func TestSMTPSendHonorsCancel(t *testing.T) {
	t.Parallel()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)
	s := &SMTP{Host: host, Port: port, From: "friendfold@localhost"}
	if err := s.Send(ctx, Message{To: "minh@example.com", Subject: "x", Body: "y"}); err == nil {
		t.Fatal("expected canceled send")
	}
}

func TestEnvelopeAddr(t *testing.T) {
	t.Parallel()
	if got := envelopeAddr("Friendfold <friendfold@localhost>"); got != "friendfold@localhost" {
		t.Fatalf("got %q", got)
	}
	if got := envelopeAddr("friendfold@localhost"); got != "friendfold@localhost" {
		t.Fatalf("got %q", got)
	}
}

func TestSanitizeHeaderStripsCRLF(t *testing.T) {
	t.Parallel()
	if got := sanitizeHeader("a\r\nBcc: evil@example.com"); strings.Contains(got, "\n") {
		t.Fatalf("got %q", got)
	}
}

func TestSMTPMailpitRoundTrip(t *testing.T) {
	host := envDefault("SMTP_HOST", "127.0.0.1")
	port := 1025
	if raw := strings.TrimSpace(os.Getenv("SMTP_PORT")); raw != "" {
		p, err := strconv.Atoi(raw)
		if err != nil {
			t.Fatalf("SMTP_PORT: %v", err)
		}
		port = p
	}
	d := net.Dialer{Timeout: 400 * time.Millisecond}
	conn, err := d.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		t.Skipf("Mailpit SMTP not reachable: %v", err)
	}
	_ = conn.Close()

	to := fmt.Sprintf("mailpit-%d@example.com", time.Now().UnixNano())
	marker := fmt.Sprintf("marker-%d", time.Now().UnixNano())
	s := &SMTP{Host: host, Port: port, From: "Friendfold <friendfold@localhost>"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Send(ctx, Message{
		To:      to,
		Subject: "Friendfold Mailpit roundtrip",
		Body:    "hello " + marker,
	}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	mailpit := strings.TrimRight(envDefault("MAILPIT_URL", "http://127.0.0.1:8025"), "/")
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if mailpitHasMarker(t, mailpit, to, marker) {
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
	t.Fatal("Mailpit did not show the message")
}

func mailpitHasMarker(t *testing.T, base, to, marker string) bool {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, base+"/api/v1/messages", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("Mailpit HTTP not reachable: %v", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var payload struct {
		Messages []struct {
			To []struct {
				Address string `json:"Address"`
			} `json:"To"`
			ID      string `json:"ID"`
			Snippet string `json:"Snippet"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("json: %v body=%s", err, body)
	}
	for _, m := range payload.Messages {
		match := false
		for _, addr := range m.To {
			if strings.EqualFold(addr.Address, to) {
				match = true
			}
		}
		if !match {
			continue
		}
		if strings.Contains(m.Snippet, marker) {
			return true
		}
		msgRes, err := http.Get(base + "/api/v1/message/" + m.ID)
		if err != nil {
			continue
		}
		raw, _ := io.ReadAll(msgRes.Body)
		_ = msgRes.Body.Close()
		if strings.Contains(string(raw), marker) {
			return true
		}
	}
	return false
}

func envDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func fakeSMTPSession(conn net.Conn) string {
	var b strings.Builder
	br := bufio.NewReader(conn)
	bw := bufio.NewWriter(conn)
	write := func(s string) {
		_, _ = bw.WriteString(s)
		_ = bw.Flush()
	}
	write("220 fake SMTP\r\n")
	inData := false
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return b.String()
		}
		b.WriteString(line)
		upper := strings.ToUpper(strings.TrimSpace(line))
		if inData {
			if line == ".\r\n" {
				inData = false
				write("250 ok\r\n")
			}
			continue
		}
		switch {
		case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
			write("250-localhost\r\n250 HELP\r\n")
		case strings.HasPrefix(upper, "MAIL"), strings.HasPrefix(upper, "RCPT"):
			write("250 ok\r\n")
		case upper == "DATA":
			write("354 go\r\n")
			inData = true
		case upper == "QUIT":
			write("221 bye\r\n")
			return b.String()
		default:
			write("250 ok\r\n")
		}
	}
}
