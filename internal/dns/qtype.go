package dns

import (
	"strconv"
	"strings"

	mdns "github.com/miekg/dns"
)

func qtypeString(q uint16) string {
	if s, ok := mdns.TypeToString[q]; ok {
		return s
	}
	return "TYPE" + strconv.FormatUint(uint64(q), 10)
}

func typeFromRecord(s string) uint16 {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "A":
		return mdns.TypeA
	case "AAAA":
		return mdns.TypeAAAA
	case "CNAME":
		return mdns.TypeCNAME
	case "TXT":
		return mdns.TypeTXT
	case "NS":
		return mdns.TypeNS
	case "MX":
		return mdns.TypeMX
	case "PTR":
		return mdns.TypePTR
	case "SRV":
		return mdns.TypeSRV
	default:
		return mdns.TypeNone
	}
}
