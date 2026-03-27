package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

func parseDashboardTimeRange(r *http.Request) (rng string, from, to time.Time, inst *int32, clientIP *string) {
	rng = r.URL.Query().Get("range")
	if rng == "" {
		rng = "day"
	}
	// 统一按服务端本地时区计算统计窗口。
	now := time.Now()
	switch rng {
	case "day":
		from = now.AddDate(0, 0, -30)
	case "week":
		from = now.AddDate(0, 0, -7*48)
	case "month":
		from = now.AddDate(0, -12, 0)
	case "year":
		from = now.AddDate(-5, 0, 0)
	default:
		from = now.AddDate(0, 0, -30)
	}
	to = now
	if v := r.URL.Query().Get("instance_id"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err == nil {
			x := int32(n)
			inst = &x
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("client_ip")); v != "" {
		clientIP = &v
	}
	return rng, from, to, inst, clientIP
}

func (s *Server) dashboardStats(w http.ResponseWriter, r *http.Request) {
	rng, from, to, inst, clientIP := parseDashboardTimeRange(r)

	topN, _ := strconv.Atoi(r.URL.Query().Get("top_n"))
	if topN <= 0 {
		topN = 10
	}

	series, err := s.Store.DashboardSeries(r.Context(), inst, from, to, rng, clientIP)
	if err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	top, err := s.Store.DashboardTopClients(r.Context(), inst, from, to, topN, clientIP)
	if err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	total, err := s.Store.CountQueriesInRange(r.Context(), inst, from, to, clientIP)
	if err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	writeObjOK(w, http.StatusOK, map[string]any{
		"range":         rng,
		"from":          from,
		"to":            to,
		"total_queries": total,
		"series":        series,
		"top_clients":   top,
	})
}

// dashboardSummary 聚合概览：实例统计、监听异常、TOP 域名、QTYPE/RCODE 分布、缓存/转发占比（与 range / instance_id 查询参数一致）。
func (s *Server) dashboardSummary(w http.ResponseWriter, r *http.Request) {
	rng, from, to, inst, clientIP := parseDashboardTimeRange(r)
	topN, _ := strconv.Atoi(r.URL.Query().Get("top_n"))
	if topN <= 0 {
		topN = 10
	}

	ctx := r.Context()
	instTotal, instPaused, err := s.Store.CountInstancesOverview(ctx)
	if err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}

	listenerErrStr := map[string]string{}
	var instancesWithListenerErr int32
	if s.Engine != nil {
		if m := s.Engine.ListenerErrors(); m != nil {
			for id, msg := range m {
				if msg == "" {
					continue
				}
				if inst != nil && id != *inst {
					continue
				}
				listenerErrStr[strconv.FormatInt(int64(id), 10)] = msg
				instancesWithListenerErr++
			}
		}
	}

	topQ, err := s.Store.DashboardTopQnames(ctx, inst, from, to, topN, clientIP)
	if err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	qtypes, err := s.Store.DashboardQtypeDistribution(ctx, inst, from, to, clientIP)
	if err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	rcodes, err := s.Store.DashboardRcodeDistribution(ctx, inst, from, to, clientIP)
	if err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	breakdown, err := s.Store.DashboardCacheForwardBreakdown(ctx, inst, from, to, clientIP)
	if err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	totalQ, err := s.Store.CountQueriesInRange(ctx, inst, from, to, clientIP)
	if err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}
	totalAll, err := s.Store.CountQueryLogsAll(ctx, inst)
	if err != nil {
		writeObjErr(w, http.StatusInternalServerError, mapStoreErr(err))
		return
	}

	writeObjOK(w, http.StatusOK, map[string]any{
		"range":                         rng,
		"from":                          from,
		"to":                            to,
		"total_queries":                 totalQ,
		"total_queries_all":             totalAll,
		"instance_total":                instTotal,
		"instance_paused":               instPaused,
		"instances_with_listener_error": instancesWithListenerErr,
		"listener_errors":               listenerErrStr,
		"top_qnames":                    topQ,
		"qtype_distribution":            qtypes,
		"rcode_distribution":            rcodes,
		"cache_forward":                 breakdown,
	})
}
