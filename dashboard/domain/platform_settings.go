package domain

import "time"

// PlatformSettings defines global TLS and certificate automation settings for managed app domains.
type PlatformSettings struct {
	CertbotEmail         string    `json:"certbot_email"`
	CertbotEnabled       bool      `json:"certbot_enabled"`
	CertbotStaging       bool      `json:"certbot_staging"`
	CertbotAutoRenew     bool      `json:"certbot_auto_renew"`
	CertbotTermsAccepted bool `json:"certbot_terms_accepted"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// DefaultPlatformSettings returns safe starting settings for a fresh installation.
func DefaultPlatformSettings() PlatformSettings {
	return PlatformSettings{}
}
