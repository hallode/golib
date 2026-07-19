// Package email is a fluent SMTP client over gomail. New(Config) returns a
// reusable Client (create one per service); build messages with the chained
// Message API — To/Subject/Text/HTML/AttachFile — and send via Send or
// SendContext. Port 465 enables implicit TLS automatically.
package email

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"strings"
	"time"

	"gopkg.in/gomail.v2"
)

const (
	defaultPort    = 587
	defaultTimeout = 30 * time.Second

	headerFrom    = "From"
	headerTo      = "To"
	headerCc      = "Cc"
	headerBcc     = "Bcc"
	headerReplyTo = "Reply-To"
	headerSubject = "Subject"
)

// Config holds SMTP connection and default sender settings.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string

	// From is the sender address. Defaults to Username when empty.
	From     string
	FromName string

	// UseSSL enables implicit TLS (typical for port 465). When Port is 465 and
	// UseSSL is unset, SSL is enabled automatically.
	UseSSL *bool

	TLSConfig *tls.Config
	LocalName string
}

// Client sends email through SMTP. Create one per service and reuse it.
type Client struct {
	dialer   *gomail.Dialer
	from     string
	fromName string
	sendFn   func(...*gomail.Message) error
}

// New validates config and returns a reusable SMTP client.
func New(cfg Config) (*Client, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	port := cfg.Port
	if port == 0 {
		port = defaultPort
	}

	dialer := gomail.NewDialer(cfg.Host, port, cfg.Username, cfg.Password)
	if cfg.TLSConfig != nil {
		dialer.TLSConfig = cfg.TLSConfig
	}
	if cfg.LocalName != "" {
		dialer.LocalName = cfg.LocalName
	}
	if cfg.UseSSL != nil {
		dialer.SSL = *cfg.UseSSL
	} else if port == 465 {
		dialer.SSL = true
	}

	from := cfg.From
	if from == "" {
		from = cfg.Username
	}

	return &Client{
		dialer:   dialer,
		from:     from,
		fromName: cfg.FromName,
		sendFn:   dialer.DialAndSend,
	}, nil
}

func (cfg Config) validate() error {
	if strings.TrimSpace(cfg.Host) == "" {
		return errors.New("email: host is required")
	}
	if strings.TrimSpace(cfg.Username) == "" {
		return errors.New("email: username is required")
	}
	if cfg.Port < 0 || cfg.Port > 65535 {
		return errors.New("email: port must be between 0 and 65535")
	}

	from := cfg.From
	if from == "" {
		from = cfg.Username
	}
	if _, err := mail.ParseAddress(from); err != nil {
		return fmt.Errorf("email: invalid from address %q: %w", from, err)
	}

	return nil
}

// Message builds a single email. Create via Client.NewMessage.
type Message struct {
	msg      *gomail.Message
	plain    string
	html     string
	hasPlain bool
	hasHTML  bool
}

// NewMessage creates a message with the client's default From header.
func (c *Client) NewMessage() *Message {
	msg := gomail.NewMessage()
	c.applyDefaultFrom(msg)
	return &Message{msg: msg}
}

func (c *Client) applyDefaultFrom(msg *gomail.Message) {
	if c.fromName != "" {
		msg.SetAddressHeader(headerFrom, c.from, c.fromName)
		return
	}
	msg.SetHeader(headerFrom, c.from)
}

// From overrides the sender for this message only.
func (m *Message) From(address, name string) *Message {
	if name != "" {
		m.msg.SetAddressHeader(headerFrom, address, name)
	} else {
		m.msg.SetHeader(headerFrom, address)
	}
	return m
}

func (m *Message) To(addrs ...string) *Message {
	if len(addrs) > 0 {
		m.msg.SetHeader(headerTo, addrs...)
	}
	return m
}

func (m *Message) Cc(addrs ...string) *Message {
	if len(addrs) > 0 {
		m.msg.SetHeader(headerCc, addrs...)
	}
	return m
}

func (m *Message) Bcc(addrs ...string) *Message {
	if len(addrs) > 0 {
		m.msg.SetHeader(headerBcc, addrs...)
	}
	return m
}

