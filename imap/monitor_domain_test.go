package imap

import (
	"testing"

	"github.com/jordan-wright/email"
)

func TestIsForwardSubject(t *testing.T) {
	cases := []struct {
		subject string
		want    bool
	}{
		{subject: "Fwd: 【需确认】请点击完成系统升级确认", want: true},
		{subject: "  FW: weekly report", want: true},
		{subject: "转发：工资核算确认表", want: true},
		{subject: "轉寄: 会议", want: true},
		{subject: "Re: Fwd: 【需确认】请点击完成系统升级确认", want: true},
		{subject: "回复：转发：工资核算确认表", want: true},
		{subject: "[转发] 请确认", want: true},
		{subject: "自动转发: 请确认", want: true},
		{subject: "Re: 【需确认】请点击完成系统升级确认", want: false},
		{subject: "【需确认】请点击完成系统升级确认", want: false},
	}
	for _, tc := range cases {
		if got := isForwardSubject(tc.subject); got != tc.want {
			t.Errorf("isForwardSubject(%q)=%v want %v", tc.subject, got, tc.want)
		}
	}
}

func TestReportSourceSeparatesReplyFromReport(t *testing.T) {
	quoted := &email.Email{
		Subject: "Re: 会议",
		Text:    []byte("好的，我会参加\n> http://phish.example/?rid=AbC1234\n"),
	}
	if got := reportSource(quoted, false, true, map[string]bool{"AbC1234": true}); got != "" {
		t.Fatalf("ordinary reply source=%q", got)
	}
	complaint := &email.Email{
		Subject: "Re: 会议",
		Text:    []byte("这是钓鱼邮件\n> http://phish.example/?rid=AbC1234\n"),
	}
	if got := reportSource(complaint, false, true, map[string]bool{"AbC1234": true}); got != reportSourceComplaint {
		t.Fatalf("complaint source=%q", got)
	}
	forward := &email.Email{Subject: "Fwd: 请确认"}
	if got := reportSource(forward, true, false, map[string]bool{"AbC1234": true}); got != reportSourceForward {
		t.Fatalf("forward source=%q", got)
	}
}

func TestAddRIDsReadsEncodedRID(t *testing.T) {
	rids := map[string]bool{}
	addRIDs("see http://safe.example/?u=http%253A%252F%252Fphish%252F%253Frid%253DAbC1234", rids)
	if !rids["AbC1234"] {
		t.Fatalf("rids=%v", rids)
	}
}

func TestSenderDomain(t *testing.T) {
	cases := []struct {
		from string
		want string
	}{
		{from: "drill@shier.local", want: "shier.local"},
		{from: "<drill@shier.local>", want: "shier.local"},
		{from: "财务甲 <fin@example.com>", want: "example.com"},
		{from: "", want: ""},
	}
	for _, tc := range cases {
		if got := senderDomain(tc.from); got != tc.want {
			t.Errorf("senderDomain(%q)=%q want %q", tc.from, got, tc.want)
		}
	}
}
