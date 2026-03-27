package store

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5"
)

// SystemSettings 单行系统配置（id 固定为 1）。
type SystemSettings struct {
	TelegramEnabled      bool
	TelegramBotToken     *string
	TelegramChatID       *string
	TelegramNotifyLevels []string
}

// SystemSettingsPatch PATCH 时仅非 nil 字段生效；BotToken 为空字符串表示清空。
type SystemSettingsPatch struct {
	TelegramEnabled       *bool
	TelegramBotToken      *string
	TelegramChatID        *string
	TelegramNotifyLevels  *[]string
}

func (s *Store) GetSystemSettings(ctx context.Context) (SystemSettings, error) {
	var out SystemSettings
	var tok, chat sql.NullString
	var levels []string
	err := s.Pool.QueryRow(ctx, `
		SELECT telegram_enabled, telegram_bot_token, telegram_chat_id, telegram_notify_levels
		FROM system_settings WHERE id = 1`,
	).Scan(&out.TelegramEnabled, &tok, &chat, &levels)
	if err != nil {
		if err == pgx.ErrNoRows {
			return SystemSettings{}, nil
		}
		return SystemSettings{}, err
	}
	if tok.Valid && tok.String != "" {
		t := tok.String
		out.TelegramBotToken = &t
	}
	if chat.Valid && chat.String != "" {
		c := chat.String
		out.TelegramChatID = &c
	}
	if levels == nil {
		out.TelegramNotifyLevels = []string{}
	} else {
		out.TelegramNotifyLevels = levels
	}
	return out, nil
}

func (s *Store) PatchSystemSettings(ctx context.Context, patch SystemSettingsPatch) error {
	cur, err := s.GetSystemSettings(ctx)
	if err != nil {
		return err
	}
	if patch.TelegramEnabled != nil {
		cur.TelegramEnabled = *patch.TelegramEnabled
	}
	if patch.TelegramBotToken != nil {
		if *patch.TelegramBotToken == "" {
			cur.TelegramBotToken = nil
		} else {
			t := *patch.TelegramBotToken
			cur.TelegramBotToken = &t
		}
	}
	if patch.TelegramChatID != nil {
		if *patch.TelegramChatID == "" {
			cur.TelegramChatID = nil
		} else {
			c := *patch.TelegramChatID
			cur.TelegramChatID = &c
		}
	}
	if patch.TelegramNotifyLevels != nil {
		cur.TelegramNotifyLevels = *patch.TelegramNotifyLevels
	}
	var tok any
	if cur.TelegramBotToken != nil {
		tok = *cur.TelegramBotToken
	}
	var chat any
	if cur.TelegramChatID != nil {
		chat = *cur.TelegramChatID
	}
	_, err = s.Pool.Exec(ctx, `
		UPDATE system_settings SET
			telegram_enabled = $1,
			telegram_bot_token = $2,
			telegram_chat_id = $3,
			telegram_notify_levels = $4
		WHERE id = 1`,
		cur.TelegramEnabled, tok, chat, cur.TelegramNotifyLevels,
	)
	return err
}
