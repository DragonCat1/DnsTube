package dns

import (
	"context"
	"fmt"
	"strings"

	"github.com/dnstube/dnstube/internal/store"
)

// FetchURLPattern 从 pattern_url 拉取并解析规则；成功时返回源文件内容、规则条数，以及 AutoProxy 类格式的展示行（非 AutoProxy 时为 nil）。
func FetchURLPattern(ctx context.Context, r store.ForwardRule) (body string, ruleCount int, autoproxyLines []string, err error) {
	if r.PatternURL == nil || strings.TrimSpace(*r.PatternURL) == "" {
		return "", 0, nil, fmt.Errorf("pattern_url required")
	}
	raw, err := fetchURLBody(ctx, strings.TrimSpace(*r.PatternURL))
	if err != nil {
		return "", 0, nil, err
	}
	_, n, lines, err := compilePatternsFromURLBody(r, raw)
	if err != nil {
		return "", 0, nil, err
	}
	return string(raw), n, lines, err
}
