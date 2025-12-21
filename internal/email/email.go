package email

import (
	"fmt"

	"go.uber.org/zap"
)

// Sender is an interface for sending emails
type Sender interface {
	SendRecoveryEmail(to, userName, userID, baseURL string) error
}

// Config holds email configuration
type Config struct {
	// For now, just use development mode
	// In production, this would include SMTP settings
	DevelopmentMode bool
	BaseURL         string
	Logger          *zap.Logger
}

// New creates a new email sender
func New(config Config) Sender {
	if config.DevelopmentMode {
		return &devSender{
			baseURL: config.BaseURL,
			logger:  config.Logger,
		}
	}
	// TODO: Implement SMTP sender for production
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

func (d *devSender) SendRecoveryEmail(to, userName, userID, baseURL string) error {
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
