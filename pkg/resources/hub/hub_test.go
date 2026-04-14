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
