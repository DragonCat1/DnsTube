package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dnstube/dnstube/internal/store"
)

func (s *Server) listQueryLogs(w http.ResponseWriter, r *http.Request) {
	lq := ParseListQuery(r, 50)
	q := r.URL.Query()
	var f store.QueryLogFilter
	if v := q.Get("instance_id"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err == nil {
			x := int32(n)
			f.InstanceID = &x
		}
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
	if v := strings.TrimSpace(q.Get("client_ip")); v != "" {
		f.ClientIP = &v
	}
	if v := strings.TrimSpace(q.Get("qname")); v != "" {
		f.Qname = &v
	}
	if v := strings.TrimSpace(q.Get("qtype")); v != "" {
		f.Qtype = &v
	}
	if v := strings.TrimSpace(q.Get("response_code")); v != "" {
		f.Rcode = &v
	}
	if v := q.Get("forward_upstream_group_id"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err == nil {
			x := int32(n)
			f.ForwardUpstreamGroupID = &x
		}
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
	list, total, err := s.Store.ListQueryLogs(r.Context(), f)
	if err != nil {
		writeListErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	writeListOK(w, http.StatusOK, list, lq.Page, total)
}

type deleteLogsReq struct {
	IDs []int64 `json:"ids"`
}

func (s *Server) deleteQueryLogs(w http.ResponseWriter, r *http.Request) {
	var req deleteLogsReq
	if err := readJSON(r, &req); err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeInvalidJSON)
		return
	}
	n, err := s.Store.DeleteQueryLogs(r.Context(), req.IDs)
	if err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	writeObjOK(w, http.StatusOK, map[string]int64{"deleted": n})
}
