package api

import (
	"net/http"
	"strconv"
	"strings"
)

// ListQuery 列表接口统一查询参数（camelCase；兼容 sort_by / sort_order）。
type ListQuery struct {
	Limit     int
	Page      int
	SortBy    string
	SortOrder string
}

// ParseListQuery 解析 limit、page、sortBy、sortOrder。
// limit 未传时使用 defaultLimit；limit==0 表示不分页（由业务层处理）。
// page 默认 1。
func ParseListQuery(r *http.Request, defaultLimit int) ListQuery {
	q := r.URL.Query()
	var limit int
	if _, ok := q["limit"]; ok {
		limit, _ = strconv.Atoi(strings.TrimSpace(q.Get("limit")))
	} else {
		limit = defaultLimit
	}
	page, _ := strconv.Atoi(strings.TrimSpace(q.Get("page")))
	if page <= 0 {
		page = 1
	}
	sortBy := strings.TrimSpace(q.Get("sortBy"))
	if sortBy == "" {
		sortBy = strings.TrimSpace(q.Get("sort_by"))
	}
	sortOrder := strings.TrimSpace(q.Get("sortOrder"))
	if sortOrder == "" {
		sortOrder = strings.TrimSpace(q.Get("sort_order"))
	}
	return ListQuery{
		Limit:     limit,
		Page:      page,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}
}
