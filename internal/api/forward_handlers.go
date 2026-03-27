package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	dnspkg "github.com/dnstube/dnstube/internal/dns"
	"github.com/dnstube/dnstube/internal/store"
	"github.com/go-chi/chi/v5"
)

func (s *Server) listRules(w http.ResponseWriter, r *http.Request) {
	lq := ParseListQuery(r, 0)
	iid, err := parseID32(chi.URLParam(r, "id"))
	if err != nil {
		writeListErr(w, http.StatusBadRequest, CodeBadID)
		return
	}
	list, err := s.Store.ListForwardRulesSummary(r.Context(), iid)
	if err != nil {
		writeListErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	if lq.SortOrder == "" {
		lq.SortOrder = "asc"
	}
	sortForwardRulesSlice(list, lq.SortBy, lq.SortOrder)
	slice, page, total := PaginateSlice(list, lq.Limit, lq.Page)
	writeListOK(w, http.StatusOK, slice, page, total)
}

type ruleReq struct {
	NamePattern   string  `json:"name_pattern"`
	PatternSource string  `json:"pattern_source"`
	PatternURL    *string `json:"pattern_url"`
	PatternFormat string  `json:"pattern_format"`
	TargetGroupID int32   `json:"target_group_id"`
	Mode          string  `json:"mode"`
	Priority      int     `json:"priority"`
	Disabled      bool    `json:"disabled"`
}

func normalizeRuleReq(req *ruleReq) store.ForwardRule {
	src := req.PatternSource
	if src == "" {
		src = "inline"
	}
	fmt := req.PatternFormat
	if fmt == "" {
		fmt = "regex_list"
	}
	return store.ForwardRule{
		NamePattern:   strings.TrimSpace(req.NamePattern),
		PatternSource: src,
		PatternURL:    req.PatternURL,
		PatternFormat: fmt,
		TargetGroupID: req.TargetGroupID,
		Mode:          req.Mode,
		Priority:      req.Priority,
		Disabled:      req.Disabled,
	}
}

func validateRuleReq(req *ruleReq) string {
	r := normalizeRuleReq(req)
	if r.TargetGroupID <= 0 {
		return "target_group_id"
	}
	if r.Mode != "sequential" && r.Mode != "parallel" {
		return "mode"
	}
	if r.Priority < 0 {
		return "priority"
	}
	switch r.PatternSource {
	case "inline":
		if r.NamePattern == "" {
			return "name_pattern"
		}
		switch r.PatternFormat {
		case "regex_list", "autoproxy", "autoproxy_base64":
		default:
			return "pattern_format"
		}
	case "url":
		if r.PatternURL == nil || strings.TrimSpace(*r.PatternURL) == "" {
			return "pattern_url"
		}
		switch r.PatternFormat {
		case "regex_list", "autoproxy", "autoproxy_base64":
		default:
			return "pattern_format"
		}
	default:
		return "pattern_source"
	}
	return ""
}

func shouldClearPatternCache(old, new store.ForwardRule) bool {
	if old.PatternSource == "url" && new.PatternSource != "url" {
		return true
	}
	if new.PatternSource != "url" {
		return false
	}
	if old.PatternSource != "url" && new.PatternSource == "url" {
		return true
	}
	oURL := ""
	if old.PatternURL != nil {
		oURL = strings.TrimSpace(*old.PatternURL)
	}
	nURL := ""
	if new.PatternURL != nil {
		nURL = strings.TrimSpace(*new.PatternURL)
	}
	if oURL != nURL {
		return true
	}
	return strings.TrimSpace(old.PatternFormat) != strings.TrimSpace(new.PatternFormat)
}

// syncForwardRulePatternAfterConfigChange 在创建/修改规则后刷新 URL 缓存或内联编译结果。
func (s *Server) syncForwardRulePatternAfterConfigChange(ctx context.Context, id int32) {
	full, err := s.Store.GetForwardRuleByID(ctx, id)
	if err != nil {
		if s.Log != nil {
			s.Log.Warn("get forward rule after write", "id", id, "err", err)
		}
		return
	}
	at := time.Now().UTC()
	if full.PatternSource == "url" {
		if full.PatternURL == nil || strings.TrimSpace(*full.PatternURL) == "" {
			return
		}
		bodyEmpty := full.PatternFetchedBody == nil || strings.TrimSpace(*full.PatternFetchedBody) == ""
		if !bodyEmpty {
			return
		}
		body, count, lines, err := dnspkg.FetchURLPattern(ctx, full)
		if err != nil {
			if s.Log != nil {
				s.Log.Warn("forward rule url fetch after write", "id", id, "err", err)
			}
			return
		}
		enc, mErr := dnspkg.EncodedResolvedEntries(lines)
		if mErr != nil {
			if s.Log != nil {
				s.Log.Warn("marshal pattern entries", "id", id, "err", mErr)
			}
			return
		}
		_ = s.Store.UpdateForwardRulePatternFetch(ctx, id, body, count, at, enc)
		return
	}
	n, enc, err := dnspkg.CompileForwardRuleForSave(full)
	if err != nil {
		if s.Log != nil {
			s.Log.Warn("forward rule inline compile after write", "id", id, "err", err)
		}
		return
	}
	_ = s.Store.UpdateForwardRuleInlinePatternCompile(ctx, id, n, at, enc)
}

func (s *Server) createRule(w http.ResponseWriter, r *http.Request) {
	iid, err := parseID32(chi.URLParam(r, "id"))
	if err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeBadID)
		return
	}
	var req ruleReq
	if err := readJSON(r, &req); err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeInvalidJSON)
		return
	}
	if v := validateRuleReq(&req); v != "" {
		writeObjErr(w, http.StatusBadRequest, CodeRuleFieldsRequired)
		return
	}
	norm := normalizeRuleReq(&req)
	id, err := s.Store.CreateForwardRule(r.Context(), iid, norm)
	if err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	s.syncForwardRulePatternAfterConfigChange(r.Context(), id)
	if !s.sync(w, r) {
		return
	}
	writeObjOK(w, http.StatusCreated, map[string]int32{"id": id})
}

