package api

import (
	"net/http"

	"github.com/dnstube/dnstube/internal/notify"
	"github.com/dnstube/dnstube/internal/store"
)

func (s *Server) getSystemSettings(w http.ResponseWriter, r *http.Request) {
	cur, err := s.Store.GetSystemSettings(r.Context())
	if err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	tokenSet := cur.TelegramBotToken != nil && *cur.TelegramBotToken != ""
	chatID := ""
	if cur.TelegramChatID != nil {
		chatID = *cur.TelegramChatID
	}
	levels := cur.TelegramNotifyLevels
	if levels == nil {
		levels = []string{}
	}
	writeObjOK(w, http.StatusOK, map[string]any{
		"telegram_enabled":         cur.TelegramEnabled,
		"telegram_chat_id":         chatID,
		"telegram_bot_token_set":   tokenSet,
		"telegram_notify_levels":   levels,
	})
}

type patchSystemSettingsReq struct {
	TelegramEnabled       *bool     `json:"telegram_enabled"`
	TelegramBotToken      *string   `json:"telegram_bot_token"`
	TelegramChatID        *string   `json:"telegram_chat_id"`
	TelegramNotifyLevels  *[]string `json:"telegram_notify_levels"`
}

func (s *Server) patchSystemSettings(w http.ResponseWriter, r *http.Request) {
	var req patchSystemSettingsReq
	if err := readJSON(r, &req); err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeInvalidJSON)
		return
	}
	patch := store.SystemSettingsPatch{
		TelegramEnabled:  req.TelegramEnabled,
		TelegramBotToken: req.TelegramBotToken,
		TelegramChatID:   req.TelegramChatID,
	}
	if req.TelegramNotifyLevels != nil {
		n := notify.NormalizeNotifyLevels(*req.TelegramNotifyLevels)
		patch.TelegramNotifyLevels = &n
	}
	if err := s.Store.PatchSystemSettings(r.Context(), patch); err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	uname, _ := UsernameFromContext(r.Context())
	s.insertAuditLog(r, "info", "system_settings_update", uname, "系统设置（通知）已更新", nil)
	writeObjOK(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) postTelegramTest(w http.ResponseWriter, r *http.Request) {
	if s.Notify == nil {
		writeObjErr(w, http.StatusInternalServerError, CodeDatabaseError)
		return
	}
	if err := s.Notify.SendTest(r.Context()); err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeTelegramSendFailed)
		return
	}
	writeObjOK(w, http.StatusOK, map[string]string{"status": "ok"})
}
