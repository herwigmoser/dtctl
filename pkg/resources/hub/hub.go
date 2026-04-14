package hub

import (
	"fmt"

	"github.com/dynatrace-oss/dtctl/pkg/client"
)

// Handler handles Dynatrace Hub catalog resources
type Handler struct {
	client *client.Client
}

// NewHandler creates a new Hub handler
func NewHandler(c *client.Client) *Handler {
	return &Handler{client: c}
}

// ---------------------------------------------------------------------------
// Hub Apps
// ---------------------------------------------------------------------------

// HubApp represents a Dynatrace Hub catalog app
type HubApp struct {
	ID          string `json:"id" table:"ID"`
	Name        string `json:"name" table:"NAME"`
	Version     string `json:"version,omitempty" table:"VERSION"`
	Publisher   string `json:"publisher,omitempty" table:"PUBLISHER,wide"`
	Category    string `json:"category,omitempty" table:"CATEGORY,wide"`
	Description string `json:"description,omitempty" table:"DESCRIPTION,wide"`
	Type        string `json:"type,omitempty" table:"-"`
}

// HubAppList represents a paginated list of Hub apps
type HubAppList struct {
	Items       []HubApp `json:"items"`
	TotalCount  int      `json:"totalCount"`
	NextPageKey string   `json:"nextPageKey,omitempty"`
}

// HubAppRelease represents a release of a Hub app
type HubAppRelease struct {
	Version     string `json:"version" table:"VERSION"`
	ReleaseDate string `json:"releaseDate,omitempty" table:"RELEASE_DATE,wide"`
	Notes       string `json:"notes,omitempty" table:"-"`
}

// HubAppReleaseList represents a list of Hub app releases
type HubAppReleaseList struct {
	Items       []HubAppRelease `json:"items"`
	TotalCount  int             `json:"totalCount"`
	NextPageKey string          `json:"nextPageKey,omitempty"`
}

// ListApps lists all Hub catalog apps with automatic pagination
func (h *Handler) ListApps(chunkSize int64) (*HubAppList, error) {
	var allItems []HubApp
	var totalCount int
	nextPageKey := ""

	for {
		var result HubAppList
		req := h.client.HTTP().R().SetResult(&result)

		client.PaginationParams{
			Style:         client.PaginationDefault,
			PageKeyParam:  "page-key",
			PageSizeParam: "page-size",
			NextPageKey:   nextPageKey,
			PageSize:      chunkSize,
		}.Apply(req)

		resp, err := req.Get("/platform/hub/v1/catalog/apps")
		if err != nil {
			return nil, fmt.Errorf("failed to list Hub apps: %w", err)
		}
		if resp.IsError() {
			return nil, fmt.Errorf("failed to list Hub apps: status %d: %s", resp.StatusCode(), resp.String())
		}

		allItems = append(allItems, result.Items...)
		totalCount = result.TotalCount

		if chunkSize == 0 {
			return &result, nil
		}
		if result.NextPageKey == "" {
			break
		}
		nextPageKey = result.NextPageKey
	}

	return &HubAppList{Items: allItems, TotalCount: totalCount}, nil
}

