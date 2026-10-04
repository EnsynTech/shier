package models

import (
	"encoding/json"
	"strconv"
	"strings"
)

const (
	SendReasonAuthFailed      = "auth_failed"
	SendReasonTLSCert         = "tls_cert"
	SendReasonDNS             = "dns"
	SendReasonConnect         = "connect"
	SendReasonRateLimited     = "rate_limited"
	SendReasonMailboxRejected = "mailbox_rejected"
	SendReasonPolicyRejected  = "policy_rejected"
	SendReasonGatewayStatus   = "gateway_status"
	SendReasonInvalidRequest  = "invalid_request"
	SendReasonMaxAttempts     = "max_attempts"
	SendReasonUnknown         = "unknown"
)

// ClassifySendError maps a raw send error onto a stable reason code.
// The code is stored and translated in the UI; the original text is kept.
func ClassifySendError(msg string) string {
	lower := strings.ToLower(strings.TrimSpace(msg))
	if lower == "" {
		return SendReasonUnknown
	}
	switch {
	case strings.Contains(lower, "max send attempts exceeded"):
		return SendReasonMaxAttempts
	case strings.Contains(lower, "not valid json") ||
		strings.Contains(lower, "phone number not specified") ||
		strings.Contains(lower, "invalid body template"):
		return SendReasonInvalidRequest
	case strings.Contains(lower, "gateway returned status"):
		return SendReasonGatewayStatus
	case strings.Contains(lower, "x509") ||
		strings.Contains(lower, "certificate") ||
		strings.Contains(lower, "unknown authority"):
		return SendReasonTLSCert
	case strings.Contains(lower, "no such host") || strings.Contains(lower, "server misbehaving"):
		return SendReasonDNS
	}

	code := leadingSMTPCode(lower)
	if code == 535 || strings.Contains(lower, "authenticat") || strings.Contains(lower, "auth failed") {
		return SendReasonAuthFailed
	}
	if code >= 400 && code <= 499 {
		return SendReasonRateLimited
	}
	if code == 554 ||
		strings.Contains(lower, "relay") ||
		strings.Contains(lower, "spam") ||
		strings.Contains(lower, "blocked") ||
		strings.Contains(lower, "policy") {
		return SendReasonPolicyRejected
	}
	if mailboxRejected(lower) || (code >= 550 && code <= 553) {
		return SendReasonMailboxRejected
	}
	if code >= 500 && code <= 599 {
		return SendReasonPolicyRejected
	}
	if strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "connection reset") ||
		strings.Contains(lower, "connection denied") ||
		strings.Contains(lower, "network is unreachable") ||
		strings.Contains(lower, "i/o timeout") ||
		strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "max connection attempts") {
		return SendReasonConnect
	}
	return SendReasonUnknown
}

func mailboxRejected(lower string) bool {
	phrases := []string{
		"user unknown",
		"no such user",
		"mailbox",
		"recipient",
		"not found",
		"does not exist",
		"invalid recipient",
	}
	for _, phrase := range phrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func leadingSMTPCode(msg string) int {
	for _, field := range strings.Fields(msg) {
		digits := field
		if len(digits) > 3 {
			digits = digits[:3]
		}
		if len(digits) != 3 {
			continue
		}
		n, err := strconv.Atoi(digits)
		if err != nil || n < 400 || n > 599 {
			continue
		}
		return n
	}
	return 0
}

// NewSendEventError builds the event payload stored for a send failure.
func NewSendEventError(err error) EventError {
	if err == nil {
		return EventError{Reason: SendReasonUnknown}
	}
	msg := err.Error()
	return EventError{Error: msg, Reason: ClassifySendError(msg)}
}

// AnnotateSendErrorDetails adds a reason code to a stored event payload
// that only has the raw error text.
func AnnotateSendErrorDetails(details string) string {
	details = strings.TrimSpace(details)
	if details == "" {
		return details
	}
	payload := map[string]interface{}{}
	if err := json.Unmarshal([]byte(details), &payload); err != nil {
		return details
	}
	raw, ok := payload["error"].(string)
	if !ok || strings.TrimSpace(raw) == "" {
		return details
	}
	if reason, ok := payload["reason"].(string); ok && strings.TrimSpace(reason) != "" {
		return details
	}
	payload["reason"] = ClassifySendError(raw)
	out, err := json.Marshal(payload)
	if err != nil {
		return details
	}
	return string(out)
}

// AnnotateEvents fills missing send-error reasons on a campaign timeline.
func AnnotateEvents(events []Event) {
	for i := range events {
		events[i].Details = AnnotateSendErrorDetails(events[i].Details)
	}
}
