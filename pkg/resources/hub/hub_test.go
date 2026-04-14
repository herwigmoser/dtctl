package hub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dynatrace-oss/dtctl/pkg/client"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestClient(t *testing.T, url string) *client.Client {
	t.Helper()
	c, err := client.NewForTesting(url, "test-token")
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}
	return c
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// paginationGuard rejects page-size combined with page-key (PaginationDefault constraint)
func paginationGuard(t *testing.T, w http.ResponseWriter, r *http.Request) bool {
	t.Helper()
	if r.URL.Query().Get("page-key") != "" && r.URL.Query().Get("page-size") != "" {
		t.Error("page-size must not be sent with page-key")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Constraints violated."})
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// NewHandler
// ---------------------------------------------------------------------------

func TestNewHandler(t *testing.T) {
	c := newTestClient(t, "https://test.example.invalid")
	h := NewHandler(c)
	if h == nil {
		t.Fatal("NewHandler() returned nil")
	}
	if h.client == nil {
		t.Error("Handler.client is nil")
	}
}

// ---------------------------------------------------------------------------
// ListApps
// ---------------------------------------------------------------------------

func TestListApps(t *testing.T) {
	tests := []struct {
		name      string
		chunkSize int64
		pages     []HubAppList
		validate  func(*testing.T, *HubAppList)
	}{
		{
			name:      "single page no chunking",
			chunkSize: 0,
			pages: []HubAppList{
				{
					TotalCount: 2,
					Items: []HubApp{
						{ID: "app-001", Name: "App One", Version: "1.0.0"},
						{ID: "app-002", Name: "App Two", Version: "2.0.0"},
					},
				},
			},
			validate: func(t *testing.T, r *HubAppList) {
				if len(r.Items) != 2 {
					t.Errorf("expected 2 items, got %d", len(r.Items))
				}
			},
		},
		{
			name:      "paginated across two pages",
			chunkSize: 50,
			pages: []HubAppList{
				{
					TotalCount:  3,
					NextPageKey: "page2key",
					Items: []HubApp{
						{ID: "app-001", Name: "App One"},
						{ID: "app-002", Name: "App Two"},
					},
				},
				{
					TotalCount: 3,
					Items: []HubApp{
						{ID: "app-003", Name: "App Three"},
					},
				},
			},
			validate: func(t *testing.T, r *HubAppList) {
				if len(r.Items) != 3 {
					t.Errorf("expected 3 items across pages, got %d", len(r.Items))
				}
				if r.TotalCount != 3 {
					t.Errorf("expected TotalCount 3, got %d", r.TotalCount)
				}
			},
		},
		{
			name:      "empty list",
			chunkSize: 0,
			pages: []HubAppList{
				{TotalCount: 0, Items: []HubApp{}},
			},
			validate: func(t *testing.T, r *HubAppList) {
				if len(r.Items) != 0 {
					t.Errorf("expected 0 items, got %d", len(r.Items))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageIdx := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/platform/hub/v1/catalog/apps" {
					t.Errorf("unexpected path: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if paginationGuard(t, w, r) {
					return
				}
				if pageIdx >= len(tt.pages) {
					t.Error("more requests than expected pages")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				writeJSON(w, http.StatusOK, tt.pages[pageIdx])
				pageIdx++
			}))
			defer server.Close()

			h := NewHandler(newTestClient(t, server.URL))
			result, err := h.ListApps(tt.chunkSize)
			if err != nil {
				t.Fatalf("ListApps() unexpected error: %v", err)
			}
			tt.validate(t, result)
		})
	}
}

func TestListApps_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}))
	defer server.Close()

	h := NewHandler(newTestClient(t, server.URL))
	_, err := h.ListApps(0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetApp
// ---------------------------------------------------------------------------

func TestGetApp(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		statusCode    int
		response      interface{}
		expectError   bool
		errorContains string
	}{
		{
			name:       "success",
			id:         "app-001",
			statusCode: http.StatusOK,
			response:   HubApp{ID: "app-001", Name: "App One", Version: "1.0.0"},
		},
		{
			name:          "not found",
			id:            "missing",
			statusCode:    http.StatusNotFound,
			response:      map[string]string{"error": "not found"},
			expectError:   true,
			errorContains: "status 404",
		},
		{
			name:          "server error",
			id:            "app-err",
			statusCode:    http.StatusInternalServerError,
			response:      map[string]string{"error": "internal"},
			expectError:   true,
			errorContains: "status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expected := "/platform/hub/v1/catalog/apps/" + tt.id
				if r.URL.Path != expected {
					t.Errorf("unexpected path: %s, want %s", r.URL.Path, expected)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				writeJSON(w, tt.statusCode, tt.response)
			}))
			defer server.Close()

			h := NewHandler(newTestClient(t, server.URL))
			result, err := h.GetApp(tt.id)

			if (err != nil) != tt.expectError {
				t.Errorf("GetApp() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if !tt.expectError && result.ID != tt.id {
				t.Errorf("expected ID %q, got %q", tt.id, result.ID)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ListAppReleases
// ---------------------------------------------------------------------------

func TestListAppReleases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/platform/hub/v1/catalog/apps/app-001/releases" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if paginationGuard(t, w, r) {
			return
		}
		writeJSON(w, http.StatusOK, HubAppReleaseList{
			TotalCount: 2,
			Items: []HubAppRelease{
				{Version: "1.1.0", ReleaseDate: "2024-11-01"},
				{Version: "1.0.0", ReleaseDate: "2024-09-15"},
			},
		})
	}))
	defer server.Close()

	h := NewHandler(newTestClient(t, server.URL))
	result, err := h.ListAppReleases("app-001", 0)
	if err != nil {
		t.Fatalf("ListAppReleases() unexpected error: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 releases, got %d", len(result.Items))
	}
	if result.Items[0].Version != "1.1.0" {
		t.Errorf("expected first release 1.1.0, got %q", result.Items[0].Version)
	}
}

