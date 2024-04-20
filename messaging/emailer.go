package messaging

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"canvas/model"
)

const (
	marketingMessageStream     = "broadcast"
	transactionalMessageStream = "outbound"
)

// nameAndEmail combo, of the form "Name <email@example.com>"
type nameAndEmail = string

//go:embed emails
var emails embed.FS

// Emailer can send transactional and marketing emails through Postmark.
// See https://postmarkapp.com/developer
type Emailer struct {
	baseURL           *url.URL
	client            *http.Client
	marketingFrom     nameAndEmail
	token             string
	transactionalFrom nameAndEmail
}

type NewEmailerOptions struct {
	BaseURL                   *url.URL
	MarketingEmailAddress     string
	MarketingEmailName        string
	Token                     string
	TransactionalEmailAddress string
	TransactionalEmailName    string
}

func NewEmailer(opts NewEmailerOptions) *Emailer {
	return &Emailer{
		baseURL:       opts.BaseURL,
		client:        &http.Client{Timeout: 3 * time.Second},
		marketingFrom: createNameAndEmail(opts.MarketingEmailName, opts.MarketingEmailAddress),
		token:         opts.Token,
		transactionalFrom: createNameAndEmail(
			opts.TransactionalEmailName,
			opts.TransactionalEmailAddress,
		),
	}
}

// SendNewsletterConfirmationEmail with a confirmation link.
// This is a transactional email, because it's a response to a user action.
func (e *Emailer) SendNewsletterConfirmationEmail(
	ctx context.Context,
	to model.Email,
	token string,
) error {
	actionUrl := e.baseURL
	actionUrl = actionUrl.JoinPath("/newsletter/confirm")
	actionUrl.RawQuery = fmt.Sprintf("token=%s", token)
	keywords := map[string]string{
		"base_url":   e.baseURL.String(),
		"action_url": actionUrl.String(),
	}

	return e.send(ctx, requestBody{
		MessageStream: transactionalMessageStream,
		From:          e.transactionalFrom,
		To:            to.String(),
		Subject:       "Confirm your subscription to the Canvas newsletter",
		HtmlBody:      getEmail("confirmation_email.html", keywords),
		TextBody:      getEmail("confirmation_email.txt", keywords),
	})
}

// SendNewsletterWelcomeEmail with just the web app URL.
func (e *Emailer) SendNewsletterWelcomeEmail(ctx context.Context, to model.Email) error {
	keywords := map[string]string{
		"base_url": e.baseURL.String(),
	}

	return e.send(ctx, requestBody{
		MessageStream: marketingMessageStream,
		From:          e.marketingFrom,
		To:            to.String(),
		Subject:       "Welcome to the Canvas newsletter",
		HtmlBody:      getEmail("welcome_email.html", keywords),
		TextBody:      getEmail("welcome_email.txt", keywords),
	})
}

// requestBody used in Emailer.send.
// See https://postmarkapp.com/developer/user-guide/send-email-with-api
type requestBody struct {
	MessageStream string
	From          nameAndEmail
	To            nameAndEmail
	Subject       string
	HtmlBody      string
	TextBody      string
}

// send using the Postmark API.
func (e *Emailer) send(ctx context.Context, body requestBody) error {
	bodyAsBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("error marshalling request body to json: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.postmarkapp.com/email",
		bytes.NewReader(bodyAsBytes),
	)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Postmark-Server-Token", e.token)

	response, err := e.client.Do(request)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()
	bodyAsBytes, err = io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}
	if response.StatusCode > 299 {
		slog.Info("Error sending email",
			slog.Int("status", response.StatusCode), slog.String("response", string(bodyAsBytes)))
		return fmt.Errorf("error sending email, got status %v", response.StatusCode)
	}

	return nil
}

// createNameAndEmail returns a name and email string ready for inserting into From and To fields.
func createNameAndEmail(name, email string) nameAndEmail {
	return fmt.Sprintf("%v <%v>", name, email)
}

// getEmail from the given path, panicking on errors.
// It also replaces keywords given in the map.
func getEmail(path string, keywords map[string]string) string {
	email, err := emails.ReadFile("emails/" + path)
	if err != nil {
		panic(err)
	}

	emailString := string(email)
	for keyword, replacement := range keywords {
		emailString = strings.ReplaceAll(emailString, "{{"+keyword+"}}", replacement)
	}

	return emailString
}
