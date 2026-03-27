package api

import (
	"net/http"
	"strconv"

	"github.com/dnstube/dnstube/internal/store"
	"github.com/go-chi/chi/v5"
)

// defaultUpstreamInfo 默认转发组展示（名称 + 服务器列表）。
type defaultUpstreamInfo struct {
	ID      int32                   `json:"id"`
	Name    string                  `json:"name"`
	Servers []store.UpstreamServer  `json:"servers"`
}

// instanceListItem 在 DNS 实例上附加运行时错误信息（非 DB 字段，如 UDP 绑定失败等）。
type instanceListItem struct {
	store.DNSInstance
	RuntimeError    *string              `json:"runtime_error,omitempty"`
	DefaultUpstream *defaultUpstreamInfo `json:"default_upstream,omitempty"`
	RecordCount     int64                `json:"record_count"`
}

func (s *Server) listInstances(w http.ResponseWriter, r *http.Request) {
	lq := ParseListQuery(r, 0)
	list, err := s.Store.ListInstances(r.Context())
	if err != nil {
		writeListErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	groups, err := s.Store.ListUpstreamGroups(r.Context())
	if err != nil {
		writeListErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	nameByID := make(map[int32]string, len(groups))
	for _, g := range groups {
		nameByID[g.ID] = g.Name
	}
	counts, err := s.Store.ListRecordCountsPerInstance(r.Context())
	if err != nil {
		writeListErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	var errs map[int32]string
	if s.Engine != nil {
		errs = s.Engine.ListenerErrors()
	}
	out := make([]instanceListItem, 0, len(list))
	for _, inst := range list {
		item := instanceListItem{DNSInstance: inst, RecordCount: counts[inst.ID]}
		if errs != nil {
			if msg, ok := errs[inst.ID]; ok && msg != "" {
				item.RuntimeError = &msg
			}
		}
		if inst.DefaultUpstreamGroupID != nil {
			gid := *inst.DefaultUpstreamGroupID
			srv, _ := s.Store.ListUpstreamServers(r.Context(), gid)
			item.DefaultUpstream = &defaultUpstreamInfo{
				ID:      gid,
				Name:    nameByID[gid],
				Servers: srv,
			}
		}
		out = append(out, item)
	}
	if lq.SortOrder == "" {
		lq.SortOrder = "asc"
	}
	sortInstanceListItems(out, lq.SortBy, lq.SortOrder)
	slice, page, total := PaginateSlice(out, lq.Limit, lq.Page)
	writeListOK(w, http.StatusOK, slice, page, total)
}

type createInstReq struct {
	Name                   string `json:"name"`
	ListenAddr             string `json:"listen_addr"`
	ListenPort             int    `json:"listen_port"`
	DefaultUpstreamGroupID *int32 `json:"default_upstream_group_id"`
}

func (s *Server) createInstance(w http.ResponseWriter, r *http.Request) {
	var req createInstReq
	if err := readJSON(r, &req); err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeInvalidJSON)
		return
	}
	if req.Name == "" || req.ListenPort <= 0 {
		writeObjErr(w, http.StatusBadRequest, CodeInstanceFieldsRequired)
		return
	}
	if req.ListenAddr == "" {
		req.ListenAddr = "0.0.0.0"
	}
	id, err := s.Store.CreateInstance(r.Context(), req.Name, req.ListenAddr, req.ListenPort, req.DefaultUpstreamGroupID)
	if err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	if !s.sync(w, r) {
		return
	}
	writeObjOK(w, http.StatusCreated, map[string]int32{"id": id})
}

func (s *Server) getInstance(w http.ResponseWriter, r *http.Request) {
	id, err := parseID32(chi.URLParam(r, "id"))
	if err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeBadID)
		return
	}
	inst, err := s.Store.GetInstance(r.Context(), id)
	if err != nil {
		writeObjErr(w, http.StatusNotFound, CodeNotFound)
		return
	}
	rc, err := s.Store.CountRecordsByInstance(r.Context(), id)
	if err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	item := instanceListItem{DNSInstance: *inst, RecordCount: rc}
	if s.Engine != nil {
		if errs := s.Engine.ListenerErrors(); errs != nil {
			if msg, ok := errs[id]; ok && msg != "" {
				item.RuntimeError = &msg
			}
		}
	}
	if inst.DefaultUpstreamGroupID != nil {
		gid := *inst.DefaultUpstreamGroupID
		groups, err := s.Store.ListUpstreamGroups(r.Context())
		if err == nil {
			var name string
			for _, g := range groups {
				if g.ID == gid {
					name = g.Name
					break
				}
			}
			srv, _ := s.Store.ListUpstreamServers(r.Context(), gid)
			item.DefaultUpstream = &defaultUpstreamInfo{ID: gid, Name: name, Servers: srv}
		}
	}
	writeObjOK(w, http.StatusOK, item)
}

