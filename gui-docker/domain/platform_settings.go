package domain

import "time"

// PlatformSettings defines globally managed admin panel exposure settings.
type PlatformSettings struct {
	AdminHost            string    `json:"admin_host"`
	AdminPort            int       `json:"admin_port"`
	AdminDomain          string    `json:"admin_domain"`
	AdminUseTLS          bool      `json:"admin_use_tls"`
	CertbotEmail         string    `json:"certbot_email"`
	CertbotEnabled       bool      `json:"certbot_enabled"`
	CertbotStaging       bool      `json:"certbot_staging"`
	CertbotAutoRenew     bool      `json:"certbot_auto_renew"`
	CertbotTermsAccepted bool      `json:"certbot_terms_accepted"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// DefaultPlatformSettings returns safe starting settings for a fresh installation.
func DefaultPlatformSettings() PlatformSettings {
	return PlatformSettings{
		AdminHost: "0.0.0.0",
		AdminPort: 3000,
	}
}
