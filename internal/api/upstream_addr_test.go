package api

import "testing"

func TestNormalizeUpstreamProtocol(t *testing.T) {
	tests := []struct {
		in     string
		want   string
		wantOk bool
	}{
		{"", "udp", true},
		{"  ", "udp", true},
		{"udp", "udp", true},
		{"UDP", "udp", true},
		{"DoT", "dot", true},
		{"doh", "doh", true},
		{"tls", "", false},
		{"https", "", false},
	}
	for _, tt := range tests {
		got, ok := NormalizeUpstreamProtocol(tt.in)
		if got != tt.want || ok != tt.wantOk {
			t.Errorf("NormalizeUpstreamProtocol(%q) = (%q,%v) want (%q,%v)", tt.in, got, ok, tt.want, tt.wantOk)
		}
	}
}

func TestNormalizeDoHPath(t *testing.T) {
	tests := []struct {
		in     string
		want   string
		wantOk bool
	}{
		{"", "/dns-query", true},
		{"/", "/dns-query", true},
		{"/dns-query", "/dns-query", true},
		{"/resolve", "/resolve", true},
		{"dns-query", "", false},
		{"/path?x=1", "", false},
		{"/path#frag", "", false},
	}
	for _, tt := range tests {
		got, ok := NormalizeDoHPath(tt.in)
		if got != tt.want || ok != tt.wantOk {
			t.Errorf("NormalizeDoHPath(%q) = (%q,%v) want (%q,%v)", tt.in, got, ok, tt.want, tt.wantOk)
		}
	}
}

func TestNormalizeTLSServerName(t *testing.T) {
	tests := []struct {
		in     string
		want   string
		wantOk bool
	}{
		{"", "", true},
		{"dns.google", "dns.google", true},
		{"a.b.c.d.example.com.", "a.b.c.d.example.com.", true},
		{"1.1.1.1", "1.1.1.1", true},
		{"-bad.example", "", false},
		{"bad-.example", "", false},
		{"contains space.example", "", false},
		{"toolongLabel" + repeat("a", 60) + ".example", "", false},
	}
	for _, tt := range tests {
		got, ok := NormalizeTLSServerName(tt.in)
		if got != tt.want || ok != tt.wantOk {
			t.Errorf("NormalizeTLSServerName(%q) = (%q,%v) want (%q,%v)", tt.in, got, ok, tt.want, tt.wantOk)
		}
	}
}

func TestDefaultPortForProtocol(t *testing.T) {
	cases := map[string]int{"udp": 53, "dot": 853, "doh": 443, "anything": 53}
	for p, want := range cases {
		if got := DefaultPortForProtocol(p); got != want {
			t.Errorf("DefaultPortForProtocol(%q)=%d want %d", p, got, want)
		}
	}
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

func TestNormalizeUpstreamAddr(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantOk  bool
	}{
		{"8.8.8.8", "8.8.8.8", true},
		{"  1.1.1.1  ", "1.1.1.1", true},
		{"2001:db8::1", "2001:db8::1", true},
		{"[2001:db8::1]", "2001:db8::1", true},
		{"::1", "::1", true},
		{"example.com", "", false},
		{"", "", false},
		{"256.1.1.1", "", false},
	}
	for _, tt := range tests {
		got, ok := NormalizeUpstreamAddr(tt.in)
		if ok != tt.wantOk || got != tt.want {
			t.Errorf("NormalizeUpstreamAddr(%q) = (%q, %v), want (%q, %v)", tt.in, got, ok, tt.want, tt.wantOk)
		}
	}
}
