package utilities

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"bufio"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func init() {
	if logger.Log == nil {
		logger.Log = logrus.New()
	}
}

// fakeSMTPServer speaks just enough SMTP to satisfy github.com/go-mail/mail's
// Dialer: greeting, EHLO, MAIL FROM, RCPT TO, DATA, QUIT. No STARTTLS/AUTH is
// advertised, so the dialer sends the message in plain text over the loopback
// connection - exactly what SendSMTP* needs to exercise their success path
// without reaching a real mail server.
type fakeSMTPServer struct {
	addr string
	host string
	port int

	mu       sync.Mutex
	messages []string // accumulated DATA bodies, one per accepted message
}

func startFakeSMTPServer(t *testing.T) *fakeSMTPServer {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start fake SMTP listener: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("failed to parse listener address: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to parse listener port: %v", err)
	}

	srv := &fakeSMTPServer{addr: ln.Addr().String(), host: host, port: port}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener closed
			}
			go srv.handle(conn)
		}
	}()

	return srv
}

func (s *fakeSMTPServer) handle(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	writeLine := func(line string) {
		conn.Write([]byte(line + "\r\n"))
	}

	writeLine("220 fake.smtp ESMTP ready")

	var body strings.Builder
	inData := false

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")

		if inData {
			if line == "." {
				inData = false
				s.mu.Lock()
				s.messages = append(s.messages, body.String())
				s.mu.Unlock()
				body.Reset()
				writeLine("250 OK: message queued")
				continue
			}
			body.WriteString(line + "\n")
			continue
		}

		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
			writeLine("250 fake.smtp Hello")
		case strings.HasPrefix(upper, "MAIL FROM"):
			writeLine("250 OK")
		case strings.HasPrefix(upper, "RCPT TO"):
			writeLine("250 OK")
		case upper == "DATA":
			inData = true
			writeLine("354 Start mail input; end with <CRLF>.<CRLF>")
		case upper == "QUIT":
			writeLine("221 Bye")
			return
		case upper == "RSET":
			writeLine("250 OK")
		default:
			writeLine("500 unrecognized command")
		}
	}
}

func (s *fakeSMTPServer) lastMessage() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.messages) == 0 {
		return ""
	}
	return s.messages[len(s.messages)-1]
}

func (s *fakeSMTPServer) messageCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.messages)
}

// configureTestSMTP points config.ConfigFile at the fake server and restores
// the original SMTP settings afterward.
func configureTestSMTP(t *testing.T, srv *fakeSMTPServer) {
	t.Helper()

	orig := config.ConfigFile
	t.Cleanup(func() { config.ConfigFile = orig })

	config.ConfigFile.SMTPHost = srv.host
	config.ConfigFile.SMTPPort = srv.port
	config.ConfigFile.SMTPUsername = ""
	config.ConfigFile.SMTPPassword = ""
	config.ConfigFile.SMTPFrom = "noreply@example.com"
	config.ConfigFile.PoenskelistenName = "Test App"
	config.ConfigFile.PoenskelistenEnvironment = "production"
	config.ConfigFile.PoenskelistenExternalURL = "https://wishlist.example.com"
}

func testUserForSMTP() models.User {
	email := "recipient@example.com"
	verificationCode := "ABC123"
	resetCode := "RESET123"
	user := models.User{
		FirstName:        "Ada",
		Email:            &email,
		VerificationCode: &verificationCode,
		ResetCode:        &resetCode,
	}
	user.ID = uuid.New()
	return user
}

func TestSendSMTPVerificationEmail(t *testing.T) {
	srv := startFakeSMTPServer(t)
	configureTestSMTP(t, srv)
	user := testUserForSMTP()

	if err := SendSMTPVerificationEmail(user); err != nil {
		t.Fatalf("SendSMTPVerificationEmail returned error: %v", err)
	}
	if srv.messageCount() != 1 {
		t.Fatalf("messageCount = %d, want 1", srv.messageCount())
	}
	if !strings.Contains(srv.lastMessage(), "ABC123") {
		t.Error("expected the verification code in the e-mail body")
	}
}

func TestSendSMTPVerificationEmailUsesTestAddressInTestEnvironment(t *testing.T) {
	srv := startFakeSMTPServer(t)
	configureTestSMTP(t, srv)
	config.ConfigFile.PoenskelistenEnvironment = "test"
	config.ConfigFile.PoenskelistenTestEmail = "sink@example.com"
	user := testUserForSMTP()

	if err := SendSMTPVerificationEmail(user); err != nil {
		t.Fatalf("SendSMTPVerificationEmail returned error: %v", err)
	}
	if srv.messageCount() != 1 {
		t.Fatalf("messageCount = %d, want 1", srv.messageCount())
	}
	if !strings.Contains(srv.lastMessage(), "sink@example.com") {
		t.Error("expected the test e-mail address to be used as the recipient, not the user's real address")
	}
}

