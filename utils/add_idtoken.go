package utils

import (
	"context"
	"fmt"
	"net/http"

	"cloud.google.com/go/compute/metadata"
)

// AddServiceAccountToken adds a the service account ID token to the request
// from the Metadata API.
func AddServiceAccountToken(ctx context.Context, req *http.Request, audienceURL string) (*http.Request, error) {
	idToken, err := GetServiceAccountToken(ctx, audienceURL)
	if err != nil {
		return req, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", idToken))
	return req, nil
}

// GetServiceAccountToken gets a service account ID token for the destination URL
// from the Metadata API.
func GetServiceAccountToken(ctx context.Context, audienceURL string) (string, error) {
	// query the id_token with ?audience as the serviceURL
	tokenURL := fmt.Sprintf("/instance/service-accounts/default/identity?audience=%s", audienceURL)
	idToken, err := metadata.Get(tokenURL)
	if err != nil {
		return "", fmt.Errorf("metadata.Get: failed to query id_token: %+v", err)
	}
	return idToken, err
}
