package notify

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dnstube/dnstube/internal/store"
	"github.com/dnstube/dnstube/internal/telegram"
)

// LevelAudit 表示审计类通知（与系统 slog 级别并列，由 telegram_notify_levels 勾选）。
const LevelAudit = "audit"

var allowedLevels = map[string]bool{
	"debug": true, "info": true, "warn": true, "error": true, LevelAudit: true,
}

// NormalizeNotifyLevels 去重、小写、过滤非法项。
func NormalizeNotifyLevels(in []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, s := range in {
		k := strings.ToLower(strings.TrimSpace(s))
		if allowedLevels[k] && !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	return out
}

// Notifier 按系统设置推送 Telegram（异步，不阻塞调用方）。
type Notifier struct {
	Store *store.Store
}

func New(st *store.Store) *Notifier {
	return &Notifier{Store: st}
}

// MaybeSystem 系统运行时日志（与 slog 写入 DB 的级别一致）。
func (n *Notifier) MaybeSystem(level, message string) {
	if n == nil || n.Store == nil {
		return
	}
	lv := strings.ToLower(strings.TrimSpace(level))
	if !allowedLevels[lv] || lv == LevelAudit {
		return
	}
	go n.push(context.Background(), lv, fmt.Sprintf("[%s] %s", strings.ToUpper(lv), message))
}

// MaybeAudit 审计事件；仅当配置包含 audit 时推送。
func (n *Notifier) MaybeAudit(event, level, message string, username *string) {
	if n == nil || n.Store == nil {
		return
	}
	var b strings.Builder
	b.WriteString("DnsTube 审计\n")
	if username != nil && strings.TrimSpace(*username) != "" {
		fmt.Fprintf(&b, "用户: %s\n", strings.TrimSpace(*username))
	}
	fmt.Fprintf(&b, "事件: %s\n级别: %s\n%s", event, level, message)
	go n.push(context.Background(), LevelAudit, b.String())
}

func (n *Notifier) push(bg context.Context, levelKey, text string) {
	ctx, cancel := context.WithTimeout(bg, 10*time.Second)
	defer cancel()
	set, err := n.Store.GetSystemSettings(ctx)
	if err != nil || !set.TelegramEnabled {
		return
	}
	if set.TelegramBotToken == nil || set.TelegramChatID == nil {
		return
	}
	if !levelEnabled(set.TelegramNotifyLevels, levelKey) {
		return
	}
	token := *set.TelegramBotToken
	chat := *set.TelegramChatID
	if token == "" || chat == "" {
		return
	}
	_ = telegram.SendMessage(ctx, token, chat, text)
}

func levelEnabled(levels []string, key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, x := range levels {
		if strings.ToLower(strings.TrimSpace(x)) == key {
			return true
		}
	}
	return false
}

// SendTest 发送测试消息（供管理 API 调用）。
func (n *Notifier) SendTest(ctx context.Context) error {
	if n == nil || n.Store == nil {
		return fmt.Errorf("notifier not configured")
	}
	set, err := n.Store.GetSystemSettings(ctx)
	if err != nil {
		return err
	}
	if !set.TelegramEnabled {
		return fmt.Errorf("telegram not enabled")
	}
	if set.TelegramBotToken == nil || set.TelegramChatID == nil {
		return fmt.Errorf("missing bot token or chat id")
	}
	token := *set.TelegramBotToken
	chat := *set.TelegramChatID
	if token == "" || chat == "" {
		return fmt.Errorf("missing bot token or chat id")
	}
	return telegram.SendMessage(ctx, token, chat, "DnsTube：测试通知（Telegram 配置有效）")
}
