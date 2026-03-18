package api

// UploadResponse represents the response from the upload endpoint.
type UploadResponse struct {
	ID          string      `json:"id"`
	UploadState string      `json:"uploadState"`
	ItemError   []ItemError `json:"itemError,omitempty"`
}

// ItemError represents an error returned by the API.
type ItemError struct {
	ErrorCode   string `json:"error_code"`
	ErrorDetail string `json:"error_detail"`
}

// PublishRequest represents the request body for the publish endpoint.
type PublishRequest struct {
	PublishType string `json:"publishType,omitempty"`
}

// PublishResponse represents the response from the publish endpoint.
type PublishResponse struct {
	Status     []string `json:"status"`
	StatusCode string   `json:"statusCode,omitempty"`
}

// StatusResponse represents the response from the fetchStatus endpoint (V2 API).
type StatusResponse struct {
	Name                         string                `json:"name"`
	ItemID                       string                `json:"itemId"`
	PublishedItemRevisionStatus  *ItemRevisionStatus   `json:"publishedItemRevisionStatus,omitempty"`
	SubmittedItemRevisionStatus  *ItemRevisionStatus   `json:"submittedItemRevisionStatus,omitempty"`
	LastAsyncUploadState         string                `json:"lastAsyncUploadState,omitempty"`
	ItemError                    []ItemError           `json:"itemError,omitempty"`
}

// ItemRevisionStatus represents the status of an item revision (published, in-review, or draft).
type ItemRevisionStatus struct {
	State                string                `json:"state"`
	CrxVersion           string                `json:"crxVersion,omitempty"`
	DistributionChannels []DistributionChannel `json:"distributionChannels,omitempty"`
}

// DistributionChannel represents a distribution channel for a published extension.
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

// DeployPercentageRequest represents the request body for setPublishedDeployPercentage.
type DeployPercentageRequest struct {
	DeployPercentage int `json:"deployPercentage"`
}

// DeployPercentageResponse represents the response from setPublishedDeployPercentage.
type DeployPercentageResponse struct {
	DeployPercentage int    `json:"deployPercentage"`
	Status           string `json:"status,omitempty"`
}

// CancelResponse represents the response from cancelSubmission.
type CancelResponse struct {
	Status string `json:"status,omitempty"`
}
