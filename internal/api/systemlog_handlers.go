package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/dnstube/dnstube/internal/store"
)

func (s *Server) listSystemLogs(w http.ResponseWriter, r *http.Request) {
	lq := ParseListQuery(r, 50)
	q := r.URL.Query()
	var f store.SystemLogFilter
	if v := strings.TrimSpace(q.Get("kind")); v != "" {
		f.Kind = &v
	}
	if v := strings.TrimSpace(q.Get("level")); v != "" {
		f.Level = &v
	}
	if v := strings.TrimSpace(q.Get("event")); v != "" {
		f.Event = &v
	}
	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			f.From = &t
		}
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			f.To = &t
		}
	}
	if v := strings.TrimSpace(q.Get("message")); v != "" {
		f.Message = &v
	}
	if v := strings.TrimSpace(q.Get("username")); v != "" {
		f.Username = &v
	}
	if v := strings.TrimSpace(q.Get("client_ip")); v != "" {
		f.ClientIP = &v
	}
	f.Limit = lq.Limit
	if f.Limit > 0 {
		f.Offset = (lq.Page - 1) * f.Limit
	}
	f.SortBy = lq.SortBy
	if f.SortBy == "" {
		f.SortBy = "created_at"
	}
	f.SortOrder = lq.SortOrder
	if f.SortOrder == "" {
		f.SortOrder = "desc"
	}
	list, total, err := s.Store.ListSystemLogs(r.Context(), f)
	if err != nil {
		writeListErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	writeListOK(w, http.StatusOK, list, lq.Page, total)
}

type deleteSystemLogsReq struct {
	IDs []int64 `json:"ids"`
}

func (s *Server) deleteSystemLogs(w http.ResponseWriter, r *http.Request) {
	var req deleteSystemLogsReq
	if err := readJSON(r, &req); err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeInvalidJSON)
		return
	}
	n, err := s.Store.DeleteSystemLogs(r.Context(), req.IDs)
	if err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	writeObjOK(w, http.StatusOK, map[string]int64{"deleted": n})
}
