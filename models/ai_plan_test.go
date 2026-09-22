package models

import (
	"testing"
	"time"
)

func TestLaunchConfirmTokenRoundTrip(t *testing.T) {
	tok, exp, err := IssueLaunchConfirmToken(42, 7)
	if err != nil {
		t.Fatal(err)
	}
	if tok == "" || exp.Before(time.Now().UTC()) {
		t.Fatalf("bad token issue: %q %v", tok, exp)
	}
	if err := ConsumeLaunchConfirmToken(tok, 42, 7); err != nil {
		t.Fatal(err)
	}
	if err := ConsumeLaunchConfirmToken(tok, 42, 7); err != ErrLaunchTokenInvalid {
		t.Fatalf("expected single-use, got %v", err)
	}
}

func TestLaunchConfirmTokenMismatch(t *testing.T) {
	tok, _, err := IssueLaunchConfirmToken(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := ConsumeLaunchConfirmToken(tok, 9, 2); err != ErrLaunchTokenInvalid {
		t.Fatalf("expected mismatch error, got %v", err)
	}
}

func TestValidateAIPlanRequestStructural(t *testing.T) {
	req := &AIPlanRequest{}
	errs := ValidateAIPlanRequest(1, req)
	need := map[string]bool{
		"url is required": false,
		"channel.smtp_id or channel.sending_channel_id is required": false,
		"targets or group_id is required": false,
		"scenario_pack_id or both template_id and page_id are required": false,
	}
	for _, e := range errs {
		if _, ok := need[e]; ok {
			need[e] = true
		}
	}
	for msg, ok := range need {
		if !ok {
			t.Fatalf("missing expected error %q in %v", msg, errs)
		}
	}
}
