package mail

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
)

// SMTPSettings holds the resolved SMTP configuration for sending email.
type SMTPSettings struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLS      string // "none", "starttls", "tls"
}

// PeerConfigEmail holds the data for a peer config email.
type PeerConfigEmail struct {
	To         string
	PeerName   string
	Note       string // optional custom message
	ConfData   []byte // the .conf file content
	QRCodePNG  []byte // QR code image
	ServerName string // for branding (e.g. "vpn.example.com")
}

// AlertEmail holds the data for a peer alert email.
type AlertEmail struct {
	To            string
	PeerName      string
	PublicKey     string
	InterfaceName string
	Event         string // "connected" or "disconnected"
	Endpoint      string
	TransferRx    int64
	TransferTx    int64
	Duration      string // formatted duration for disconnect events
	Timestamp     string
	ServerName    string // for branding
}

// SendAlertEmail sends a peer connection alert email.
func SendAlertEmail(s SMTPSettings, a AlertEmail) error {
	event := "Connected"
	if a.Event != "connected" {
		event = "Disconnected"
	}
	subject := fmt.Sprintf("WireGuard Alert: %s %s", a.PeerName, event)

	htmlContent, err := renderAlertEmail(a)
	if err != nil {
		return fmt.Errorf("rendering alert email: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("From: " + s.From + "\r\n")
	buf.WriteString("To: " + a.To + "\r\n")
	buf.WriteString("Subject: " + subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(htmlContent)

	return sendMail(s, a.To, buf.Bytes())
}

// SendPeerConfig sends a peer configuration email with .conf attachment and inline QR code.
func SendPeerConfig(s SMTPSettings, p PeerConfigEmail) error {
	subject := "WireGuard Configuration"
	if p.PeerName != "" {
		subject += ": " + p.PeerName
	}

	body, err := buildPeerConfigMessage(s.From, p, subject)
	if err != nil {
		return fmt.Errorf("building email: %w", err)
	}

	return sendMail(s, p.To, body)
}

// SendTestEmail sends a styled test email to verify SMTP configuration.
func SendTestEmail(s SMTPSettings, to, serverURL string) error {
	htmlContent, err := renderTestEmail(serverURL, to)
	if err != nil {
		return fmt.Errorf("rendering test email: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("From: " + s.From + "\r\n")
	buf.WriteString("To: " + to + "\r\n")
	buf.WriteString("Subject: wgRift SMTP Test\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(htmlContent)

	return sendMail(s, to, buf.Bytes())
}

func buildPeerConfigMessage(from string, p PeerConfigEmail, subject string) ([]byte, error) {
	var buf bytes.Buffer

	// Top-level headers
	buf.WriteString("From: " + from + "\r\n")
	buf.WriteString("To: " + p.To + "\r\n")
	buf.WriteString("Subject: " + subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")

	// multipart/mixed (body + .conf attachment)
	mixedWriter := multipart.NewWriter(&buf)
	buf.WriteString("Content-Type: multipart/mixed; boundary=\"" + mixedWriter.Boundary() + "\"\r\n")
	buf.WriteString("\r\n")

	// Part 1: multipart/related (HTML with inline QR)
	relatedHeader := textproto.MIMEHeader{}
	relatedWriter := multipart.NewWriter(nil) // just for boundary
	relatedHeader.Set("Content-Type", "multipart/related; boundary=\""+relatedWriter.Boundary()+"\"")
	relatedPart, err := mixedWriter.CreatePart(relatedHeader)
	if err != nil {
		return nil, err
	}

	// Re-create writer with the part as destination
	relatedWriter2 := multipart.NewWriter(relatedPart)
	relatedWriter2.SetBoundary(relatedWriter.Boundary())

	// HTML body part
	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", "text/html; charset=utf-8")
	htmlPart, err := relatedWriter2.CreatePart(htmlHeader)
	if err != nil {
		return nil, err
	}

	htmlContent, err := renderPeerConfigEmail(p)
	if err != nil {
		return nil, err
	}
	htmlPart.Write([]byte(htmlContent))

	// Inline QR code image
	if len(p.QRCodePNG) > 0 {
		qrHeader := textproto.MIMEHeader{}
		qrHeader.Set("Content-Type", "image/png")
		qrHeader.Set("Content-Transfer-Encoding", "base64")
		qrHeader.Set("Content-ID", "<qrcode>")
		qrHeader.Set("Content-Disposition", "inline; filename=\"qrcode.png\"")
		qrPart, err := relatedWriter2.CreatePart(qrHeader)
		if err != nil {
			return nil, err
		}
		encoded := base64.StdEncoding.EncodeToString(p.QRCodePNG)
		// Wrap at 76 chars per line
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			qrPart.Write([]byte(encoded[i:end] + "\r\n"))
		}
	}

	relatedWriter2.Close()

	// Part 2: .conf file attachment
	filename := "wgrift.conf"
	if p.PeerName != "" {
		// Sanitize peer name for filename
		safe := strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
				return r
			}
			return '_'
		}, p.PeerName)
		filename = safe + ".conf"
	}

	confHeader := textproto.MIMEHeader{}
	confHeader.Set("Content-Type", "application/octet-stream")
	confHeader.Set("Content-Transfer-Encoding", "base64")
	confHeader.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	confPart, err := mixedWriter.CreatePart(confHeader)
	if err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(p.ConfData)
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		confPart.Write([]byte(encoded[i:end] + "\r\n"))
	}

	mixedWriter.Close()

	return buf.Bytes(), nil
}

func sendMail(s SMTPSettings, to string, msg []byte) error {
	addr := net.JoinHostPort(s.Host, fmt.Sprintf("%d", s.Port))

	switch strings.ToLower(s.TLS) {
	case "tls":
		return sendMailTLS(s, addr, to, msg)
	case "none":
		return sendMailPlain(s, addr, to, msg)
	default: // "starttls" or empty
		return sendMailSTARTTLS(s, addr, to, msg)
	}
}

func sendMailSTARTTLS(s SMTPSettings, addr, to string, msg []byte) error {
	var auth smtp.Auth
	if s.Username != "" {
		auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
	}
	return smtp.SendMail(addr, auth, s.From, []string{to}, msg)
}

func sendMailTLS(s SMTPSettings, addr, to string, msg []byte) error {
	tlsConfig := &tls.Config{ServerName: s.Host}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial: %w", err)
	}

	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("SMTP client: %w", err)
	}
	defer client.Close()

	if s.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", s.Username, s.Password, s.Host)); err != nil {
			return fmt.Errorf("SMTP auth: %w", err)
		}
	}

	if err := client.Mail(s.From); err != nil {
		return fmt.Errorf("SMTP MAIL: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("writing message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("closing message: %w", err)
	}

	return client.Quit()
}

func sendMailPlain(s SMTPSettings, addr, to string, msg []byte) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("SMTP client: %w", err)
	}
	defer client.Close()

	if s.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", s.Username, s.Password, s.Host)); err != nil {
			return fmt.Errorf("SMTP auth: %w", err)
		}
	}

	if err := client.Mail(s.From); err != nil {
		return fmt.Errorf("SMTP MAIL: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("writing message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("closing message: %w", err)
	}

	return client.Quit()
}