func TestSendSMTPResetEmail(t *testing.T) {
	srv := startFakeSMTPServer(t)
	configureTestSMTP(t, srv)
	user := testUserForSMTP()

	if err := SendSMTPResetEmail(user); err != nil {
		t.Fatalf("SendSMTPResetEmail returned error: %v", err)
	}
	if !strings.Contains(srv.lastMessage(), "RESET123") {
		t.Error("expected the reset link/code in the e-mail body")
	}
}

func TestSendSMTPDeletedClaimedWish(t *testing.T) {
	srv := startFakeSMTPServer(t)
	configureTestSMTP(t, srv)
	user := testUserForSMTP()
	wish := models.WishObject{Name: "A nice gift"}
	wishlist := models.WishlistUser{Name: "Birthday list"}
	wishlist.ID = uuid.New()

	if err := SendSMTPDeletedClaimedWish(user, wish, wishlist); err != nil {
		t.Fatalf("SendSMTPDeletedClaimedWish returned error: %v", err)
	}
	if !strings.Contains(srv.lastMessage(), "Birthday list") {
		t.Error("expected the wishlist name in the e-mail body")
	}
}

func TestSendSMTPVerificationEmailDialFailure(t *testing.T) {
	orig := config.ConfigFile
	t.Cleanup(func() { config.ConfigFile = orig })

	// Nothing is listening on this port, so dialing must fail.
	config.ConfigFile.SMTPHost = "127.0.0.1"
	config.ConfigFile.SMTPPort = 1
	config.ConfigFile.SMTPFrom = "noreply@example.com"
	config.ConfigFile.PoenskelistenEnvironment = "production"
	user := testUserForSMTP()

	if err := SendSMTPVerificationEmail(user); err == nil {
		t.Error("expected an error when the SMTP server is unreachable")
	}
}

// TestSMTPSendersDialFailure covers the error return of every sender when the
// SMTP server can't be reached.
func TestSMTPSendersDialFailure(t *testing.T) {
	orig := config.ConfigFile
	t.Cleanup(func() { config.ConfigFile = orig })

	// Bind then close a listener so the port is known to have nothing on it.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	config.ConfigFile.SMTPHost = "127.0.0.1"
	config.ConfigFile.SMTPPort = port
	config.ConfigFile.SMTPFrom = "noreply@example.com"
	config.ConfigFile.PoenskelistenEnvironment = "production"
	user := testUserForSMTP()

	if err := SendSMTPResetEmail(user); err == nil {
		t.Error("SendSMTPResetEmail: expected an error when the SMTP server is unreachable")
	}
	wishlist := models.WishlistUser{Name: "List"}
	if err := SendSMTPDeletedClaimedWish(user, models.WishObject{Name: "Gift"}, wishlist); err == nil {
		t.Error("SendSMTPDeletedClaimedWish: expected an error when the SMTP server is unreachable")
	}
}

// TestSMTPSendersUseTestAddressInTestEnvironment makes sure the reset and
// deleted-claim mails, like the verification mail, never reach a real user
// address in the test environment.
func TestSMTPSendersUseTestAddressInTestEnvironment(t *testing.T) {
	srv := startFakeSMTPServer(t)
	configureTestSMTP(t, srv)
	config.ConfigFile.PoenskelistenEnvironment = "TEST"
	config.ConfigFile.PoenskelistenTestEmail = "sink@example.com"
	user := testUserForSMTP()

	if err := SendSMTPResetEmail(user); err != nil {
		t.Fatalf("SendSMTPResetEmail returned error: %v", err)
	}
	if msg := srv.lastMessage(); !strings.Contains(msg, "sink@example.com") || strings.Contains(msg, "recipient@example.com") {
		t.Errorf("reset mail not redirected to the test address:\n%s", msg)
	}

	wishlist := models.WishlistUser{Name: "List"}
	wishlist.ID = uuid.New()
	if err := SendSMTPDeletedClaimedWish(user, models.WishObject{Name: "Gift"}, wishlist); err != nil {
		t.Fatalf("SendSMTPDeletedClaimedWish returned error: %v", err)
	}
	if msg := srv.lastMessage(); !strings.Contains(msg, "sink@example.com") || strings.Contains(msg, "recipient@example.com") {
		t.Errorf("deleted-claim mail not redirected to the test address:\n%s", msg)
	}
}
