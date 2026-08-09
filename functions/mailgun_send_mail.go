package functions

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/mailgun/mailgun-go/v4"
)

func SendSimpleMessage(toEmail string, subject string, body string) (string, error) {
	apiKey := os.Getenv("MAILGUN_API_KEY")
	domain := os.Getenv("MAILGUN_DOMAIN")

	if apiKey == "" || domain == "" {
		return "", errors.New("mailgun credentials not set")
	}

	mg := mailgun.NewMailgun(domain, apiKey)

	message := mg.NewMessage(
		"Mailgun Sandbox <postmaster@"+domain+">",
		subject,
		body,
		toEmail,
	)

	log.Println("Sending email via Mailgun...")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_, id, err := mg.Send(ctx, message)
	return id, err
}
