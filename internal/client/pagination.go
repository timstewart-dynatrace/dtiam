package client

import "strconv"

// PaginationStyle identifies how a Dynatrace API paginates list responses.
//
// The Account Management API is not uniform: some endpoints return the whole
// collection in one response, some use an opaque cursor, and some use 1-based
// page numbers. Sending the wrong parameters is not always an error — several
// endpoints silently ignore unknown query parameters and return only the first
// page, which looks like a complete result.
type PaginationStyle int

const (
	// PaginationNone means the endpoint returns the full collection in a single
	// response. Used by users, groups, policies, bindings, boundaries, and
	// environments, which return {count, items} with no page parameters.
	PaginationNone PaginationStyle = iota

	// PaginationPageKey is cursor-based: the response carries a nextPageKey that
	// is echoed back to fetch the following page. Used by the service user
	// management API, whose responses are {results, nextPageKey, totalCount}.
	PaginationPageKey

	// PaginationPageNumber is 1-based page numbering: request page N until the
	// accumulated item count reaches the reported total. Used by the platform
	// tokens API, whose responses are {pageSize, pageNumber, total, results}.
	PaginationPageNumber
)

// Default page sizes. These are deliberately large: dtiam's job is to return the
// whole collection, so fewer round trips is strictly better, and every endpoint
// below caps oversized requests server-side rather than rejecting them.
const (
	// DefaultPageSize is the page size requested when a config omits one.
	DefaultPageSize = 500

	// MaxPageRequests bounds the paging loop so a server that keeps returning a
	// nextPageKey cannot hang the CLI indefinitely.
	MaxPageRequests = 200
)

// PaginationConfig describes how to page through one endpoint. A nil
// *PaginationConfig means the endpoint is not paginated.
type PaginationConfig struct {
	// Style selects the paging algorithm.
	Style PaginationStyle

	// PageParam is the query parameter carrying the page number
	// ("page" for both paginated Account Management endpoints).
	PageParam string

	// PageSizeParam is the query parameter carrying the page size. It differs
	// per endpoint: "size" for platform tokens, "page-size" for service users,
	// "pageSize" for the environment-level Platform IAM API.
	PageSizeParam string

	// PageKeyParam is the query parameter carrying the opaque cursor. Only used
	// by PaginationPageKey.
	PageKeyParam string

	// ItemsKey is the response field holding the page's items. Empty falls back
	// to the handler's own key resolution.
	ItemsKey string

	// NextKeyField is the response field holding the cursor for the next page.
	// Only used by PaginationPageKey.
	NextKeyField string

	// TotalField is the response field holding the total item count across all
	// pages, used to decide when PaginationPageNumber is done.
	TotalField string

	// PageSize is the number of entries to request per page. Zero means
	// DefaultPageSize.
	PageSize int
}

// EffectivePageSize returns the configured page size, or DefaultPageSize.
func (p *PaginationConfig) EffectivePageSize() int {
	if p == nil || p.PageSize <= 0 {
		return DefaultPageSize
	}
	return p.PageSize
}

// Params builds the query parameters for one page request.
//
// base is the caller's own filter parameters; they are copied rather than
// mutated, and are resent on every page because none of these endpoints embeds
// filters in the page cursor. page is 1-based. pageKey is the cursor from the
// previous response and is empty for the first request.
func (p *PaginationConfig) Params(base map[string]string, page int, pageKey string) map[string]string {
	params := make(map[string]string, len(base)+3)
	for k, v := range base {
		params[k] = v
	}
	if p == nil {
		return params
	}

	if p.PageSizeParam != "" {
		params[p.PageSizeParam] = strconv.Itoa(p.EffectivePageSize())
	}

	switch p.Style {
	case PaginationPageKey:
		// The cursor supersedes the page number once the server issues one.
		if pageKey != "" && p.PageKeyParam != "" {
			params[p.PageKeyParam] = pageKey
		} else if p.PageParam != "" {
			params[p.PageParam] = strconv.Itoa(page)
		}
	case PaginationPageNumber:
		if p.PageParam != "" {
			params[p.PageParam] = strconv.Itoa(page)
		}
	case PaginationNone:
		// No paging parameters.
	}

	return params
}

// ServiceUserPagination returns the paging config for the service user
// management API: GET /iam/v1/accounts/{uuid}/service-users, which responds
// with {results, nextPageKey, totalCount}.
func ServiceUserPagination() *PaginationConfig {
	return &PaginationConfig{
		Style:         PaginationPageKey,
		PageParam:     "page",
		PageSizeParam: "page-size",
		PageKeyParam:  "page-key",
		ItemsKey:      "results",
		NextKeyField:  "nextPageKey",
		TotalField:    "totalCount",
	}
}

// PlatformTokenPagination returns the paging config for the platform tokens
// API: GET /iam/v1/accounts/{uuid}/platform-tokens, which responds with
// {pageSize, pageNumber, total, results}.
func PlatformTokenPagination() *PaginationConfig {
	return &PaginationConfig{
		Style:         PaginationPageNumber,
		PageParam:     "page",
		PageSizeParam: "size",
		ItemsKey:      "results",
		TotalField:    "total",
	}
}

// OrganizationalLevelPagination returns the paging config for the
// environment-level Platform IAM API, which uses page/pageSize.
func OrganizationalLevelPagination() *PaginationConfig {
	return &PaginationConfig{
		Style:         PaginationPageNumber,
		PageParam:     "page",
		PageSizeParam: "pageSize",
		ItemsKey:      "results",
		TotalField:    "totalCount",
	}
}
