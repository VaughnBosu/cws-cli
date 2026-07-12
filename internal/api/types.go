package api

// Types in this file mirror the Chrome Web Store API v2 contract
// (https://chromewebstore.googleapis.com/$discovery/rest?version=v2).

// UploadResponse represents UploadItemPackageResponse.
type UploadResponse struct {
	Name        string `json:"name,omitempty"`
	ItemID      string `json:"itemId,omitempty"`
	CrxVersion  string `json:"crxVersion,omitempty"`
	UploadState string `json:"uploadState"`
}

// PublishRequest represents PublishItemRequest.
type PublishRequest struct {
	PublishType     string       `json:"publishType,omitempty"`
	SkipReview      bool         `json:"skipReview,omitempty"`
	BlockOnWarnings bool         `json:"blockOnWarnings,omitempty"`
	DeployInfos     []DeployInfo `json:"deployInfos,omitempty"`
}

// DeployInfo carries the desired initial rollout percentage for a publish.
type DeployInfo struct {
	DeployPercentage int `json:"deployPercentage"`
}

// PublishResponse represents PublishItemResponse.
type PublishResponse struct {
	Name        string        `json:"name,omitempty"`
	ItemID      string        `json:"itemId,omitempty"`
	State       string        `json:"state,omitempty"`
	WarningInfo *WarningsInfo `json:"warningInfo,omitempty"`
}

// WarningsInfo holds non-blocking warnings returned by a publish.
type WarningsInfo struct {
	Warnings []Warning `json:"warnings,omitempty"`
}

// Warning is a single non-blocking publish warning.
type Warning struct {
	Reason      string `json:"reason,omitempty"`
	Description string `json:"description,omitempty"`
}

// StatusResponse represents FetchItemStatusResponse.
type StatusResponse struct {
	Name                        string              `json:"name"`
	ItemID                      string              `json:"itemId"`
	PublicKey                   string              `json:"publicKey,omitempty"`
	Warned                      bool                `json:"warned,omitempty"`
	TakenDown                   bool                `json:"takenDown,omitempty"`
	PublishedItemRevisionStatus *ItemRevisionStatus `json:"publishedItemRevisionStatus,omitempty"`
	SubmittedItemRevisionStatus *ItemRevisionStatus `json:"submittedItemRevisionStatus,omitempty"`
	LastAsyncUploadState        string              `json:"lastAsyncUploadState,omitempty"`
}

// ItemRevisionStatus represents the status of an item revision (published or submitted).
type ItemRevisionStatus struct {
	State                string                `json:"state"`
	CrxVersion           string                `json:"crxVersion,omitempty"`
	DistributionChannels []DistributionChannel `json:"distributionChannels,omitempty"`
}

// Version returns the revision's crxVersion, falling back to the first
// distribution channel's version.
func (r *ItemRevisionStatus) Version() string {
	if r == nil {
		return ""
	}
	if r.CrxVersion != "" {
		return r.CrxVersion
	}
	if len(r.DistributionChannels) > 0 {
		return r.DistributionChannels[0].CrxVersion
	}
	return ""
}

// DistributionChannel represents a distribution channel for a revision.
type DistributionChannel struct {
	DeployPercentage int    `json:"deployPercentage"`
	CrxVersion       string `json:"crxVersion"`
}

// APIError represents a Google API error response.
type APIError struct {
	Error *APIErrorBody `json:"error,omitempty"`
}

// APIErrorBody represents the body of a Google API error.
type APIErrorBody struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Status  string           `json:"status"`
	Details []APIErrorDetail `json:"details,omitempty"`
}

// APIErrorDetail represents a detail entry in a Google API error.
type APIErrorDetail struct {
	Type            string           `json:"@type"`
	Reason          string           `json:"reason,omitempty"`
	FieldViolations []FieldViolation `json:"fieldViolations,omitempty"`
}

// FieldViolation represents a field-level error.
type FieldViolation struct {
	Field       string `json:"field"`
	Description string `json:"description"`
	Reason      string `json:"reason,omitempty"`
}

// DeployPercentageRequest represents SetPublishedDeployPercentageRequest.
type DeployPercentageRequest struct {
	DeployPercentage int `json:"deployPercentage"`
}

const (
	UploadStateUnspecified = "UPLOAD_STATE_UNSPECIFIED"
	UploadStateSucceeded   = "SUCCEEDED"
	UploadStateInProgress  = "IN_PROGRESS"
	UploadStateFailed      = "FAILED"
	UploadStateNotFound    = "NOT_FOUND"
)

const (
	PublishTypeDefault = "DEFAULT_PUBLISH"
	PublishTypeStaged  = "STAGED_PUBLISH"
)

func IsUploadInProgress(state string) bool {
	return state == UploadStateInProgress
}

func IsUploadSucceeded(state string) bool {
	return state == UploadStateSucceeded
}

func IsUploadFailed(state string) bool {
	return state == UploadStateFailed
}