type patchInstReq struct {
	Name                   *string `json:"name"`
	ListenAddr             *string `json:"listen_addr"`
	ListenPort             *int    `json:"listen_port"`
	Paused                 *bool   `json:"paused"`
	DefaultUpstreamGroupID *int32  `json:"default_upstream_group_id"`
}

func (s *Server) patchInstance(w http.ResponseWriter, r *http.Request) {
	id, err := parseID32(chi.URLParam(r, "id"))
	if err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeBadID)
		return
	}
	var req patchInstReq
	if err := readJSON(r, &req); err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeInvalidJSON)
		return
	}
	if err := s.Store.UpdateInstance(r.Context(), id, req.Name, req.ListenAddr, req.ListenPort, req.Paused, req.DefaultUpstreamGroupID); err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	if !s.sync(w, r) {
		return
	}
	writeObjOK(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) deleteInstance(w http.ResponseWriter, r *http.Request) {
	id, err := parseID32(chi.URLParam(r, "id"))
	if err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeBadID)
		return
	}
	if err := s.Store.DeleteInstance(r.Context(), id); err != nil {
		writeObjErr(w, http.StatusNotFound, CodeNotFound)
		return
	}
	if !s.sync(w, r) {
		return
	}
	writeObjOK(w, http.StatusOK, nil)
}

func parseID32(s string) (int32, error) {
	n, err := strconv.ParseInt(s, 10, 32)
	return int32(n), err
}

func (s *Server) listRecords(w http.ResponseWriter, r *http.Request) {
	lq := ParseListQuery(r, 0)
	iid, err := parseID32(chi.URLParam(r, "id"))
	if err != nil {
		writeListErr(w, http.StatusBadRequest, CodeBadID)
		return
	}
	recs, err := s.Store.ListRecordsByInstance(r.Context(), iid)
	if err != nil {
		writeListErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	if lq.SortOrder == "" {
		lq.SortOrder = "asc"
	}
	sortDNSRecordsSlice(recs, lq.SortBy, lq.SortOrder)
	slice, page, total := PaginateSlice(recs, lq.Limit, lq.Page)
	writeListOK(w, http.StatusOK, slice, page, total)
}

type recordReq struct {
	Name    string `json:"name"`
	Rtype   string `json:"rtype"`
	TTL     int    `json:"ttl"`
	Content string `json:"content"`
}

func (s *Server) createRecord(w http.ResponseWriter, r *http.Request) {
	iid, err := parseID32(chi.URLParam(r, "id"))
	if err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeBadID)
		return
	}
	var req recordReq
	if err := readJSON(r, &req); err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeInvalidJSON)
		return
	}
	if code := recordRDataErrCode(req.Rtype, req.Content); code != "" {
		writeObjErr(w, http.StatusBadRequest, code)
		return
	}
	id, err := s.Store.CreateRecord(r.Context(), iid, req.Name, req.Rtype, req.TTL, req.Content)
	if err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	if !s.sync(w, r) {
		return
	}
	writeObjOK(w, http.StatusCreated, map[string]int32{"id": id})
}

func (s *Server) patchRecord(w http.ResponseWriter, r *http.Request) {
	id, err := parseID32(chi.URLParam(r, "id"))
	if err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeBadID)
		return
	}
	var req recordReq
	if err := readJSON(r, &req); err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeInvalidJSON)
		return
	}
	if code := recordRDataErrCode(req.Rtype, req.Content); code != "" {
		writeObjErr(w, http.StatusBadRequest, code)
		return
	}
	if err := s.Store.UpdateRecord(r.Context(), id, req.Name, req.Rtype, req.TTL, req.Content); err != nil {
		if err == store.ErrNotFound {
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

func (s *Server) deleteRecord(w http.ResponseWriter, r *http.Request) {
	id, err := parseID32(chi.URLParam(r, "id"))
	if err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeBadID)
		return
	}
	if err := s.Store.DeleteRecord(r.Context(), id); err != nil {
		writeObjErr(w, http.StatusNotFound, CodeNotFound)
		return
	}
	if !s.sync(w, r) {
		return
	}
	writeObjOK(w, http.StatusOK, nil)
}
