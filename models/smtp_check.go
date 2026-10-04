package models

import (
	"errors"
	"net"
	"net/mail"
	"strings"
)

const (
	SMTPCheckPass    = "pass"
	SMTPCheckWarn    = "warn"
	SMTPCheckFail    = "fail"
	SMTPCheckConnect = "connect"
	SMTPCheckSPF     = "spf"
	SMTPCheckDMARC   = "dmarc"
	SMTPCheckMissing = "missing"
)

// SMTPCheck is one advisory result from a sending-profile check.
// It does not send mail and does not block a campaign launch.
type SMTPCheck struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// CheckSMTP dials the sending profile and looks up SPF and DMARC for the From domain.
// A missing DNS policy is a warning: the SMTP session can still succeed.
func CheckSMTP(s *SMTP) ([]SMTPCheck, error) {
	if s == nil {
		return nil, ErrHostNotSpecified
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	checks := []SMTPCheck{checkSMTPConnect(s)}
	domain := fromDomain(s.FromAddress)
	spfRecords, spfErr := net.LookupTXT(domain)
	dmarcRecords, dmarcErr := net.LookupTXT("_dmarc." + domain)
	spf, dmarc := DomainPolicyChecks(spfRecords, spfErr, dmarcRecords, dmarcErr)
	return append(checks, spf, dmarc), nil
}

func checkSMTPConnect(s *SMTP) SMTPCheck {
	dialer, err := s.GetDialer()
	if err != nil {
		return failedConnect(err)
	}
	sender, err := dialer.Dial()
	if err != nil {
		return failedConnect(err)
	}
	if sender != nil {
		_ = sender.Close()
	}
	return SMTPCheck{ID: SMTPCheckConnect, Status: SMTPCheckPass}
}

func failedConnect(err error) SMTPCheck {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	return SMTPCheck{
		ID:     SMTPCheckConnect,
		Status: SMTPCheckFail,
		Reason: ClassifySendError(msg),
		Detail: truncateDetail(msg),
	}
}

func fromDomain(from string) string {
	addr, err := mail.ParseAddress(from)
	if err != nil {
		return ""
	}
	parts := strings.Split(addr.Address, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(parts[1]))
}

// DomainPolicyChecks classifies SPF and DMARC TXT lookups.
// A not-found lookup is a warning, not a connection failure.
func DomainPolicyChecks(spfRecords []string, spfErr error, dmarcRecords []string, dmarcErr error) (SMTPCheck, SMTPCheck) {
	return policyCheck(SMTPCheckSPF, "v=spf1", spfRecords, spfErr),
		policyCheck(SMTPCheckDMARC, "v=dmarc1", dmarcRecords, dmarcErr)
}

func policyCheck(id, prefix string, records []string, err error) SMTPCheck {
	if err != nil && !isDNSNotFound(err) {
		return SMTPCheck{
			ID:     id,
			Status: SMTPCheckFail,
			Reason: SendReasonDNS,
			Detail: truncateDetail(err.Error()),
		}
	}
	if rec, ok := matchingTXT(records, prefix); ok {
		return SMTPCheck{
			ID:     id,
			Status: SMTPCheckPass,
			Detail: truncateDetail(rec),
		}
	}
	return SMTPCheck{
		ID:     id,
		Status: SMTPCheckWarn,
		Reason: SMTPCheckMissing,
	}
}

func matchingTXT(records []string, prefix string) (string, bool) {
	prefix = strings.ToLower(prefix)
	for _, rec := range records {
		trimmed := strings.TrimSpace(rec)
		if strings.HasPrefix(strings.ToLower(trimmed), prefix) {
			return trimmed, true
		}
	}
	return "", false
}

func isDNSNotFound(err error) bool {
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return dnsErr.IsNotFound
	}
	return false
}

func truncateDetail(s string) string {
	s = strings.TrimSpace(s)
	const max = 180
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
