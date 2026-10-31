package dns

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"

	"github.com/dnstube/dnstube/internal/store"
)

// ruleCompileCache 缓存单条规则的编译结果，供 Reload 时按内容指纹复用。
type ruleCompileCache struct {
	fp      string
	regexps []*regexp.Regexp
	domains []string
}

// ruleCompileFingerprint 只含影响匹配编译结果的内容字段；不含 pattern_fetched_at / hit_count。
func ruleCompileFingerprint(r store.ForwardRule) string {
	h := sha256.New()
	write := func(s string) {
		_, _ = h.Write([]byte(s))
		_, _ = h.Write([]byte{0})
	}
	write(r.PatternSource)
	write(r.NamePattern)
	write(r.PatternFormat)
	if r.PatternURL != nil {
		write(*r.PatternURL)
	} else {
		write("")
	}
	if r.PatternFetchedBody != nil {
		write(*r.PatternFetchedBody)
	} else {
		write("")
	}
	_, _ = h.Write(r.PatternResolvedEntries)
	_, _ = h.Write([]byte{0})
	write(fmt.Sprintf("%t", r.Disabled))
	write(fmt.Sprintf("%d", r.TargetGroupID))
	write(r.Mode)
	write(fmt.Sprintf("%d", r.Priority))
	return hex.EncodeToString(h.Sum(nil))
}
