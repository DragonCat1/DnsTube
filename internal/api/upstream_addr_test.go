package api

import "testing"

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
