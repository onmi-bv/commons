package utils

import (
	"context"
	"fmt"
	"net/http"

	"cloud.google.com/go/compute/metadata"
)

// AddServiceAccountToken adds a the service account ID token to the request
// as obtained from the Metadata API.
func AddServiceAccountToken(ctx context.Context, req *http.Request, serviceURL string) (*http.Request, error) {
	// query the id_token with ?audience as the serviceURL
	tokenURL := fmt.Sprintf("/instance/service-accounts/default/identity?audience=%s", serviceURL)
	idToken, err := metadata.Get(tokenURL)
	if err != nil {
		return req, fmt.Errorf("metadata.Get: failed to query id_token: %+v", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", idToken))
	return req, nil
}
