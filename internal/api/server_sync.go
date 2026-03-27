package api

import (
	"net/http"
)

// sync 在写库成功后调用，使 DNS 引擎重载并同步 UDP 监听；失败时写入错误响应并返回 false。
// UDP 端口占用等运行时错误在引擎内记录并由列表接口 runtime_error 返回，不阻塞本函数成功返回。
func (s *Server) sync(w http.ResponseWriter, r *http.Request) bool {
	if s.Engine == nil {
		return true
	}
	if err := s.Engine.SyncNow(r.Context()); err != nil {
		if s.Log != nil {
			s.Log.Error("dns sync", "err", err)
		}
		writeObjErr(w, http.StatusInternalServerError, CodeDNSReloadFailed)
		return false
	}
	return true
}
