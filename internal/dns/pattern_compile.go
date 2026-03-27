package dns

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/dnstube/dnstube/internal/store"
)

const maxPatternBody = 4 << 20 // 4 MiB

func fetchURLBody(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "DnsTube/1.0")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxPatternBody))
}

func decodeBase64Flexible(b []byte) ([]byte, error) {
	s := strings.TrimSpace(string(b))
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\t", "")
	var lastErr error
	for _, dec := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	} {
		raw, err := dec.DecodeString(s)
		if err == nil {
			return raw, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func maybeGunzip(b []byte) ([]byte, error) {
	if len(b) >= 2 && b[0] == 0x1f && b[1] == 0x8b {
		zr, err := gzip.NewReader(bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		defer zr.Close()
		return io.ReadAll(io.LimitReader(zr, maxPatternBody))
	}
	return b, nil
}

func compileRegexList(content string) ([]*regexp.Regexp, error) {
	var out []*regexp.Regexp
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		re, err := regexp.Compile(line)
		if err != nil {
			return nil, fmt.Errorf("regex line %q: %w", line, err)
		}
		out = append(out, re)
	}
	return out, nil
}

func autoproxyLineToRegexp(line string) (re *regexp.Regexp, exactDomain string, err error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "!") || strings.HasPrefix(line, "[") {
		return nil, "", nil
	}
	// 白名单规则：当前架构"匹配=转发"，无"不转发"语义，跳过。
	if strings.HasPrefix(line, "@@") {
		return nil, "", nil
	}
	// /regex/ 正则规则：去首尾 / 后原样编译。
	if len(line) > 2 && line[0] == '/' && line[len(line)-1] == '/' {
		re, err = regexp.Compile(line[1 : len(line)-1])
		return re, "", err
	}
	// ||domain 域名锚定（支持通配符 *）
	if strings.HasPrefix(line, "||") {
		return autoproxyDomainAnchor(line[2:])
	}
	// |http(s)://host URL 锚定：提取 host 部分做域名匹配
	if strings.HasPrefix(line, "|http://") || strings.HasPrefix(line, "|https://") {
		return autoproxyURLAnchor(line[1:])
	}
	// plain domain / keyword 兜底
	return autoproxyPlain(line)
}

// autoproxyDomainAnchor 处理 ||domain^ 形式，支持通配符 *。
// 纯域名（无 *）时返回 (nil, domain, nil)，不编译正则。
func autoproxyDomainAnchor(rest string) (*regexp.Regexp, string, error) {
	// 截断到 ^ 或 / 处（保留 *）
	for i, c := range rest {
		if c == '^' || c == '/' {
			rest = rest[:i]
			break
		}
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return nil, "", nil
	}
	if !strings.Contains(rest, "*") {
		return nil, strings.ToLower(rest), nil
	}
	pat := domainPatternToRegexp(rest)
	re, err := regexp.Compile(`(?i)(^|\.)` + pat + `$`)
	return re, "", err
}

// autoproxyURLAnchor 从 URL 中提取 host 做域名匹配。
// 纯域名（无 *）时返回 (nil, domain, nil)，不编译正则。
func autoproxyURLAnchor(rawURL string) (*regexp.Regexp, string, error) {
	idx := strings.Index(rawURL, "://")
	if idx < 0 {
		return nil, "", nil
	}
	host := rawURL[idx+3:]
	for i, c := range host {
		if c == '/' || c == '^' || c == ':' {
			host = host[:i]
			break
		}
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return nil, "", nil
	}
	if !strings.Contains(host, "*") {
		return nil, strings.ToLower(host), nil
	}
	pat := domainPatternToRegexp(host)
	re, err := regexp.Compile(`(?i)(^|\.)` + pat + `$`)
	return re, "", err
}

// autoproxyPlain 处理裸域名或关键字行。
// 含 . 且无 * 的域名返回 (nil, domain, nil)，不编译正则；关键字仍编译正则。
func autoproxyPlain(line string) (*regexp.Regexp, string, error) {
	// 含 . 视为域名
	if strings.Contains(line, ".") {
		for i, c := range line {
			if c == '/' || c == '^' {
				line = line[:i]
				break
			}
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return nil, "", nil
		}
		if !strings.Contains(line, "*") {
			return nil, strings.ToLower(line), nil
		}
		pat := domainPatternToRegexp(line)
		re, err := regexp.Compile(`(?i)(^|\.)` + pat + `$`)
		return re, "", err
	}
	// 不含 . 视为关键字，做 contains 匹配
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, "", nil
	}
	pat := domainPatternToRegexp(line)
	re, err := regexp.Compile(`(?i)` + pat)
	return re, "", err
}

// domainPatternToRegexp 将域名模式（可能含 *）转为正则片段。
func domainPatternToRegexp(s string) string {
	if !strings.Contains(s, "*") {
		return regexp.QuoteMeta(s)
	}
	parts := strings.Split(s, "*")
	for i, p := range parts {
		parts[i] = regexp.QuoteMeta(p)
	}
	return strings.Join(parts, ".*")
}

