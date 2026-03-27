package api

import (
	"net"
	"strings"
)

// recordRDataErrCode 校验静态记录正文：A 须为合法 IPv4，AAAA 须为合法 IPv6；其它类型不校验。
// 合法时返回空字符串；不合法时返回对应错误码常量。
func recordRDataErrCode(rtype, content string) string {
	switch strings.TrimSpace(strings.ToUpper(rtype)) {
	case "A":
		ip := net.ParseIP(strings.TrimSpace(content))
		if ip != nil && ip.To4() != nil {
			return ""
		}
		return CodeDNSRecordIPv4Invalid
	case "AAAA":
		ip := net.ParseIP(strings.TrimSpace(content))
		if ip == nil {
			return CodeDNSRecordIPv6Invalid
		}
		if ip.To4() != nil {
			return CodeDNSRecordIPv6Invalid
		}
		return ""
	default:
		return ""
	}
}
