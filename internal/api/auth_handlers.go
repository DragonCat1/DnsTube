package api

import (
	"net/http"
	"time"

	"github.com/dnstube/dnstube/internal/auth"
)

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResp struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
	Username  string `json:"username"`
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := readJSON(r, &req); err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeInvalidJSON)
		return
	}
	u, err := s.Store.GetAdminByUsername(r.Context(), req.Username)
	if err != nil {
		s.insertAuditLog(r, "warn", "login_failed", req.Username, "登录失败", nil)
		writeObjErr(w, http.StatusUnauthorized, CodeInvalidCredentials)
		return
	}
	if !auth.CheckPassword(u.PasswordHash, req.Password) {
		s.insertAuditLog(r, "warn", "login_failed", req.Username, "登录失败", nil)
		writeObjErr(w, http.StatusUnauthorized, CodeInvalidCredentials)
		return
	}
	jwtTTL := 7 * 24 * time.Hour
	tok, err := auth.SignJWT(s.Secret, u.ID, u.Username, jwtTTL)
	if err != nil {
		writeObjErr(w, http.StatusInternalServerError, CodeTokenSignFailed)
		return
	}
	s.insertAuditLog(r, "info", "login_success", u.Username, "登录成功", nil)
	writeObjOK(w, http.StatusOK, loginResp{Token: tok, ExpiresIn: int(jwtTTL / time.Second), Username: u.Username})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	id, ok := UserIDFromContext(r.Context())
	if !ok {
		writeObjErr(w, http.StatusUnauthorized, CodeNoUserContext)
		return
	}
	u, err := s.Store.GetAdminByID(r.Context(), id)
	if err != nil {
		writeObjErr(w, http.StatusNotFound, CodeNotFound)
		return
	}
	writeObjOK(w, http.StatusOK, map[string]any{"id": u.ID, "username": u.Username})
}

type profileReq struct {
	Username *string `json:"username"`
	Password *string `json:"password"`
}

func (s *Server) patchProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := UserIDFromContext(r.Context())
	if !ok {
		writeObjErr(w, http.StatusUnauthorized, CodeNoUserContext)
		return
	}
	cur, err := s.Store.GetAdminByID(r.Context(), id)
	if err != nil {
		writeObjErr(w, http.StatusNotFound, CodeNotFound)
		return
	}
	oldName := cur.Username
	var req profileReq
	if err := readJSON(r, &req); err != nil {
		writeObjErr(w, http.StatusBadRequest, CodeInvalidJSON)
		return
	}
	if req.Username != nil && *req.Username != "" {
		if err := s.Store.UpdateAdminUsername(r.Context(), id, *req.Username); err != nil {
			writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
			return
		}
		if *req.Username != oldName {
			s.insertAuditLog(r, "info", "username_change", *req.Username, "用户名已修改", map[string]any{
				"from": oldName,
				"to":   *req.Username,
			})
		}
	}
	if req.Password != nil && *req.Password != "" {
		h, err := auth.HashPassword(*req.Password)
		if err != nil {
			writeObjErr(w, http.StatusInternalServerError, CodePasswordHashFailed)
			return
		}
		if err := s.Store.UpdateAdminPassword(r.Context(), id, h); err != nil {
			writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
			return
		}
		name := oldName
		if req.Username != nil && *req.Username != "" {
			name = *req.Username
		}
		s.insertAuditLog(r, "info", "password_change", name, "密码已修改", nil)
	}
	if !s.sync(w, r) {
		return
	}
	writeObjOK(w, http.StatusOK, map[string]string{"status": "ok"})
}
