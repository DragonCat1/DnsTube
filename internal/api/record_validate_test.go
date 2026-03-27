package api

import "testing"

func TestRecordRDataErrCode(t *testing.T) {
	tests := []struct {
		rtype   string
		content string
		want    string
	}{
		{"A", "192.0.2.1", ""},
		{"A", "2001:db8::1", CodeDNSRecordIPv4Invalid},
		{"A", "not-an-ip", CodeDNSRecordIPv4Invalid},
		{"AAAA", "2001:db8::1", ""},
		{"AAAA", "::1", ""},
		{"AAAA", "192.0.2.1", CodeDNSRecordIPv6Invalid},
		{"AAAA", "not-an-ip", CodeDNSRecordIPv6Invalid},
		{"TXT", "anything", ""},
	}
	for _, tt := range tests {
		if got := recordRDataErrCode(tt.rtype, tt.content); got != tt.want {
			t.Errorf("recordRDataErrCode(%q, %q) = %q, want %q", tt.rtype, tt.content, got, tt.want)
		}
	}
}
