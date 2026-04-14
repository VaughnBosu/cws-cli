package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/vaughnbosu/cws-cli/internal/api"
	"github.com/vaughnbosu/cws-cli/internal/auth"
)

const chromeWebStoreDiscoveryURL = "https://chromewebstore.googleapis.com/$discovery/rest?version=v2"

type discoveryRef struct {
	Ref string `json:"$ref"`
}

type discoveryMethod struct {
	Request  discoveryRef `json:"request"`
	Response discoveryRef `json:"response"`
}

type discoverySchema struct {
	Properties map[string]discoverySchema `json:"properties"`
	Enum       []string                   `json:"enum"`
}

type discoveryDoc struct {
	Resources struct {
		Publishers struct {
			Resources struct {
				Items struct {
					Methods struct {
						Publish                      discoveryMethod `json:"publish"`
						SetPublishedDeployPercentage discoveryMethod `json:"setPublishedDeployPercentage"`
					} `json:"methods"`
				} `json:"items"`
			} `json:"resources"`
		} `json:"publishers"`
	} `json:"resources"`
	Schemas map[string]discoverySchema `json:"schemas"`
}

// These tests are opt-in because they hit Google's live discovery/auth endpoints.
// They are safe for CI when skipped and do not mutate store items.

func TestLiveDiscoveryContract(t *testing.T) {
	if os.Getenv("CWS_LIVE_CONTRACT") == "" {
		t.Skip("set CWS_LIVE_CONTRACT=1 to validate against the live Chrome Web Store discovery document")
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, chromeWebStoreDiscoveryURL, nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("fetching discovery document: %v", err)
	}
	defer resp.Body.Close()

	var doc discoveryDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatalf("decoding discovery document: %v", err)
	}

	if got := doc.Resources.Publishers.Resources.Items.Methods.Publish.Response.Ref; got != "PublishItemResponse" {
		t.Fatalf("publish response ref = %q, want PublishItemResponse", got)
	}

	if got := doc.Resources.Publishers.Resources.Items.Methods.SetPublishedDeployPercentage.Response.Ref; got != "SetPublishedDeployPercentageResponse" {
		t.Fatalf("rollout response ref = %q, want SetPublishedDeployPercentageResponse", got)
	}

	uploadSchema := doc.Schemas["UploadItemPackageResponse"]
	uploadState := uploadSchema.Properties["uploadState"]
	for _, want := range []string{"SUCCEEDED", "IN_PROGRESS", "FAILED"} {
		if !slices.Contains(uploadState.Enum, want) {
			t.Fatalf("uploadState enum missing %q in live discovery doc: %v", want, uploadState.Enum)
		}
	}

	publishSchema := doc.Schemas["PublishItemResponse"]
	if _, ok := publishSchema.Properties["state"]; !ok {
		t.Fatal("PublishItemResponse.state missing from live discovery doc")
	}

	rolloutSchema := doc.Schemas["SetPublishedDeployPercentageResponse"]
	if len(rolloutSchema.Properties) != 0 {
		t.Fatalf("expected empty rollout success schema, got properties: %v", rolloutSchema.Properties)
	}
}

func TestLiveFetchStatus(t *testing.T) {
	if os.Getenv("CWS_LIVE_FETCH_STATUS") == "" {
		t.Skip("set CWS_LIVE_FETCH_STATUS=1 to run an authenticated live fetchStatus smoke test")
	}

	clientID := os.Getenv("CWS_CLIENT_ID")
	clientSecret := os.Getenv("CWS_CLIENT_SECRET")
	refreshToken := os.Getenv("CWS_REFRESH_TOKEN")
	publisherID := os.Getenv("CWS_PUBLISHER_ID")
	extensionID := os.Getenv("CWS_EXTENSION_ID")
	if clientID == "" || clientSecret == "" || refreshToken == "" || publisherID == "" || extensionID == "" {
		t.Skip("CWS_CLIENT_ID, CWS_CLIENT_SECRET, CWS_REFRESH_TOKEN, CWS_PUBLISHER_ID, and CWS_EXTENSION_ID must be set")
	}

	authenticator := auth.NewOAuthAuthenticator(clientID, clientSecret, refreshToken)
	client := api.NewClient(authenticator, publisherID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, _, err := client.FetchStatus(ctx, extensionID)
	if err != nil {
		t.Fatalf("live fetchStatus failed: %v", err)
	}
	if resp.ItemID != extensionID {
		t.Fatalf("live fetchStatus itemId = %q, want %q", resp.ItemID, extensionID)
	}
}
