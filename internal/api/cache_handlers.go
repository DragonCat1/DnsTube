package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/dnstube/dnstube/internal/dns"
)

func (s *Server) listCacheEntries(w http.ResponseWriter, r *http.Request) {
	lq := ParseListQuery(r, 0)
	if s.Engine == nil {
		writeListOK(w, http.StatusOK, []dns.CacheListItem{}, lq.Page, 0)
		return
	}
	items := s.Engine.ListCacheEntries()
	q := r.URL.Query()
	var inst *int32
	if v := q.Get("instance_id"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err == nil {
			x := int32(n)
			inst = &x
		}
	}
	qnameSub := strings.TrimSpace(q.Get("qname"))
	qtypeEq := strings.TrimSpace(q.Get("qtype"))
	var filtered []dns.CacheListItem
	for _, it := range items {
		if inst != nil && it.InstanceID != *inst {
			continue
		}
		if qnameSub != "" && !strings.Contains(strings.ToLower(it.Qname), strings.ToLower(qnameSub)) {
			continue
		}
		if qtypeEq != "" && !strings.EqualFold(it.Qtype, qtypeEq) {
			continue
		}
		filtered = append(filtered, it)
	}
	sortBy := lq.SortBy
	if sortBy == "" {
		sortBy = "expires_at"
	}
	order := lq.SortOrder
	if order == "" {
		order = "asc"
	}
	dns.SortCacheList(filtered, sortBy, order)
	slice, page, total := PaginateSlice(filtered, lq.Limit, lq.Page)
	writeListOK(w, http.StatusOK, slice, page, total)
}
