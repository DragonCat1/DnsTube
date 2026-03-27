package dns

import (
	"fmt"
	"strings"

	mdns "github.com/miekg/dns"
)

const maxResultSummaryLen = 1024

// AnswerSummaryForLog 生成写入查询日志的「结果」摘要（展示用）。
func AnswerSummaryForLog(msg *mdns.Msg) string {
	if msg == nil {
		return ""
	}
	if len(msg.Answer) == 0 {
		switch msg.Rcode {
		case mdns.RcodeSuccess:
			return "—"
		case mdns.RcodeNameError:
			return "NXDOMAIN"
		default:
			if s, ok := mdns.RcodeToString[msg.Rcode]; ok {
				return s
			}
			return "—"
		}
	}
	var parts []string
	for _, rr := range msg.Answer {
		p := rrShort(rr)
		if p == "" {
			continue
		}
		parts = append(parts, p)
	}
	return truncateRunes(strings.Join(parts, "; "), maxResultSummaryLen)
}

func rrShort(rr mdns.RR) string {
	switch x := rr.(type) {
	case *mdns.A:
		return x.A.String()
	case *mdns.AAAA:
		return x.AAAA.String()
	case *mdns.CNAME:
		return strings.TrimSuffix(mdns.Fqdn(x.Target), ".")
	case *mdns.PTR:
		return strings.TrimSuffix(mdns.Fqdn(x.Ptr), ".")
	case *mdns.MX:
		return fmt.Sprintf("%d %s", x.Preference, strings.TrimSuffix(mdns.Fqdn(x.Mx), "."))
	case *mdns.TXT:
		return strings.Join(x.Txt, " ")
	case *mdns.NS:
		return strings.TrimSuffix(mdns.Fqdn(x.Ns), ".")
	case *mdns.SRV:
		return fmt.Sprintf("%d %d %d %s", x.Priority, x.Weight, x.Port, strings.TrimSuffix(mdns.Fqdn(x.Target), "."))
	case *mdns.SOA:
		return fmt.Sprintf("SOA %s", strings.TrimSuffix(mdns.Fqdn(x.Ns), "."))
	default:
		s := rr.String()
		parts := strings.Fields(s)
		if len(parts) >= 5 {
			return strings.Join(parts[4:], " ")
		}
		return s
	}
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

func summarizeForQueryLog(errMsg *string, respMsg *mdns.Msg) *string {
	if errMsg != nil && *errMsg != "" {
		t := truncateRunes(*errMsg, maxResultSummaryLen)
		return &t
	}
	if respMsg != nil {
		t := AnswerSummaryForLog(respMsg)
		return &t
	}
	t := "NXDOMAIN"
	return &t
}