func (s *Server) patchRule(w http.ResponseWriter, r *http.Request) {
	id, err := parseID32(chi.URLParam(r, "id"))
	if err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeBadID)
		return
	}
	var req ruleReq
	if err := readJSON(r, &req); err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeInvalidJSON)
		return
	}
	if v := validateRuleReq(&req); v != "" {
		writeObjErr(w, http.StatusBadRequest, CodeRuleFieldsRequired)
		return
	}
	old, err := s.Store.GetForwardRuleByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeObjErr(w, http.StatusNotFound, CodeNotFound)
			return
		}
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	newR := normalizeRuleReq(&req)
	if err := s.Store.UpdateForwardRule(r.Context(), id, newR); err != nil {
		if err == store.ErrNotFound {
			writeObjErr(w, http.StatusNotFound, CodeNotFound)
			return
		}
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	if shouldClearPatternCache(old, newR) {
		_ = s.Store.ClearForwardRulePatternCache(r.Context(), id)
	}
	s.syncForwardRulePatternAfterConfigChange(r.Context(), id)
	if !s.sync(w, r) {
		return
	}
	writeObjOK(w, http.StatusOK, map[string]string{"status": "ok"})
}

type ruleDisabledReq struct {
	Disabled bool `json:"disabled"`
}

func (s *Server) patchRuleDisabled(w http.ResponseWriter, r *http.Request) {
	id, err := parseID32(chi.URLParam(r, "id"))
	if err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeBadID)
		return
	}
	var req ruleDisabledReq
	if err := readJSON(r, &req); err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeInvalidJSON)
		return
	}
	if err := s.Store.UpdateForwardRuleDisabled(r.Context(), id, req.Disabled); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeObjErr(w, http.StatusNotFound, CodeNotFound)
			return
		}
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	if !s.sync(w, r) {
		return
	}
	writeObjOK(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) deleteRule(w http.ResponseWriter, r *http.Request) {
	id, err := parseID32(chi.URLParam(r, "id"))
	if err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeBadID)
		return
	}
	if err := s.Store.DeleteForwardRule(r.Context(), id); err != nil {
		writeObjErr(w, http.StatusNotFound, CodeNotFound)
		return
	}
	if !s.sync(w, r) {
		return
	}
	writeObjOK(w, http.StatusOK, nil)
}

func (s *Server) refreshRulePattern(w http.ResponseWriter, r *http.Request) {
	id, err := parseID32(chi.URLParam(r, "id"))
	if err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeBadID)
		return
	}
	rule, err := s.Store.GetForwardRuleByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeObjErr(w, http.StatusNotFound, CodeNotFound)
			return
		}
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	if rule.PatternSource != "url" {
		writeObjErr(w, http.StatusBadRequest, CodeBadRequest)
		return
	}
	body, count, lines, err := dnspkg.FetchURLPattern(r.Context(), rule)
	if err != nil {
		if s.Log != nil {
			s.Log.Warn("forward rule manual refresh failed", "id", id, "err", err)
		}
		writeObjErr(w, http.StatusBadGateway, CodePatternURLFetchFailed)
		return
	}
	enc, _ := dnspkg.EncodedResolvedEntries(lines)
	at := time.Now().UTC()
	if err := s.Store.UpdateForwardRulePatternFetch(r.Context(), id, body, count, at, enc); err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	if !s.sync(w, r) {
		return
	}
	writeObjOK(w, http.StatusOK, map[string]any{
		"pattern_rule_count": count,
		"pattern_fetched_at": at,
	})
}

func (s *Server) getForwardRulePatternEntries(w http.ResponseWriter, r *http.Request) {
	id, err := parseID32(chi.URLParam(r, "id"))
	if err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeBadID)
		return
	}
	rule, err := s.Store.GetForwardRuleByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeObjErr(w, http.StatusNotFound, CodeNotFound)
			return
		}
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	entries, err := dnspkg.PatternEntryLinesForAdmin(rule)
	if err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeBadRequest)
		return
	}
	writeObjOK(w, http.StatusOK, map[string]any{"entries": entries})
}
