package dns

import (
	"regexp"
	"testing"

	"github.com/dnstube/dnstube/internal/store"
)

func TestCompileAutoproxyWithLines(t *testing.T) {
	t.Parallel()
	p, lines, err := compileAutoproxyWithLines("!comment\n||example.com^\n||foo.org/path\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(p) != 2 || len(lines) != 2 {
		t.Fatalf("got %d regexps %d lines", len(p), len(lines))
	}
	// 纯域名展示为 "[域名] domain"
	want := []string{"[域名] example.com", "[域名] foo.org"}
	for i := range lines {
		if lines[i] != want[i] {
			t.Fatalf("pattern %d: want %q got %q", i, want[i], lines[i])
		}
	}
}

func TestCompilePatternBytesRegexList(t *testing.T) {
	t.Parallel()
	raw := []byte("#c\n^foo$\n")
	p, n, lines, err := CompilePatternBytes("regex_list", raw)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 || lines != nil {
		t.Fatalf("n=%d lines=%v", n, lines)
	}
	if len(p) != 1 || p[0].String() != "^foo$" {
		t.Fatalf("regexp %v", p[0])
	}
}

func TestCompileForwardRuleForSaveInlineAutoproxy(t *testing.T) {
	t.Parallel()
	r := store.ForwardRule{
		PatternSource: "inline",
		PatternFormat: "autoproxy",
		NamePattern:   "||a.com^\n||b.com^",
	}
	n, j, err := CompileForwardRuleForSave(r)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 || len(j) == 0 {
		t.Fatalf("n=%d j=%s", n, string(j))
	}
}

func TestPatternEntryLinesForAdminRegexListSingleLine(t *testing.T) {
	t.Parallel()
	r := store.ForwardRule{
		PatternSource: "inline",
		PatternFormat: "regex_list",
		NamePattern:   ".*\\.x$",
	}
	lines, err := PatternEntryLinesForAdmin(r)
	if err != nil || len(lines) != 1 || lines[0] != ".*\\.x$" {
		t.Fatalf("%v %v", lines, err)
	}
}

func TestPatternEntryLinesForAdminRegexList(t *testing.T) {
	t.Parallel()
	r := store.ForwardRule{
		PatternSource: "inline",
		PatternFormat: "regex_list",
		NamePattern:   "#h\n^a$\n",
	}
	lines, err := PatternEntryLinesForAdmin(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || lines[0] != "^a$" {
		t.Fatalf("%#v", lines)
	}
}

func TestAutoproxyLineToRegexp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		line       string
		wantNil    bool   // 期望 re==nil && domain==""（跳过）
		wantDomain string // 期望返回 exactDomain（纯域名，re==nil）
		match      string // 应匹配的域名（仅 re!=nil 时）
		noMatch    string // 不应匹配的域名（仅 re!=nil 时）
	}{
		// — 跳过 —
		{name: "empty", line: "", wantNil: true},
		{name: "comment", line: "!this is a comment", wantNil: true},
		{name: "header", line: "[AutoProxy 0.2.1]", wantNil: true},
		{name: "whitelist", line: "@@||example.com", wantNil: true},

		// — /regex/ —
		{name: "regex_domain", line: "/example\\.com/", match: "example.com", noMatch: "exampleXcom"},
		{name: "regex_complex", line: "/^(.*\\.)?blogspot\\./", match: "foo.blogspot.com"},

		// — ||domain 域名锚定（纯域名 → exactDomain）—
		{name: "domain_anchor", line: "||example.com^", wantDomain: "example.com"},
		{name: "domain_anchor_noslash", line: "||foo.org/path", wantDomain: "foo.org"},

		// — ||domain 域名锚定（含通配符 → regexp）—
		{name: "domain_wildcard", line: "||boxun*.azurewebsites.net", match: "boxun123.azurewebsites.net", noMatch: "other.net"},
		{name: "domain_wildcard2", line: "||cdn*.example.com^", match: "cdn-images.example.com"},

		// — |http(s):// URL 锚定（纯域名 → exactDomain）—
		{name: "url_http", line: "|http://secure.example.com", wantDomain: "secure.example.com"},
		{name: "url_https_path", line: "|https://www.example.org/page", wantDomain: "www.example.org"},
		{name: "url_http_port", line: "|http://example.com:8080/", wantDomain: "example.com"},

		// — |http(s):// URL 锚定（含通配符 → regexp）—
		{name: "url_http_wildcard", line: "|http://*.bahamut.com.tw", match: "forum.bahamut.com.tw"},

		// — plain domain（纯域名 → exactDomain）—
		{name: "plain_domain", line: "www.aolnews.com", wantDomain: "www.aolnews.com"},

		// — plain keyword（不含 . → regexp）—
		{name: "keyword", line: "blogspot", match: "blogspot.com"},
		{name: "keyword_mid", line: "blogspot", match: "www.blogspot.co.uk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			re, domain, err := autoproxyLineToRegexp(tt.line)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if re != nil || domain != "" {
					t.Fatalf("expected skip, got re=%v domain=%q", re, domain)
				}
				return
			}
			if tt.wantDomain != "" {
				if re != nil {
					t.Fatalf("expected re==nil for pure domain, got %v", re)
				}
				if domain != tt.wantDomain {
					t.Fatalf("domain: want %q got %q", tt.wantDomain, domain)
				}
				return
			}
			// 期望 regexp
			if re == nil {
				t.Fatal("expected non-nil regexp")
			}
			if domain != "" {
				t.Fatalf("expected empty domain for regexp rule, got %q", domain)
			}
			if tt.match != "" && !re.MatchString(tt.match) {
				t.Errorf("regexp %q should match %q", re, tt.match)
			}
			if tt.noMatch != "" && re.MatchString(tt.noMatch) {
				t.Errorf("regexp %q should NOT match %q", re, tt.noMatch)
			}
		})
	}
}

