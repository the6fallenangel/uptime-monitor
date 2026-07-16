package notifier

import (
	"context"
	"fmt"

	"gopkg.in/gomail.v2"
)

type EmailNotifier struct {
	dialer *gomail.Dialer
	from   string
}

func NewEmailNotifier(host string, port int, user, pass, from string) *EmailNotifier {
	return &EmailNotifier{
		dialer: gomail.NewDialer(host, port, user, pass),
		from:   from,
	}
}

func (n *EmailNotifier) Notify(ctx context.Context, event Event) error {
	if event.OwnerEmail == "" {
		return fmt.Errorf("notify: event has no owner email set")
	}

	subject := fmt.Sprintf("[%s] %s is now %s", event.Monitor.Name, event.Monitor.URL, event.NewStatus)

	body := fmt.Sprintf(
		"Monitor: %s\nURL: %s\nStatus changed: %s -> %s\nChecked at: %s\n",
		event.Monitor.Name, event.Monitor.URL,
		event.PreviousStatus, event.NewStatus,
		event.Check.CheckedAt.Format("2006-01-02 15:04:05"),
	)
	if event.Check.Error != "" {
		body += fmt.Sprintf("Error: %s\n", event.Check.Error)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", n.from)
	m.SetHeader("To", event.OwnerEmail)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	if err := n.dialer.DialAndSend(m); err != nil {
		return fmt.Errorf("sending email: %w", err)
	}
	return nil
}