// ---------------------------------------------------------------------------
// ListExtensions
// ---------------------------------------------------------------------------

func TestListExtensions(t *testing.T) {
	tests := []struct {
		name      string
		chunkSize int64
		pages     []HubExtensionList
		validate  func(*testing.T, *HubExtensionList)
	}{
		{
			name:      "single page no chunking",
			chunkSize: 0,
			pages: []HubExtensionList{
				{
					TotalCount: 2,
					Items: []HubExtension{
						{ID: "ext-001", Name: "Extension One", LatestVersion: "1.0.0"},
						{ID: "ext-002", Name: "Extension Two", LatestVersion: "2.1.3"},
					},
				},
			},
			validate: func(t *testing.T, r *HubExtensionList) {
				if len(r.Items) != 2 {
					t.Errorf("expected 2 items, got %d", len(r.Items))
				}
			},
		},
		{
			name:      "paginated across two pages",
			chunkSize: 50,
			pages: []HubExtensionList{
				{
					TotalCount:  3,
					NextPageKey: "page2key",
					Items: []HubExtension{
						{ID: "ext-001", Name: "Extension One"},
						{ID: "ext-002", Name: "Extension Two"},
					},
				},
				{
					TotalCount: 3,
					Items: []HubExtension{
						{ID: "ext-003", Name: "Extension Three"},
					},
				},
			},
			validate: func(t *testing.T, r *HubExtensionList) {
				if len(r.Items) != 3 {
					t.Errorf("expected 3 items across pages, got %d", len(r.Items))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageIdx := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/platform/hub/v1/catalog/extensions" {
					t.Errorf("unexpected path: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if paginationGuard(t, w, r) {
					return
				}
				if pageIdx >= len(tt.pages) {
					t.Error("more requests than expected pages")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				writeJSON(w, http.StatusOK, tt.pages[pageIdx])
				pageIdx++
			}))
			defer server.Close()

			h := NewHandler(newTestClient(t, server.URL))
			result, err := h.ListExtensions(tt.chunkSize)
			if err != nil {
				t.Fatalf("ListExtensions() unexpected error: %v", err)
			}
			tt.validate(t, result)
		})
	}
}

// ---------------------------------------------------------------------------
// GetExtension
// ---------------------------------------------------------------------------

func TestGetExtension(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/platform/hub/v1/catalog/extensions/ext-001" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, HubExtension{
			ID:            "ext-001",
			Name:          "Extension One",
			LatestVersion: "1.0.0",
			Publisher:     "Dynatrace",
		})
	}))
	defer server.Close()

	h := NewHandler(newTestClient(t, server.URL))
	result, err := h.GetExtension("ext-001")
	if err != nil {
		t.Fatalf("GetExtension() unexpected error: %v", err)
	}
	if result.ID != "ext-001" {
		t.Errorf("expected ID ext-001, got %q", result.ID)
	}
	if result.Publisher != "Dynatrace" {
		t.Errorf("expected publisher Dynatrace, got %q", result.Publisher)
	}
}

func TestGetExtension_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	}))
	defer server.Close()

	h := NewHandler(newTestClient(t, server.URL))
	_, err := h.GetExtension("missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// ListExtensionReleases
// ---------------------------------------------------------------------------

func TestListExtensionReleases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/platform/hub/v1/catalog/extensions/ext-001/releases" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if paginationGuard(t, w, r) {
			return
		}
		writeJSON(w, http.StatusOK, HubExtensionReleaseList{
			TotalCount: 2,
			Items: []HubExtensionRelease{
				{Version: "1.0.1", ReleaseDate: "2024-12-01"},
				{Version: "1.0.0", ReleaseDate: "2024-10-10"},
			},
		})
	}))
	defer server.Close()

	h := NewHandler(newTestClient(t, server.URL))
	result, err := h.ListExtensionReleases("ext-001", 0)
	if err != nil {
		t.Fatalf("ListExtensionReleases() unexpected error: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 releases, got %d", len(result.Items))
	}
}
