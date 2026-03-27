package api

import (
	"net"
	"strings"
)

// NormalizeUpstreamAddr 校验上游地址须为合法 IPv4 或 IPv6（不接受主机名）。
// 允许输入带方括号的 IPv6，如 [2001:db8::1]；返回规范字符串（与 net.IP.String() 一致，IPv6 无方括号）。
func NormalizeUpstreamAddr(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	if len(s) >= 2 && s[0] == '[' {
		i := strings.LastIndex(s, "]")
		if i <= 1 {
			return "", false
		}
		inner := s[1:i]
		ip := net.ParseIP(inner)
		if ip == nil {
			return "", false
		}
		return ip.String(), true
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return "", false
	}
	return ip.String(), true
}