// compileAutoproxyWithLines 返回编译后的正则，以及每条对应的展示字符串。
// 用于 save/display 路径（CompilePatternBytes）。纯域名展示为 "[域名] domain"，其余展示 re.String()。
func compileAutoproxyWithLines(content string) ([]*regexp.Regexp, []string, error) {
	var res []*regexp.Regexp
	var patterns []string
	for _, line := range strings.Split(content, "\n") {
		re, domain, err := autoproxyLineToRegexp(line)
		if err != nil {
			return nil, nil, fmt.Errorf("autoproxy line %q: %w", line, err)
		}
		if re != nil {
			res = append(res, re)
			patterns = append(patterns, re.String())
		} else if domain != "" {
			// 纯域名：补编译正则供条数统计，展示为 [域名] domain
			pat := `(?i)(^|\.)` + regexp.QuoteMeta(domain) + `$`
			compiled, err := regexp.Compile(pat)
			if err != nil {
				return nil, nil, fmt.Errorf("autoproxy domain %q: %w", domain, err)
			}
			res = append(res, compiled)
			patterns = append(patterns, "[域名] "+domain)
		}
	}
	return res, patterns, nil
}

// compileAutoproxyForEngine 引擎专用：纯域名不编译正则，分离为 domains 列表。
func compileAutoproxyForEngine(content string) (regexps []*regexp.Regexp, domains []string, err error) {
	for _, line := range strings.Split(content, "\n") {
		re, domain, lineErr := autoproxyLineToRegexp(line)
		if lineErr != nil {
			return nil, nil, fmt.Errorf("autoproxy line %q: %w", line, lineErr)
		}
		if re != nil {
			regexps = append(regexps, re)
		} else if domain != "" {
			domains = append(domains, domain)
		}
	}
	return regexps, domains, nil
}

func compileAutoproxy(content string) ([]*regexp.Regexp, error) {
	p, _, err := compileAutoproxyWithLines(content)
	return p, err
}

// CompilePatternBytes 按 pattern_format 解析原始字节；autoproxy 两类第三返回值非 nil 时为各条 re.String()（与匹配一致）。
func CompilePatternBytes(patternFormat string, raw []byte) ([]*regexp.Regexp, int, []string, error) {
	if len(raw) > maxPatternBody {
		return nil, 0, nil, fmt.Errorf("pattern body exceeds max size")
	}
	fmtKind := strings.TrimSpace(patternFormat)
	if fmtKind == "" {
		return nil, 0, nil, fmt.Errorf("empty pattern_format")
	}
	switch fmtKind {
	case "regex_list":
		pats, err := compileRegexList(string(raw))
		if err != nil {
			return nil, 0, nil, err
		}
		return pats, len(pats), nil, nil
	case "autoproxy":
		pats, lines, err := compileAutoproxyWithLines(string(raw))
		if err != nil {
			return nil, 0, nil, err
		}
		return pats, len(pats), lines, nil
	case "autoproxy_base64":
		dec, err := decodeBase64Flexible(raw)
		if err != nil {
			return nil, 0, nil, err
		}
		dec, err = maybeGunzip(dec)
		if err != nil {
			return nil, 0, nil, err
		}
		if len(dec) > maxPatternBody {
			return nil, 0, nil, fmt.Errorf("decoded pattern body exceeds max size")
		}
		pats, lines, err := compileAutoproxyWithLines(string(dec))
		if err != nil {
			return nil, 0, nil, err
		}
		return pats, len(pats), lines, nil
	default:
		return nil, 0, nil, fmt.Errorf("pattern_format %q not valid for list body", fmtKind)
	}
}

func compilePatternsFromURLBody(r store.ForwardRule, raw []byte) ([]*regexp.Regexp, int, []string, error) {
	return CompilePatternBytes(r.PatternFormat, raw)
}

// EncodedResolvedEntries 将 autoproxy 类解析出的正则模式串序列化为 JSON；lines 为 nil 时不写入 JSONB（传 nil 给 store）。
func EncodedResolvedEntries(lines []string) ([]byte, error) {
	if lines == nil {
		return nil, nil
	}
	return json.Marshal(lines)
}

func patternEffectiveBody(r *store.ForwardRule) string {
	if r.PatternSource == "url" {
		if r.PatternFetchedBody != nil {
			return *r.PatternFetchedBody
		}
		return ""
	}
	return r.NamePattern
}

func patternRawBytesForCompile(r *store.ForwardRule, fmtKind string) ([]byte, error) {
	switch fmtKind {
	case "autoproxy_base64":
		if r.PatternSource == "url" {
			if r.PatternFetchedBody == nil || strings.TrimSpace(*r.PatternFetchedBody) == "" {
				return nil, fmt.Errorf("no cached pattern body")
			}
			return []byte(strings.TrimSpace(*r.PatternFetchedBody)), nil
		}
		s := strings.TrimSpace(r.NamePattern)
		if s == "" {
			return nil, fmt.Errorf("empty name_pattern")
		}
		return []byte(s), nil
	case "autoproxy", "regex_list":
		text := patternEffectiveBody(r)
		if strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("no pattern content")
		}
		return []byte(text), nil
	default:
		return nil, fmt.Errorf("unsupported format %q", fmtKind)
	}
}

