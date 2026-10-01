package client

import "testing"

func TestPaginationConfig_EffectivePageSize(t *testing.T) {
	tests := []struct {
		name string
		cfg  *PaginationConfig
		want int
	}{
		{name: "should use default when config is nil", cfg: nil, want: DefaultPageSize},
		{name: "should use default when size is zero", cfg: &PaginationConfig{}, want: DefaultPageSize},
		{name: "should use default when size is negative", cfg: &PaginationConfig{PageSize: -1}, want: DefaultPageSize},
		{name: "should honor explicit size", cfg: &PaginationConfig{PageSize: 25}, want: 25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.EffectivePageSize(); got != tt.want {
				t.Errorf("EffectivePageSize() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPaginationConfig_Params_PageKeyStyle(t *testing.T) {
	cfg := ServiceUserPagination()

	first := cfg.Params(nil, 1, "")
	if first["page"] != "1" {
		t.Errorf("first page: page = %q, want '1'", first["page"])
	}
	if _, ok := first["page-key"]; ok {
		t.Error("first page must not send page-key")
	}
	if first["page-size"] == "" {
		t.Error("first page must send page-size")
	}

	next := cfg.Params(nil, 2, "CURSOR")
	if next["page-key"] != "CURSOR" {
		t.Errorf("page-key = %q, want 'CURSOR'", next["page-key"])
	}
	// Once a cursor exists it supersedes the page number.
	if _, ok := next["page"]; ok {
		t.Error("page must not be sent alongside page-key")
	}
}

func TestPaginationConfig_Params_PageNumberStyle(t *testing.T) {
	cfg := PlatformTokenPagination()

	got := cfg.Params(nil, 3, "")
	if got["page"] != "3" {
		t.Errorf("page = %q, want '3'", got["page"])
	}
	if got["size"] == "" {
		t.Error("size must be sent on every page-number request")
	}
	if _, ok := got["page-key"]; ok {
		t.Error("page-number style must not send page-key")
	}
}

func TestPaginationConfig_Params_DoesNotMutateCallerMap(t *testing.T) {
	base := map[string]string{"service-users": "true"}
	cfg := ServiceUserPagination()

	got := cfg.Params(base, 1, "")
	got["injected"] = "x"

	if len(base) != 1 {
		t.Errorf("caller map was mutated: %v", base)
	}
	if _, ok := base["page"]; ok {
		t.Error("paging params leaked into the caller's map")
	}
}

func TestPaginationConfig_Params_PreservesCallerFilters(t *testing.T) {
	cfg := PlatformTokenPagination()
	got := cfg.Params(map[string]string{"searchTerm": "ci"}, 1, "")
	if got["searchTerm"] != "ci" {
		t.Errorf("searchTerm = %q, want 'ci'", got["searchTerm"])
	}
}

func TestPaginationConfig_Params_NilConfigAddsNothing(t *testing.T) {
	var cfg *PaginationConfig
	got := cfg.Params(map[string]string{"a": "b"}, 2, "KEY")
	if len(got) != 1 || got["a"] != "b" {
		t.Errorf("nil config altered params: %v", got)
	}
}

func TestPresetsMatchDocumentedAPIShapes(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *PaginationConfig
		style         PaginationStyle
		pageSizeParam string
		itemsKey      string
		totalField    string
	}{
		{
			name: "service users", cfg: ServiceUserPagination(), style: PaginationPageKey,
			pageSizeParam: "page-size", itemsKey: "results", totalField: "totalCount",
		},
		{
			name: "platform tokens", cfg: PlatformTokenPagination(), style: PaginationPageNumber,
			pageSizeParam: "size", itemsKey: "results", totalField: "total",
		},
		{
			name: "organizational levels", cfg: OrganizationalLevelPagination(), style: PaginationPageNumber,
			pageSizeParam: "pageSize", itemsKey: "results", totalField: "totalCount",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cfg.Style != tt.style {
				t.Errorf("Style = %v, want %v", tt.cfg.Style, tt.style)
			}
			if tt.cfg.PageSizeParam != tt.pageSizeParam {
				t.Errorf("PageSizeParam = %q, want %q", tt.cfg.PageSizeParam, tt.pageSizeParam)
			}
			if tt.cfg.ItemsKey != tt.itemsKey {
				t.Errorf("ItemsKey = %q, want %q", tt.cfg.ItemsKey, tt.itemsKey)
			}
			if tt.cfg.TotalField != tt.totalField {
				t.Errorf("TotalField = %q, want %q", tt.cfg.TotalField, tt.totalField)
			}
		})
	}
}
