package test

import (
	"fmt"
	"metadatatool/internal/domain"
	"metadatatool/internal/test/testutil"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// CreateTestUser creates a new user for testing
func CreateTestUser(t *testing.T, ts *testutil.TestServer, email, password string, role domain.Role) *domain.User {
	user := ts.CreateTestUser(t, email, password, role)
	require.NotEmpty(t, user.ID)
	return user
}

// GetAuthToken gets an authentication token for a user
func GetAuthToken(t *testing.T, ts *testutil.TestServer, email, password string) string {
	token := ts.GetAuthToken(t, email, password)
	require.NotEmpty(t, token)
	return token
}

// MakeRequest makes an HTTP request to the test server
func MakeRequest(t *testing.T, ts *testutil.TestServer, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	return ts.MakeRequest(method, path, body, headers)
}

// GetAuthHeader returns the Authorization header with the token
func GetAuthHeader(token string) map[string]string {
	return map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
	}
}
