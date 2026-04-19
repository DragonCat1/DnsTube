package api

import (
	"sort"
	"strings"

	"github.com/dnstube/dnstube/internal/store"
)

// PaginateSlice 对内存切片分页；limit==0 返回全量。
func PaginateSlice[T any](items []T, limit, page int) ([]T, int, int64) {
	total := int64(len(items))
	if limit == 0 {
		return items, page, total
	}
	if limit < 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	if offset >= len(items) {
		return []T{}, page, total
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end], page, total
}

func sortInstanceListItems(items []instanceListItem, sortBy, order string) {
	if sortBy == "" {
		sortBy = "id"
	}
	desc := strings.EqualFold(order, "desc")
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		var less bool
		switch sortBy {
		case "name":
			if c := strings.Compare(a.Name, b.Name); c != 0 {
				less = c < 0
			} else {
				less = a.ID < b.ID
			}
		case "listen_port":
			if a.ListenPort != b.ListenPort {
				less = a.ListenPort < b.ListenPort
			} else {
				less = a.ID < b.ID
			}
		case "listen_addr":
			if c := strings.Compare(a.ListenAddr, b.ListenAddr); c != 0 {
				less = c < 0
			} else {
				less = a.ID < b.ID
			}
		case "created_at":
			if !a.CreatedAt.Equal(b.CreatedAt) {
				less = a.CreatedAt.Before(b.CreatedAt)
			} else {
				less = a.ID < b.ID
			}
		case "paused":
			if a.Paused == b.Paused {
				less = a.ID < b.ID
			} else {
				less = !a.Paused && b.Paused
			}
		case "record_count":
			if a.RecordCount == b.RecordCount {
				less = a.ID < b.ID
			} else {
				less = a.RecordCount < b.RecordCount
			}
		default:
			less = a.ID < b.ID
		}
		if desc {
			return !less
		}
		return less
	})
}

func sortUpstreamGroupsSlice(groups []store.UpstreamGroup, sortBy, order string) {
	if sortBy == "" {
		sortBy = "id"
	}
	desc := strings.EqualFold(order, "desc")
	sort.SliceStable(groups, func(i, j int) bool {
		a, b := groups[i], groups[j]
		var less bool
		switch sortBy {
		case "name":
			if c := strings.Compare(a.Name, b.Name); c != 0 {
				less = c < 0
			} else {
				less = a.ID < b.ID
			}
		case "created_at":
			if !a.CreatedAt.Equal(b.CreatedAt) {
				less = a.CreatedAt.Before(b.CreatedAt)
			} else {
				less = a.ID < b.ID
			}
		default:
			less = a.ID < b.ID
		}
		if desc {
			return !less
		}
		return less
	})
}

func sortUpstreamServersSlice(servers []store.UpstreamServer, sortBy, order string) {
	if sortBy == "" {
		sortBy = "sort_order"
	}
	desc := strings.EqualFold(order, "desc")
	sort.SliceStable(servers, func(i, j int) bool {
		a, b := servers[i], servers[j]
		var less bool
		switch sortBy {
		case "address":
			if c := strings.Compare(a.Address, b.Address); c != 0 {
				less = c < 0
			} else {
				less = a.ID < b.ID
			}
		case "port":
			if a.Port != b.Port {
				less = a.Port < b.Port
			} else {
				less = a.ID < b.ID
			}
		case "protocol":
			if c := strings.Compare(a.Protocol, b.Protocol); c != 0 {
				less = c < 0
			} else {
				less = a.ID < b.ID
			}
		case "created_at":
			if !a.CreatedAt.Equal(b.CreatedAt) {
				less = a.CreatedAt.Before(b.CreatedAt)
			} else {
				less = a.ID < b.ID
			}
		case "sort_order":
			if a.SortOrder == b.SortOrder {
				less = a.ID < b.ID
			} else {
				less = a.SortOrder < b.SortOrder
			}
		default:
			less = a.ID < b.ID
		}
		if desc {
			return !less
		}
		return less
	})
}

func sortDNSRecordsSlice(recs []store.DNSRecord, sortBy, order string) {
	if sortBy == "" {
		sortBy = "id"
	}
	desc := strings.EqualFold(order, "desc")
	sort.SliceStable(recs, func(i, j int) bool {
		a, b := recs[i], recs[j]
		var less bool
		switch sortBy {
		case "name":
			if c := strings.Compare(a.Name, b.Name); c != 0 {
				less = c < 0
			} else {
				less = a.ID < b.ID
			}
		case "rtype":
			if c := strings.Compare(a.Rtype, b.Rtype); c != 0 {
				less = c < 0
			} else {
				less = a.ID < b.ID
			}
		case "ttl":
			if a.TTL != b.TTL {
				less = a.TTL < b.TTL
			} else {
				less = a.ID < b.ID
			}
		case "created_at":
			if !a.CreatedAt.Equal(b.CreatedAt) {
				less = a.CreatedAt.Before(b.CreatedAt)
			} else {
				less = a.ID < b.ID
			}
		default:
			less = a.ID < b.ID
		}
		if desc {
			return !less
		}
		return less
	})
}

func sortForwardRulesSlice(rules []store.ForwardRule, sortBy, order string) {
	if sortBy == "" {
		sortBy = "priority"
	}
	desc := strings.EqualFold(order, "desc")
	sort.SliceStable(rules, func(i, j int) bool {
		a, b := rules[i], rules[j]
		var less bool
		switch sortBy {
		case "name_pattern":
			if c := strings.Compare(a.NamePattern, b.NamePattern); c != 0 {
				less = c < 0
			} else {
				less = a.ID < b.ID
			}
		case "created_at":
			if !a.CreatedAt.Equal(b.CreatedAt) {
				less = a.CreatedAt.Before(b.CreatedAt)
			} else {
				less = a.ID < b.ID
			}
		case "target_group_id":
			if a.TargetGroupID == b.TargetGroupID {
				less = a.ID < b.ID
			} else {
				less = a.TargetGroupID < b.TargetGroupID
			}
		case "priority":
			if a.Priority == b.Priority {
				less = a.ID < b.ID
			} else {
				less = a.Priority < b.Priority
			}
		case "mode":
			if c := strings.Compare(a.Mode, b.Mode); c != 0 {
				less = c < 0
			} else {
				less = a.ID < b.ID
			}
		case "hit_count":
			if a.HitCount == b.HitCount {
				less = a.ID < b.ID
			} else {
				less = a.HitCount < b.HitCount
			}
		case "pattern_rule_count":
			if a.PatternRuleCount == b.PatternRuleCount {
				less = a.ID < b.ID
			} else {
				less = a.PatternRuleCount < b.PatternRuleCount
			}
		case "pattern_fetched_at":
			switch {
			case a.PatternFetchedAt == nil && b.PatternFetchedAt == nil:
				less = a.ID < b.ID
			case a.PatternFetchedAt == nil:
				less = true
			case b.PatternFetchedAt == nil:
				less = false
			default:
				if a.PatternFetchedAt.Equal(*b.PatternFetchedAt) {
					less = a.ID < b.ID
				} else {
					less = a.PatternFetchedAt.Before(*b.PatternFetchedAt)
				}
			}
		case "disabled":
			if a.Disabled == b.Disabled {
				less = a.ID < b.ID
			} else {
				less = !a.Disabled && b.Disabled
			}
		default:
			less = a.ID < b.ID
		}
		if desc {
			return !less
		}
		return less
	})
}