func (m *Message) ReplyTo(addrs ...string) *Message {
	if len(addrs) > 0 {
		m.msg.SetHeader(headerReplyTo, addrs...)
	}
	return m
}

func (m *Message) Subject(subject string) *Message {
	m.msg.SetHeader(headerSubject, subject)
	return m
}

func (m *Message) Text(body string) *Message {
	m.plain = body
	m.hasPlain = true
	return m
}

func (m *Message) HTML(body string) *Message {
	m.html = body
	m.hasHTML = true
	return m
}

// AttachFile attaches a file from disk. Optional name overrides the attachment filename.
func (m *Message) AttachFile(path string, name ...string) *Message {
	settings := make([]gomail.FileSetting, 0, 1)
	if len(name) > 0 && name[0] != "" {
		settings = append(settings, gomail.Rename(name[0]))
	}
	m.msg.Attach(path, settings...)
	return m
}

// AttachBytes attaches in-memory content. contentType is optional (e.g. application/pdf).
func (m *Message) AttachBytes(filename string, data []byte, contentType string) *Message {
	settings := []gomail.FileSetting{
		gomail.Rename(filename),
		gomail.SetCopyFunc(func(w io.Writer) error {
			_, err := w.Write(data)
			return err
		}),
	}
	if contentType != "" {
		settings = append(settings, gomail.SetHeader(map[string][]string{
			"Content-Type": {contentType},
		}))
	}
	m.msg.Attach("bytes", settings...)
	return m
}

// EmbedFile embeds an image referenced in HTML via cid, e.g. <img src="cid:logo.png">.
func (m *Message) EmbedFile(path string, cid ...string) *Message {
	settings := make([]gomail.FileSetting, 0, 1)
	if len(cid) > 0 && cid[0] != "" {
		settings = append(settings, gomail.Rename(cid[0]))
	}
	m.msg.Embed(path, settings...)
	return m
}

// SetHeader sets a custom MIME header when the built-in helpers are not enough.
func (m *Message) SetHeader(key string, values ...string) *Message {
	m.msg.SetHeader(key, values...)
	return m
}

// Send delivers the message over SMTP.
func (c *Client) Send(msg *Message) error {
	return c.SendContext(context.Background(), msg)
}

// SendContext sends the message and returns when finished or when ctx is cancelled.
func (c *Client) SendContext(ctx context.Context, msg *Message) error {
	if msg == nil {
		return errors.New("email: message is nil")
	}
	if err := msg.finalize(); err != nil {
		return err
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}

	type result struct {
		err error
	}
	ch := make(chan result, 1)
	go func() {
		ch <- result{c.sendFn(msg.msg)}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case r := <-ch:
		return r.err
	}
}

// SendMany sends multiple messages in one SMTP connection.
func (c *Client) SendMany(ctx context.Context, msgs ...*Message) error {
	if len(msgs) == 0 {
		return errors.New("email: no messages to send")
	}

	gMsgs := make([]*gomail.Message, 0, len(msgs))
	for i, msg := range msgs {
		if msg == nil {
			return fmt.Errorf("email: message %d is nil", i+1)
		}
		if err := msg.finalize(); err != nil {
			return fmt.Errorf("email: message %d: %w", i+1, err)
		}
		gMsgs = append(gMsgs, msg.msg)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}

	type result struct {
		err error
	}
	ch := make(chan result, 1)
	go func() {
		ch <- result{c.sendFn(gMsgs...)}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case r := <-ch:
		return r.err
	}
}

func (m *Message) finalize() error {
	if len(m.msg.GetHeader(headerTo)) == 0 &&
		len(m.msg.GetHeader(headerCc)) == 0 &&
		len(m.msg.GetHeader(headerBcc)) == 0 {
		return errors.New("email: at least one recipient is required")
	}

	switch {
	case m.hasPlain && m.hasHTML:
		m.msg.SetBody("text/plain; charset=UTF-8", m.plain)
		m.msg.AddAlternative("text/html; charset=UTF-8", m.html)
	case m.hasHTML:
		m.msg.SetBody("text/html; charset=UTF-8", m.html)
	case m.hasPlain:
		m.msg.SetBody("text/plain; charset=UTF-8", m.plain)
	default:
		return errors.New("email: message body is required")
	}

	return nil
}
