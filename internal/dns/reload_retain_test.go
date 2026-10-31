package dns

import (
	"testing"
)

func TestKeepCompileFromSnap(t *testing.T) {
	snap := &instanceSnap{
		rules: []compiledRule{{ruleID: 7}, {ruleID: 8}},
		domainMap: map[string]compiledRule{
			"a.com": {ruleID: 9},
		},
	}
	keep := map[int32]struct{}{}
	keepCompileFromSnap(snap, keep)
	for _, id := range []int32{7, 8, 9} {
		if _, ok := keep[id]; !ok {
			t.Fatalf("missing rule id %d", id)
		}
	}
}
