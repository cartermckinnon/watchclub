package mail

import (
	"fmt"

	"github.com/resend/resend-go/v2"
	"go.uber.org/zap"
)

// resendSender sends emails using the Resend API
type resendSender struct {
	client      *resend.Client
	fromAddress string
	fromName    string
	baseURL     string
	logger      *zap.Logger
}

func newResendSender(apiKey, fromAddress, fromName, baseURL string, logger *zap.Logger) (*resendSender, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("resend API key is required")
	}
	if fromAddress == "" {
		return nil, fmt.Errorf("from address is required")
	}

	client := resend.NewClient(apiKey)

	return &resendSender{
		client:      client,
		fromAddress: fromAddress,
		fromName:    fromName,
		baseURL:     baseURL,
		logger:      logger,
	}, nil
}

func (r *resendSender) SendLogin(to, userName, userID, baseURL string) error {
	if baseURL == "" {
		baseURL = r.baseURL
	}

	loginLink := fmt.Sprintf("%s#/login/%s", baseURL, userID)

	// Build from address with optional name
	from := r.fromAddress
	if r.fromName != "" {
		from = fmt.Sprintf("%s <%s>", r.fromName, r.fromAddress)
	}

	// Create HTML email body
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 8px;
            padding: 32px;
            box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        }
        h1 {
            color: #667eea;
            font-size: 24px;
            margin-bottom: 24px;
        }
        .button {
            display: inline-block;
            padding: 12px 24px;
            background: #667eea;
            color: white !important;
            text-decoration: none;
            border-radius: 6px;
            font-weight: 600;
            margin: 24px 0;
        }
        .footer {
            margin-top: 32px;
            padding-top: 24px;
            border-top: 1px solid #e0e0e0;
            color: #666;
            font-size: 14px;
        }
        .link {
            color: #667eea;
            word-break: break-all;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎬 WatchClub Login</h1>
        <p>Hi %s,</p>
        <p>Click the button below to log into your WatchClub account:</p>
        <a href="%s" class="button">Log in to WatchClub</a>
        <p>Or copy and paste this link into your browser:</p>
        <p class="link">%s</p>
        <div class="footer">
            <p>This link will automatically log you in to your account.</p>
            <p>If you didn't request this login link, you can safely ignore this email.</p>
        </div>
    </div>
</body>
</html>
`, userName, loginLink, loginLink)

	// Create plain text version as fallback
	textBody := fmt.Sprintf(`
WatchClub Login

Hi %s,

Click the link below to log into your account:

%s

This link will automatically log you in.

If you didn't request this login link, you can safely ignore this email.
`, userName, loginLink)

	// Send email using Resend API
	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{to},
		Subject: "Log in to WatchClub",
		Html:    htmlBody,
		Text:    textBody,
	}

	sent, err := r.client.Emails.Send(params)
	if err != nil {
		r.logger.Error("Failed to send email via Resend",
			zap.String("to", to),
			zap.Error(err),
		)
		return fmt.Errorf("failed to send email: %w", err)
	}

	r.logger.Info("📧 Login email sent via Resend",
		zap.String("to", to),
		zap.String("userName", userName),
		zap.String("emailId", sent.Id),
	)

	return nil
}
