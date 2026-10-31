package dns

import (
	"bytes"
	"log/slog"

	"github.com/dnstube/dnstube/internal/store"
)

// PatternFetchSource 标识 URL 规则拉取触发来源。
const (
	PatternFetchSourceScheduled = "scheduled"
	PatternFetchSourceManual    = "manual"
	PatternFetchSourceSave      = "save"
)

// PatternFetchContentChanged 比较拉取前后正文与 resolved_entries 是否变化。
func PatternFetchContentChanged(before store.ForwardRule, newBody string, newResolved []byte) bool {
	oldBody := ""
	if before.PatternFetchedBody != nil {
		oldBody = *before.PatternFetchedBody
	}
	if oldBody != newBody {
		return true
	}
	oldResolved := before.PatternResolvedEntries
	if len(oldResolved) == 0 && len(newResolved) == 0 {
		return false
	}
	return !bytes.Equal(oldResolved, newResolved)
}

// LogPatternFetchSuccess 写入系统日志（slog → kind=system）：拉取成功及条数增减。
func LogPatternFetchSuccess(log *slog.Logger, ruleID int32, oldCount, newCount int, contentChanged bool, source string) {
	if log == nil {
		log = slog.Default()
	}
	log.Info("forward rule pattern fetch success",
		"rule_id", ruleID,
		"old_count", oldCount,
		"new_count", newCount,
		"delta", newCount-oldCount,
		"content_changed", contentChanged,
		"source", source,
	)
}
