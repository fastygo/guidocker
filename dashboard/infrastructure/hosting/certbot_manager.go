package hosting

import (
	"context"
	"dashboard/domain"
	"fmt"
	"os"
	"strings"
)

const defaultCertbotBinary = "certbot"

type CertbotManager struct {
	binary    string
	runScript func(context.Context, string, ...string) ([]byte, error)
}

func NewCertbotManager() *CertbotManager {
	binary := strings.TrimSpace(os.Getenv("PAAS_CERTBOT_BINARY"))
	return NewCertbotManagerWithBinary(binary)
}

func NewCertbotManagerWithBinary(binary string) *CertbotManager {
	trimmed := strings.TrimSpace(binary)
	if trimmed == "" {
		trimmed = defaultCertbotBinary
	}
	return &CertbotManager{
		binary:    trimmed,
		runScript: runCommand,
	}
}

func (m *CertbotManager) EnsureCertificate(ctx context.Context, settings domain.PlatformSettings, domainName string) error {
	domainValue := strings.TrimSpace(domainName)
	if domainValue == "" {
		return domain.ErrInvalidDomain
	}
	if !settings.CertbotEnabled {
		return fmt.Errorf("certbot is disabled for platform")
	}
	email := strings.TrimSpace(settings.CertbotEmail)
	if email == "" {
		return fmt.Errorf("certbot email is required to issue certificates")
	}
	if !settings.CertbotTermsAccepted {
		return fmt.Errorf("certbot terms of service must be accepted")
	}

	args := []string{"certonly", "--nginx", "--non-interactive", "--agree-tos", "--email", email, "-d", domainValue}
	if settings.CertbotStaging {
		args = append(args, "--test-cert")
	}
	if settings.CertbotAutoRenew {
		// Best effort: keep renewal configuration when available in certbot internals.
	}

	_, err := m.runScript(ctx, m.binary, args...)
	if err != nil {
		return fmt.Errorf("ensure certificate: %w", err)
	}
	return nil
}

func (m *CertbotManager) RemoveCertificate(ctx context.Context, domainName string) error {
	domainValue := strings.TrimSpace(domainName)
	if domainValue == "" {
		return nil
	}

	args := []string{"delete", "--non-interactive", "--cert-name", domainValue}
	output, err := m.runScript(ctx, m.binary, args...)
	if err != nil {
		lowerOutput := strings.ToLower(strings.TrimSpace(string(output)))
		if strings.Contains(lowerOutput, "no such entry") || strings.Contains(lowerOutput, "no cert") {
			return nil
		}
		return fmt.Errorf("remove certificate: %w", err)
	}
	return nil
}

func (m *CertbotManager) RenewCertificates(ctx context.Context) error {
	_, err := m.runScript(ctx, m.binary, "renew")
	if err != nil {
		return fmt.Errorf("renew certificates: %w", err)
	}
	return nil
}

