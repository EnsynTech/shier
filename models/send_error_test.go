package models

import (
	"encoding/json"
	"testing"
	"time"

	"gopkg.in/check.v1"
)

func TestClassifySendError(t *testing.T) {
	cases := []struct {
		msg  string
		want string
	}{
		{"", SendReasonUnknown},
		{"max send attempts exceeded", SendReasonMaxAttempts},
		{"rendered body is not valid JSON", SendReasonInvalidRequest},
		{"Phone number not specified", SendReasonInvalidRequest},
		{"invalid body template: missing end", SendReasonInvalidRequest},
		{"gateway returned status 502", SendReasonGatewayStatus},
		{"x509: certificate signed by unknown authority", SendReasonTLSCert},
		{"dial tcp: lookup smtp.example.com: no such host", SendReasonDNS},
		{"535 5.7.8 authentication failed", SendReasonAuthFailed},
		{"421 4.7.0 try again later", SendReasonRateLimited},
		{"450 too many connections", SendReasonRateLimited},
		{"550 user unknown", SendReasonMailboxRejected},
		{"500 Recipient not found", SendReasonMailboxRejected},
		{"550 relay access denied", SendReasonPolicyRejected},
		{"554 message rejected as spam", SendReasonPolicyRejected},
		{"dial tcp 127.0.0.1:25: connection refused", SendReasonConnect},
		{"Max connection attempts exceeded - i/o timeout", SendReasonConnect},
		{"upstream connection denied to internal host", SendReasonConnect},
		{"something unexpected happened", SendReasonUnknown},
	}
	for _, tc := range cases {
		if got := ClassifySendError(tc.msg); got != tc.want {
			t.Errorf("ClassifySendError(%q) = %q, want %q", tc.msg, got, tc.want)
		}
	}
}

func TestAnnotateSendErrorDetails(t *testing.T) {
	got := AnnotateSendErrorDetails(`{"error":"535 authentication failed"}`)
	payload := map[string]string{}
	if err := json.Unmarshal([]byte(got), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["reason"] != SendReasonAuthFailed {
		t.Fatalf("reason = %q", payload["reason"])
	}
	if payload["error"] != "535 authentication failed" {
		t.Fatalf("error text changed: %q", payload["error"])
	}

	already := `{"error":"550 user unknown","reason":"mailbox_rejected"}`
	if AnnotateSendErrorDetails(already) != already {
		t.Fatal("existing reason was rewritten")
	}
	if AnnotateSendErrorDetails(`{"browser":{"user-agent":"x"}}`) != `{"browser":{"user-agent":"x"}}` {
		t.Fatal("non-error details were rewritten")
	}
	if AnnotateSendErrorDetails("not-json") != "not-json" {
		t.Fatal("invalid JSON was rewritten")
	}
}

func (s *ModelsSuite) TestLegacySendErrorIsAnnotatedOnRead(c *check.C) {
	campaign := s.createCampaign(c)
	raw, err := json.Marshal(map[string]string{"error": "535 authentication failed"})
	c.Assert(err, check.IsNil)
	ev := &Event{
		CampaignId: campaign.Id,
		Email:      "test1@example.com",
		Message:    EventSendingError,
		Details:    string(raw),
		Time:       time.Now().UTC(),
	}
	c.Assert(db.Save(ev).Error, check.IsNil)
	got, err := GetCampaign(campaign.Id, campaign.UserId)
	c.Assert(err, check.IsNil)
	var found *Event
	for i := range got.Events {
		if got.Events[i].Message == EventSendingError {
			found = &got.Events[i]
			break
		}
	}
	if found == nil {
		c.Fatal("send error event was not loaded")
	}
	payload := map[string]string{}
	c.Assert(json.Unmarshal([]byte(found.Details), &payload), check.IsNil)
	c.Assert(payload["reason"], check.Equals, SendReasonAuthFailed)

	var stored Event
	c.Assert(db.Where("id = ?", ev.Id).First(&stored).Error, check.IsNil)
	storedPayload := map[string]string{}
	c.Assert(json.Unmarshal([]byte(stored.Details), &storedPayload), check.IsNil)
	if _, ok := storedPayload["reason"]; ok {
		c.Fatal("read path wrote the reason back to the database")
	}
}

func (s *ModelsSuite) TestSendSuccessRateExcludesRetrying(c *check.C) {
	campaign := s.createCampaign(c)
	var results []Result
	c.Assert(db.Where("campaign_id = ?", campaign.Id).Order("email").Find(&results).Error, check.IsNil)
	c.Assert(len(results) >= 4, check.Equals, true)
	results[0].Status = EventSent
	results[1].Status = EventOpened
	results[2].Status = Error
	results[3].Status = StatusRetry
	for i := 0; i < 4; i++ {
		c.Assert(db.Save(&results[i]).Error, check.IsNil)
	}
	stats, err := getCampaignStats(campaign.Id)
	c.Assert(err, check.IsNil)
	// Opened implies sent, so two people count as delivered.
	c.Assert(stats.EmailsSent, check.Equals, int64(2))
	c.Assert(stats.Error, check.Equals, int64(1))
	c.Assert(stats.Retrying, check.Equals, int64(1))
	if stats.SendSuccessRate == nil {
		c.Fatal("expected a send success rate")
	}
	c.Assert(*stats.SendSuccessRate, check.Equals, float64(2)/float64(3))

	for i := range results {
		results[i].Status = StatusScheduled
		c.Assert(db.Save(&results[i]).Error, check.IsNil)
	}
	pending, err := getCampaignStats(campaign.Id)
	c.Assert(err, check.IsNil)
	c.Assert(pending.SendSuccessRate == nil, check.Equals, true)
}