func regexListDisplayLines(content string) []string {
	var out []string
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		out = append(out, t)
	}
	return out
}

// PatternEntryLinesForAdmin 供管理 API 展示规则条目（autoproxy 类优先使用已入库的 re.String() 列表）。
func PatternEntryLinesForAdmin(r store.ForwardRule) ([]string, error) {
	fmtKind := strings.TrimSpace(r.PatternFormat)
	if fmtKind == "" {
		return nil, fmt.Errorf("empty pattern_format")
	}
	switch fmtKind {
	case "regex_list":
		text := patternEffectiveBody(&r)
		if strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("no pattern content")
		}
		return regexListDisplayLines(text), nil
	case "autoproxy", "autoproxy_base64":
		if len(r.PatternResolvedEntries) > 0 {
			var out []string
			if err := json.Unmarshal(r.PatternResolvedEntries, &out); err == nil {
				return out, nil
			}
		}
		raw, err := patternRawBytesForCompile(&r, fmtKind)
		if err != nil {
			return nil, err
		}
		_, _, lines, err := CompilePatternBytes(fmtKind, raw)
		return lines, err
	default:
		return nil, fmt.Errorf("unknown pattern_format %q", fmtKind)
	}
}

// CompileForwardRuleForSave 根据已保存字段计算条数与待写入的 pattern_resolved_entries（JSON，autoproxy 类为各条匹配用正则模式串）；regex_list 时 resolved 为 nil。
func CompileForwardRuleForSave(r store.ForwardRule) (ruleCount int, resolvedJSON []byte, err error) {
	fmtKind := strings.TrimSpace(r.PatternFormat)
	if fmtKind == "" {
		return 0, nil, fmt.Errorf("empty pattern_format")
	}
	src := r.PatternSource
	if src == "" {
		src = "inline"
	}
	switch fmtKind {
	case "regex_list", "autoproxy", "autoproxy_base64":
		var raw []byte
		if src == "inline" {
			if strings.TrimSpace(r.NamePattern) == "" {
				return 0, nil, fmt.Errorf("empty name_pattern")
			}
			if fmtKind == "autoproxy_base64" {
				raw = []byte(strings.TrimSpace(r.NamePattern))
			} else {
				raw = []byte(r.NamePattern)
			}
		} else {
			if r.PatternFetchedBody == nil || strings.TrimSpace(*r.PatternFetchedBody) == "" {
				return 0, nil, fmt.Errorf("no cached pattern body")
			}
			raw = []byte(*r.PatternFetchedBody)
		}
		_, n, lines, err := CompilePatternBytes(fmtKind, raw)
		if err != nil {
			return 0, nil, err
		}
		if lines != nil {
			b, err := json.Marshal(lines)
			if err != nil {
				return 0, nil, err
			}
			return n, b, nil
		}
		return n, nil, nil
	default:
		return 0, nil, fmt.Errorf("unknown pattern_format %q", fmtKind)
	}
}

// compilePatternsForEngine 引擎专用：返回分离的正则和纯域名列表。
func compilePatternsForEngine(ctx context.Context, r store.ForwardRule) (regexps []*regexp.Regexp, domains []string, err error) {
	_ = ctx
	src := r.PatternSource
	if src == "" {
		src = "inline"
	}
	fmtKind := strings.TrimSpace(r.PatternFormat)
	if fmtKind == "" {
		return nil, nil, fmt.Errorf("empty pattern_format")
	}

	var raw []byte
	if src == "inline" {
		if strings.TrimSpace(r.NamePattern) == "" {
			return nil, nil, fmt.Errorf("empty name_pattern")
		}
		if fmtKind == "autoproxy_base64" {
			raw = []byte(strings.TrimSpace(r.NamePattern))
		} else {
			raw = []byte(r.NamePattern)
		}
	} else {
		if r.PatternURL == nil || strings.TrimSpace(*r.PatternURL) == "" {
			return nil, nil, fmt.Errorf("pattern_url required")
		}
		if r.PatternFetchedBody == nil || strings.TrimSpace(*r.PatternFetchedBody) == "" {
			return nil, nil, fmt.Errorf("no cached pattern body")
		}
		raw = []byte(*r.PatternFetchedBody)
	}

	switch fmtKind {
	case "regex_list":
		pats, compErr := compileRegexList(string(raw))
		return pats, nil, compErr
	case "autoproxy":
		return compileAutoproxyForEngine(string(raw))
	case "autoproxy_base64":
		dec, decErr := decodeBase64Flexible(raw)
		if decErr != nil {
			return nil, nil, decErr
		}
		dec, decErr = maybeGunzip(dec)
		if decErr != nil {
			return nil, nil, decErr
		}
		if len(dec) > maxPatternBody {
			return nil, nil, fmt.Errorf("decoded pattern body exceeds max size")
		}
		return compileAutoproxyForEngine(string(dec))
	default:
		return nil, nil, fmt.Errorf("pattern_format %q not valid", fmtKind)
	}
}