// GetApp gets a specific Hub app by ID
func (h *Handler) GetApp(id string) (*HubApp, error) {
	var result HubApp

	resp, err := h.client.HTTP().R().
		SetResult(&result).
		Get(fmt.Sprintf("/platform/hub/v1/catalog/apps/%s", id))

	if err != nil {
		return nil, fmt.Errorf("failed to get Hub app: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("failed to get Hub app %q: status %d: %s", id, resp.StatusCode(), resp.String())
	}

	return &result, nil
}

// ListAppReleases lists all releases for a Hub app
func (h *Handler) ListAppReleases(id string, chunkSize int64) (*HubAppReleaseList, error) {
	var allItems []HubAppRelease
	var totalCount int
	nextPageKey := ""

	for {
		var result HubAppReleaseList
		req := h.client.HTTP().R().SetResult(&result)

		client.PaginationParams{
			Style:         client.PaginationDefault,
			PageKeyParam:  "page-key",
			PageSizeParam: "page-size",
			NextPageKey:   nextPageKey,
			PageSize:      chunkSize,
		}.Apply(req)

		resp, err := req.Get(fmt.Sprintf("/platform/hub/v1/catalog/apps/%s/releases", id))
		if err != nil {
			return nil, fmt.Errorf("failed to list releases for Hub app %q: %w", id, err)
		}
		if resp.IsError() {
			return nil, fmt.Errorf("failed to list releases for Hub app %q: status %d: %s", id, resp.StatusCode(), resp.String())
		}

		allItems = append(allItems, result.Items...)
		totalCount = result.TotalCount

		if chunkSize == 0 {
			return &result, nil
		}
		if result.NextPageKey == "" {
			break
		}
		nextPageKey = result.NextPageKey
	}

	return &HubAppReleaseList{Items: allItems, TotalCount: totalCount}, nil
}

// ---------------------------------------------------------------------------
// Hub Extensions
// ---------------------------------------------------------------------------

// HubExtension represents a Dynatrace Hub catalog extension
type HubExtension struct {
	ID            string `json:"id" table:"ID"`
	Name          string `json:"name" table:"NAME"`
	LatestVersion string `json:"latestVersion,omitempty" table:"LATEST_VERSION"`
	Publisher     string `json:"publisher,omitempty" table:"PUBLISHER,wide"`
	Category      string `json:"category,omitempty" table:"CATEGORY,wide"`
	Description   string `json:"description,omitempty" table:"DESCRIPTION,wide"`
	Type          string `json:"type,omitempty" table:"-"`
}

// HubExtensionList represents a paginated list of Hub extensions
type HubExtensionList struct {
	Items       []HubExtension `json:"items"`
	TotalCount  int            `json:"totalCount"`
	NextPageKey string         `json:"nextPageKey,omitempty"`
}

// HubExtensionRelease represents a release of a Hub extension
type HubExtensionRelease struct {
	Version     string `json:"version" table:"VERSION"`
	ReleaseDate string `json:"releaseDate,omitempty" table:"RELEASE_DATE,wide"`
	Notes       string `json:"notes,omitempty" table:"-"`
}

// HubExtensionReleaseList represents a list of Hub extension releases
type HubExtensionReleaseList struct {
	Items       []HubExtensionRelease `json:"items"`
	TotalCount  int                   `json:"totalCount"`
	NextPageKey string                `json:"nextPageKey,omitempty"`
}

// ListExtensions lists all Hub catalog extensions with automatic pagination
func (h *Handler) ListExtensions(chunkSize int64) (*HubExtensionList, error) {
	var allItems []HubExtension
	var totalCount int
	nextPageKey := ""

	for {
		var result HubExtensionList
		req := h.client.HTTP().R().SetResult(&result)

		client.PaginationParams{
			Style:         client.PaginationDefault,
			PageKeyParam:  "page-key",
			PageSizeParam: "page-size",
			NextPageKey:   nextPageKey,
			PageSize:      chunkSize,
		}.Apply(req)

		resp, err := req.Get("/platform/hub/v1/catalog/extensions")
		if err != nil {
			return nil, fmt.Errorf("failed to list Hub extensions: %w", err)
		}
		if resp.IsError() {
			return nil, fmt.Errorf("failed to list Hub extensions: status %d: %s", resp.StatusCode(), resp.String())
		}

		allItems = append(allItems, result.Items...)
		totalCount = result.TotalCount

		if chunkSize == 0 {
			return &result, nil
		}
		if result.NextPageKey == "" {
			break
		}
		nextPageKey = result.NextPageKey
	}

	return &HubExtensionList{Items: allItems, TotalCount: totalCount}, nil
}

// GetExtension gets a specific Hub extension by ID
func (h *Handler) GetExtension(id string) (*HubExtension, error) {
	var result HubExtension

	resp, err := h.client.HTTP().R().
		SetResult(&result).
		Get(fmt.Sprintf("/platform/hub/v1/catalog/extensions/%s", id))

	if err != nil {
		return nil, fmt.Errorf("failed to get Hub extension: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("failed to get Hub extension %q: status %d: %s", id, resp.StatusCode(), resp.String())
	}

	return &result, nil
}

// ListExtensionReleases lists all releases for a Hub extension
func (h *Handler) ListExtensionReleases(id string, chunkSize int64) (*HubExtensionReleaseList, error) {
	var allItems []HubExtensionRelease
	var totalCount int
	nextPageKey := ""

	for {
		var result HubExtensionReleaseList
		req := h.client.HTTP().R().SetResult(&result)

		client.PaginationParams{
			Style:         client.PaginationDefault,
			PageKeyParam:  "page-key",
			PageSizeParam: "page-size",
			NextPageKey:   nextPageKey,
			PageSize:      chunkSize,
		}.Apply(req)

		resp, err := req.Get(fmt.Sprintf("/platform/hub/v1/catalog/extensions/%s/releases", id))
		if err != nil {
			return nil, fmt.Errorf("failed to list releases for Hub extension %q: %w", id, err)
		}
		if resp.IsError() {
			return nil, fmt.Errorf("failed to list releases for Hub extension %q: status %d: %s", id, resp.StatusCode(), resp.String())
		}

		allItems = append(allItems, result.Items...)
		totalCount = result.TotalCount

		if chunkSize == 0 {
			return &result, nil
		}
		if result.NextPageKey == "" {
			break
		}
		nextPageKey = result.NextPageKey
	}

	return &HubExtensionReleaseList{Items: allItems, TotalCount: totalCount}, nil
}