func TestPickForwardRuleWithDomainMap(t *testing.T) {
	t.Parallel()

	domainMap := map[string]compiledRule{
		"example.com":    {ruleID: 1, targetID: 10, mode: "parallel", priority: 100},
		"google.com":     {ruleID: 2, targetID: 20, mode: "sequential", priority: 50},
		"sub.google.com": {ruleID: 3, targetID: 20, mode: "parallel", priority: 40},
	}

	wildcard := regexp.MustCompile(`(?i)(^|\.)cdn.*\.example\.com$`)
	regexRules := []compiledRule{
		{re: wildcard, ruleID: 4, targetID: 40, mode: "parallel", priority: 100},
	}

	tests := []struct {
		name   string
		qname  string
		wantID int32 // 0 = expect nil
	}{
		{name: "exact_domain_hit", qname: "example.com.", wantID: 1},
		{name: "subdomain_suffix", qname: "sub.example.com.", wantID: 1},
		{name: "deep_subdomain", qname: "a.b.example.com.", wantID: 1},
		{name: "higher_priority_domain", qname: "google.com.", wantID: 2},
		{name: "sub_exact_match", qname: "sub.google.com.", wantID: 3},
		{name: "regex_hit_no_domain", qname: "cdn-images.other.com.", wantID: 0},
		{name: "no_match", qname: "unknown.org.", wantID: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := pickForwardRule(tt.qname, regexRules, domainMap)
			if tt.wantID == 0 {
				if got != nil {
					t.Fatalf("expected nil, got ruleID=%d", got.ruleID)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected ruleID=%d, got nil", tt.wantID)
			}
			if got.ruleID != tt.wantID {
				t.Fatalf("ruleID: want %d got %d", tt.wantID, got.ruleID)
			}
		})
	}
}

func TestPickForwardRulePriorityDomainVsRegex(t *testing.T) {
	t.Parallel()

	// 域名规则优先级低于正则规则
	domainMap := map[string]compiledRule{
		"example.com": {ruleID: 1, targetID: 10, mode: "parallel", priority: 200},
	}
	re := regexp.MustCompile(`(?i)(^|\.)example\.com$`)
	regexRules := []compiledRule{
		{re: re, ruleID: 2, targetID: 20, mode: "parallel", priority: 50},
	}
	got := pickForwardRule("example.com.", regexRules, domainMap)
	if got == nil || got.ruleID != 2 {
		t.Fatalf("expected regex rule (ruleID=2) to win, got %+v", got)
	}

	// 域名规则优先级高于正则规则
	domainMap2 := map[string]compiledRule{
		"example.com": {ruleID: 1, targetID: 10, mode: "parallel", priority: 10},
	}
	got2 := pickForwardRule("example.com.", regexRules, domainMap2)
	if got2 == nil || got2.ruleID != 1 {
		t.Fatalf("expected domain rule (ruleID=1) to win, got %+v", got2)
	}
}
