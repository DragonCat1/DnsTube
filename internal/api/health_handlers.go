package api

import "net/http"

// healthz 供负载均衡 / 容器探针使用：仅当 DNS 引擎中无任何 UDP 监听绑定错误时返回 200。
// 不鉴权，不返回错误细节（避免信息泄露）；详见 Engine.ListenerErrors。
func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if errs := s.Engine.ListenerErrors(); len(errs) > 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}
