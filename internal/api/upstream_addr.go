package api

import (
	"net"
	"strings"
)

// 上游协议常量；与 DB CHECK 约束、forward 层分发保持同步。
const (
	UpstreamProtocolUDP = "udp"
	UpstreamProtocolDoT = "dot"
	UpstreamProtocolDoH = "doh"
)

// NormalizeUpstreamProtocol 大小写无关地归一协议字符串；空值视为 udp。
// 返回 (规范化值, 是否合法)。
func NormalizeUpstreamProtocol(s string) (string, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return UpstreamProtocolUDP, true
	}
	switch s {
	case UpstreamProtocolUDP, UpstreamProtocolDoT, UpstreamProtocolDoH:
		return s, true
	}
	return "", false
}

// NormalizeUpstreamAddr 校验上游地址须为合法 IPv4 或 IPv6（不接受主机名）。
// 允许输入带方括号的 IPv6，如 [2001:db8::1]；返回规范字符串（与 net.IP.String() 一致，IPv6 无方括号）。
// 三种协议都共享此校验，避免引入 bootstrap DNS 的"鸡生蛋"问题。
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

// NormalizeDoHPath 归一 DoH 端点路径：
//   - 空 / "/" 视为缺省 "/dns-query"。
//   - 必须以 "/" 开头，不含查询字符串或片段。
//   - 返回 (规范化路径, 是否合法)。
func NormalizeDoHPath(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" || s == "/" {
		return "/dns-query", true
	}
	if !strings.HasPrefix(s, "/") {
		return "", false
	}
	if strings.ContainsAny(s, "?#") {
		return "", false
	}
	return s, true
}

// NormalizeTLSServerName 校验可选 SNI；空字符串视为缺省（保持 nil 由调用方处理）。
// 仅做 RFC1123 风格的简单校验：每段 1-63 字符，由字母/数字/连字符组成，连字符不在首尾。
func NormalizeTLSServerName(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", true
	}
	if len(s) > 253 {
		return "", false
	}
	// 允许 IP 字面量直接做 SNI（部分服务端用 IP SAN 证书）。
	if ip := net.ParseIP(s); ip != nil {
		return ip.String(), true
	}
	for _, label := range strings.Split(strings.TrimSuffix(s, "."), ".") {
		if label == "" || len(label) > 63 {
			return "", false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return "", false
		}
		for i := 0; i < len(label); i++ {
			c := label[i]
			ok := (c >= 'a' && c <= 'z') ||
				(c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') ||
				c == '-'
			if !ok {
				return "", false
			}
		}
	}
	return s, true
}

// DefaultPortForProtocol 返回协议的常用默认端口。
func DefaultPortForProtocol(proto string) int {
	switch proto {
	case UpstreamProtocolDoT:
		return 853
	case UpstreamProtocolDoH:
		return 443
	default:
		return 53
	}
}
