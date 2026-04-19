package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/dnstube/dnstube/internal/store"
	"github.com/go-chi/chi/v5"
)

func (s *Server) listGroupServers(w http.ResponseWriter, r *http.Request) {
	lq := ParseListQuery(r, 0)
	id, err := parseID32(chi.URLParam(r, "id"))
	if err != nil {
		writeListErr(w, http.StatusBadRequest, CodeBadID)
		return
	}
	list, err := s.Store.ListUpstreamServers(r.Context(), id)
	if err != nil {
		writeListErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	if lq.SortOrder == "" {
		lq.SortOrder = "asc"
	}
	sortUpstreamServersSlice(list, lq.SortBy, lq.SortOrder)
	slice, page, total := PaginateSlice(list, lq.Limit, lq.Page)
	writeListOK(w, http.StatusOK, slice, page, total)
}

func (s *Server) listGroups(w http.ResponseWriter, r *http.Request) {
	lq := ParseListQuery(r, 0)
	list, err := s.Store.ListUpstreamGroups(r.Context())
	if err != nil {
		writeListErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	if lq.SortOrder == "" {
		lq.SortOrder = "asc"
	}
	sortUpstreamGroupsSlice(list, lq.SortBy, lq.SortOrder)
	slice, page, total := PaginateSlice(list, lq.Limit, lq.Page)
	writeListOK(w, http.StatusOK, slice, page, total)
}

type groupReq struct {
	Name string `json:"name"`
}

func (s *Server) createGroup(w http.ResponseWriter, r *http.Request) {
	var req groupReq
	if err := readJSON(r, &req); err != nil || req.Name == "" {
		if err != nil {
			writeObjErr(w, http.StatusBadRequest, CodeInvalidJSON)
			return
		}
		writeObjErr(w, http.StatusBadRequest, CodeGroupNameRequired)
		return
	}
	id, err := s.Store.CreateUpstreamGroup(r.Context(), req.Name)
	if err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	if !s.sync(w, r) {
		return
	}
	writeObjOK(w, http.StatusCreated, map[string]int32{"id": id})
}

func (s *Server) patchGroup(w http.ResponseWriter, r *http.Request) {
	id, err := parseID32(chi.URLParam(r, "id"))
	if err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeBadID)
		return
	}
	var req groupReq
	if err := readJSON(r, &req); err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeInvalidJSON)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeObjErr(w, http.StatusBadRequest, CodeGroupNameRequired)
		return
	}
	if err := s.Store.UpdateUpstreamGroup(r.Context(), id, name); err != nil {
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

func (s *Server) deleteGroup(w http.ResponseWriter, r *http.Request) {
	id, err := parseID32(chi.URLParam(r, "id"))
	if err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeBadID)
		return
	}
	if err := s.Store.DeleteUpstreamGroup(r.Context(), id); err != nil {
		writeObjErr(w, http.StatusNotFound, CodeNotFound)
		return
	}
	if !s.sync(w, r) {
		return
	}
	writeObjOK(w, http.StatusOK, nil)
}

type serverReq struct {
	Address       string  `json:"address"`
	Port          int     `json:"port"`
	SortOrder     int     `json:"sort_order"`
	Protocol      string  `json:"protocol"`
	Path          *string `json:"path,omitempty"`
	TLSServerName *string `json:"tls_server_name,omitempty"`
}

func (s *Server) addServer(w http.ResponseWriter, r *http.Request) {
	gid, err := parseID32(chi.URLParam(r, "id"))
	if err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeBadID)
		return
	}
	var req serverReq
	if err := readJSON(r, &req); err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeInvalidJSON)
		return
	}
	if req.Address == "" {
		writeObjErr(w, http.StatusBadRequest, CodeServerAddressRequired)
		return
	}
	proto, ok := NormalizeUpstreamProtocol(req.Protocol)
	if !ok {
		writeObjErr(w, http.StatusBadRequest, CodeUpstreamProtocolInvalid)
		return
	}
	addr, ok := NormalizeUpstreamAddr(req.Address)
	if !ok {
		writeObjErr(w, http.StatusBadRequest, CodeUpstreamServerIPInvalid)
		return
	}
	if req.Port <= 0 {
		req.Port = DefaultPortForProtocol(proto)
	}

	in := store.AddUpstreamServerInput{
		GroupID:   gid,
		Address:   addr,
		Port:      req.Port,
		SortOrder: req.SortOrder,
		Protocol:  proto,
	}

	// DoH：必须提供合法 path（缺省 /dns-query）。
	if proto == UpstreamProtocolDoH {
		raw := ""
		if req.Path != nil {
			raw = *req.Path
		}
		path, ok := NormalizeDoHPath(raw)
		if !ok {
			writeObjErr(w, http.StatusBadRequest, CodeUpstreamDoHPathInvalid)
			return
		}
		in.Path = &path
	}

	// DoT/DoH 的 SNI 可选；提供时必须合法。
	if proto != UpstreamProtocolUDP && req.TLSServerName != nil {
		sni, ok := NormalizeTLSServerName(*req.TLSServerName)
		if !ok {
			writeObjErr(w, http.StatusBadRequest, CodeUpstreamSNIInvalid)
			return
		}
		if sni != "" {
			in.TLSServerName = &sni
		}
	}

	id, err := s.Store.AddUpstreamServer(r.Context(), in)
	if err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	if !s.sync(w, r) {
		return
	}
	writeObjOK(w, http.StatusCreated, map[string]int32{"id": id})
}

func (s *Server) deleteServer(w http.ResponseWriter, r *http.Request) {
	id, err := parseID32(chi.URLParam(r, "id"))
	if err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeBadID)
		return
	}
	if err := s.Store.DeleteUpstreamServer(r.Context(), id); err != nil {
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
	writeObjOK(w, http.StatusOK, nil)
}
