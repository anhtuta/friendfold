package mail

import "context"

// Message is a plain-text email. Body may contain a one-time token; never log
// Body, To, or Subject.
type Message struct {
	UserID  int64
	To      string
	Subject string
	Body    string
}

// Sender delivers one message. Implementations must not log the body or token.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}
