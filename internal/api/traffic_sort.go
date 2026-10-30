package api

import "fmt"

// TrafficSort is the selected ranking for detailed traffic buckets.
type TrafficSort string

const (
	trafficSortQueryKey          = "sort"
	errorValueInvalidTrafficSort = "invalid_sort"
	// TrafficSortPageViews orders buckets by their page-view count.
	TrafficSortPageViews TrafficSort = "page_views"
	// TrafficSortUniqueVisitors orders buckets by their distinct visitor count.
	TrafficSortUniqueVisitors TrafficSort = "unique_visitors"
)

func parseTrafficSort(value string) (TrafficSort, error) {
	switch value {
	case "", string(TrafficSortPageViews):
		return TrafficSortPageViews, nil
	case string(TrafficSortUniqueVisitors):
		return TrafficSortUniqueVisitors, nil
	default:
		return "", fmt.Errorf("parse traffic sort %q: unsupported ranking", value)
	}
}

func (ranking TrafficSort) count(views, unique int64) int64 {
	if ranking == TrafficSortUniqueVisitors {
		return unique
	}
	return views
}

func (ranking TrafficSort) orderClause(key string) string {
	if ranking == TrafficSortUniqueVisitors {
		return "unique_visitor_count desc, " + key + " asc"
	}
	return "visit_count desc, " + key + " asc"
}

type bucketVisitorCounts map[string]map[string]struct{}

func (counts bucketVisitorCounts) add(bucket, visitorID string) {
	if visitorID == "" {
		return
	}
	if counts[bucket] == nil {
		counts[bucket] = make(map[string]struct{})
	}
	counts[bucket][visitorID] = struct{}{}
}
