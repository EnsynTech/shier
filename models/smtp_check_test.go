package models

import (
	"net"
	"strings"
	"testing"
)

func TestDomainPolicyChecks(t *testing.T) {
	spf, dmarc := DomainPolicyChecks(
		[]string{"v=spf1 include:example.com -all"},
		nil,
		[]string{"v=DMARC1; p=quarantine"},
		nil,
	)
	if spf.Status != SMTPCheckPass || spf.ID != SMTPCheckSPF {
		t.Fatalf("spf = %+v", spf)
	}
	if !strings.Contains(spf.Detail, "v=spf1") {
		t.Fatalf("spf detail = %q", spf.Detail)
	}
	if dmarc.Status != SMTPCheckPass || !strings.Contains(dmarc.Detail, "v=DMARC1") {
		t.Fatalf("dmarc = %+v", dmarc)
	}

	missingSPF, missingDMARC := DomainPolicyChecks(nil, &net.DNSError{IsNotFound: true}, []string{"google-site-verification=abc"}, nil)
	if missingSPF.Status != SMTPCheckWarn || missingSPF.Reason != SMTPCheckMissing {
		t.Fatalf("missing spf = %+v", missingSPF)
	}
	if missingDMARC.Status != SMTPCheckWarn || missingDMARC.Reason != SMTPCheckMissing {
		t.Fatalf("txt without dmarc = %+v", missingDMARC)
	}

	failed, _ := DomainPolicyChecks(nil, &net.DNSError{Err: "server misbehaving", IsTemporary: true}, nil, nil)
	if failed.Status != SMTPCheckFail || failed.Reason != SendReasonDNS {
		t.Fatalf("dns failure = %+v", failed)
	}
}

func TestTruncateDetail(t *testing.T) {
	long := strings.Repeat("测", 200)
	got := truncateDetail(long)
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("truncated detail = %q", got)
	}
	if len([]rune(got)) != 181 {
		t.Fatalf("rune length = %d", len([]rune(got)))
	}
}
