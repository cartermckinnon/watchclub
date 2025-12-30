package mail

import (
	"fmt"

	"go.uber.org/zap"
)

// Config holds email configuration
type Config struct {
	// DevelopmentMode logs emails to console instead of sending
	DevelopmentMode bool

	// BaseURL is the base URL for generating login links
	BaseURL string

	// Resend configuration
	ResendAPIKey   string
	ResendFrom     string // Email address to send from (e.g., "you@yourdomain.com")
	ResendFromName string // Optional display name (e.g., "WatchClub")

	Logger *zap.Logger
}

// New creates a new email sender
func New(config Config) Sender {
	// Use development mode if explicitly enabled
	if config.DevelopmentMode {
		return &devSender{
			baseURL: config.BaseURL,
			logger:  config.Logger,
		}
	}

	// Use Resend if API key is provided
	if config.ResendAPIKey != "" {
		sender, err := newResendSender(
			config.ResendAPIKey,
			config.ResendFrom,
			config.ResendFromName,
			config.BaseURL,
			config.Logger,
		)
		if err != nil {
			config.Logger.Error("Failed to create Resend sender, falling back to dev mode",
				zap.Error(err),
			)
			return &devSender{
				baseURL: config.BaseURL,
				logger:  config.Logger,
			}
		}
		return sender
	}

	// Default to development mode if no email provider is configured
	config.Logger.Warn("No email provider configured, using development mode (console logging)")
	return &devSender{
		baseURL: config.BaseURL,
		logger:  config.Logger,
	}
}

// devSender logs emails to console instead of sending them
type devSender struct {
	baseURL string
	logger  *zap.Logger
}

func (d *devSender) SendLogin(to, userName, userID, baseURL string) error {
	if baseURL == "" {
		baseURL = d.baseURL
	}

	recoveryLink := fmt.Sprintf("%s#/login/%s", baseURL, userID)

	emailBody := fmt.Sprintf(`
========================================
WATCHCLUB ACCOUNT RECOVERY
========================================

Hi %s,

Click the link below to log back into your account:

%s

This link will automatically log you in.

========================================
`, userName, recoveryLink)

	d.logger.Info("📧 RECOVERY EMAIL (Development Mode)",
		zap.String("to", to),
		zap.String("userName", userName),
		zap.String("recoveryLink", recoveryLink),
	)

	fmt.Println(emailBody)

	return nil
}
