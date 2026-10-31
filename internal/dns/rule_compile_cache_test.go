package dns

import (
	"testing"
	"time"

	"github.com/dnstube/dnstube/internal/store"
)

func TestRuleCompileFingerprintIgnoresFetchedAt(t *testing.T) {
	body := "||example.com^\n"
	url := "https://example.com/list.txt"
	at1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	at2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	r1 := store.ForwardRule{
		ID:                 1,
		NamePattern:        "",
		PatternSource:      "url",
		PatternURL:         &url,
		PatternFormat:      "autoproxy",
		PatternFetchedBody: &body,
		TargetGroupID:      2,
		Mode:               "parallel",
		Priority:           100,
		PatternFetchedAt:   &at1,
	}
	r2 := r1
	r2.PatternFetchedAt = &at2
	if ruleCompileFingerprint(r1) != ruleCompileFingerprint(r2) {
		t.Fatal("fingerprint must ignore pattern_fetched_at")
	}
}

func TestRuleCompileFingerprintChangesOnBody(t *testing.T) {
	b1 := "||a.com^\n"
	b2 := "||b.com^\n"
	url := "https://example.com/list.txt"
	r1 := store.ForwardRule{
		PatternSource:      "url",
		PatternURL:         &url,
		PatternFormat:      "autoproxy",
		PatternFetchedBody: &b1,
		TargetGroupID:      1,
		Mode:               "sequential",
		Priority:           10,
	}
	r2 := r1
	r2.PatternFetchedBody = &b2
	if ruleCompileFingerprint(r1) == ruleCompileFingerprint(r2) {
		t.Fatal("fingerprint must change when body changes")
	}
}

func TestPatternFetchContentChanged(t *testing.T) {
	old := "old"
	before := store.ForwardRule{PatternFetchedBody: &old, PatternResolvedEntries: []byte(`["a"]`)}
	if !PatternFetchContentChanged(before, "new", []byte(`["a"]`)) {
		t.Fatal("body change should be detected")
	}
	if PatternFetchContentChanged(before, "old", []byte(`["a"]`)) {
		t.Fatal("identical content should be unchanged")
	}
	if !PatternFetchContentChanged(before, "old", []byte(`["b"]`)) {
		t.Fatal("resolved entries change should be detected")
	}
}
