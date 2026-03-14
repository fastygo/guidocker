package domain

import "time"

// PlatformSettings defines globally managed admin panel exposure settings.
type PlatformSettings struct {
	AdminHost     string `json:"admin_host"`
	AdminPort     int    `json:"admin_port"`
	AdminDomain   string `json:"admin_domain"`
	AdminUseTLS   bool   `json:"admin_use_tls"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// DefaultPlatformSettings returns safe starting settings for a fresh installation.
func DefaultPlatformSettings() PlatformSettings {
	return PlatformSettings{
		AdminHost: "0.0.0.0",
		AdminPort: 3000,
	}
}
