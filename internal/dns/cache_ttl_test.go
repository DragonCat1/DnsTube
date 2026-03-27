package dns

import (
	"testing"
	"time"

	mdns "github.com/miekg/dns"
)

func TestAgeCachedMsgTTLs(t *testing.T) {
	m := new(mdns.Msg)
	m.Answer = []mdns.RR{
		mustRR(t, "a.example.com. 300 IN A 1.2.3.4"),
		mustRR(t, "a.example.com. 60 IN AAAA ::1"),
	}
	m.Ns = []mdns.RR{
		mustRR(t, "example.com. 120 IN NS ns1.example.com."),
	}
	m.Extra = []mdns.RR{
		mustRR(t, "ns1.example.com. 180 IN A 10.0.0.1"),
	}

	t0 := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	ageCachedMsgTTLs(m, t0, t0.Add(45*time.Second))

	assertTTL(t, m.Answer[0], 255)
	assertTTL(t, m.Answer[1], 15)
	assertTTL(t, m.Ns[0], 75)
	assertTTL(t, m.Extra[0], 135)

	m2 := new(mdns.Msg)
	m2.Answer = []mdns.RR{mustRR(t, "x.test. 10 IN A 9.9.9.9")}
	ageCachedMsgTTLs(m2, t0, t0.Add(12*time.Second))
	assertTTL(t, m2.Answer[0], 0)

	ageCachedMsgTTLs(nil, t0, t0.Add(time.Hour))
}

func TestCacheAgeSeconds(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := cacheAgeSeconds(t0, t0); got != 0 {
		t.Fatalf("same instant: got %d want 0", got)
	}
	if got := cacheAgeSeconds(t0, t0.Add(3*time.Second+500*time.Millisecond)); got != 3 {
		t.Fatalf("floor seconds: got %d want 3", got)
	}
	if got := cacheAgeSeconds(t0, t0.Add(-time.Second)); got != 0 {
		t.Fatalf("clock before cachedAt: got %d want 0", got)
	}
}

func mustRR(t *testing.T, s string) mdns.RR {
	t.Helper()
	rr, err := mdns.NewRR(s)
	if err != nil {
		t.Fatal(err)
	}
	return rr
}

func assertTTL(t *testing.T, rr mdns.RR, want uint32) {
	t.Helper()
	if rr.Header().Ttl != want {
		t.Fatalf("TTL got %d want %d", rr.Header().Ttl, want)
	}
}
