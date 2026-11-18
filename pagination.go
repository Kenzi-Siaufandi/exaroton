package exaroton

import (
	"net/url"
	"strconv"
)

// PageInfo holds the pagination metadata returned by paginated endpoints.
type PageInfo struct {
	// The requested page.
	Page int `json:"page"`

	// The last page with items (at least 1).
	LastPage int `json:"lastPage"`

	// Total number of items in all pages.
	Total int `json:"total"`
}

// pageQuery builds the ?page= query for paginated endpoints. Pages start at
// 1; a value <= 0 omits the parameter so the API returns the first page.
func pageQuery(page int) url.Values {
	if page <= 0 {
		return nil
	}
	return url.Values{"page": []string{strconv.Itoa(page)}}
}
