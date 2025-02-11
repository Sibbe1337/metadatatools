package integration

import (
	"encoding/json"
	"metadatatool/internal/domain"
	"metadatatool/internal/test/testutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthFlow(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Close()

	// Test user registration
	user := ts.CreateTestUser(t, "test@example.com", "password123", domain.RoleUser)
	require.NotEmpty(t, user.ID)

	// Test login
	loginReq := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}
	resp := ts.MakeRequest(http.MethodPost, "/api/v1/auth/login", loginReq, nil)
	require.Equal(t, http.StatusOK, resp.Code)

	// Extract session cookie
	cookies := resp.Result().Cookies()
	require.NotEmpty(t, cookies)

	// Test protected route access
	headers := map[string]string{
		"Cookie": cookies[0].String(),
	}
	resp = ts.MakeRequest(http.MethodGet, "/api/v1/tracks", nil, headers)
	require.Equal(t, http.StatusOK, resp.Code)

	// Test logout
	resp = ts.MakeRequest(http.MethodPost, "/api/v1/auth/logout", nil, headers)
	require.Equal(t, http.StatusOK, resp.Code)

	// Verify session is cleared
	resp = ts.MakeRequest(http.MethodGet, "/api/v1/tracks", nil, headers)
	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestAdminRoutes(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Close()

	// Create admin user
	admin := ts.CreateTestUser(t, "admin@example.com", "admin123", domain.RoleAdmin)
	require.NotEmpty(t, admin.ID)

	// Get admin token
	token := ts.GetAuthToken(t, "admin@example.com", "admin123")
	require.NotEmpty(t, token)

	// Test admin-only route access
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}
	resp := ts.MakeRequest(http.MethodGet, "/api/v1/admin/users", nil, headers)
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestRegularUserRoutes(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Close()

	// Create regular user
	user := ts.CreateTestUser(t, "user@example.com", "user123", domain.RoleUser)
	require.NotEmpty(t, user.ID)

	// Get user token
	token := ts.GetAuthToken(t, "user@example.com", "user123")
	require.NotEmpty(t, token)

	// Test regular user route access
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}
	resp := ts.MakeRequest(http.MethodGet, "/api/v1/tracks", nil, headers)
	require.Equal(t, http.StatusOK, resp.Code)

	// Test admin route access (should fail)
	resp = ts.MakeRequest(http.MethodGet, "/api/v1/admin/users", nil, headers)
	require.Equal(t, http.StatusForbidden, resp.Code)
}

func TestSessionManagement(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Close()

	// Create test user and get auth token
	user := ts.CreateTestUser(t, "test@example.com", "password123", domain.RoleUser)
	require.NotEmpty(t, user.ID)

	token := ts.GetAuthToken(t, "test@example.com", "password123")
	require.NotEmpty(t, token)

	t.Run("session operations", func(t *testing.T) {
		// 1. Get active sessions
		w := ts.MakeRequest(http.MethodGet, "/api/v1/protected/user", nil, map[string]string{
			"Authorization": "Bearer " + token,
		})
		require.Equal(t, http.StatusOK, w.Code)

		// 2. Create another session (login again)
		token2 := ts.GetAuthToken(t, "test@example.com", "password123")
		require.NotEmpty(t, token2)

		// 3. Verify both sessions work
		w = ts.MakeRequest(http.MethodGet, "/api/v1/protected/user", nil, map[string]string{
			"Authorization": "Bearer " + token,
		})
		require.Equal(t, http.StatusOK, w.Code)

		w = ts.MakeRequest(http.MethodGet, "/api/v1/protected/user", nil, map[string]string{
			"Authorization": "Bearer " + token2,
		})
		require.Equal(t, http.StatusOK, w.Code)

		// 4. Logout from first session
		w = ts.MakeRequest(http.MethodPost, "/api/v1/auth/logout", nil, map[string]string{
			"Authorization": "Bearer " + token,
		})
		require.Equal(t, http.StatusOK, w.Code)

		// 5. Verify first session is invalid but second still works
		w = ts.MakeRequest(http.MethodGet, "/api/v1/protected/user", nil, map[string]string{
			"Authorization": "Bearer " + token,
		})
		require.Equal(t, http.StatusUnauthorized, w.Code)

		w = ts.MakeRequest(http.MethodGet, "/api/v1/protected/user", nil, map[string]string{
			"Authorization": "Bearer " + token2,
		})
		require.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAPIKeyAuth(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Close()

	// Create test user with API key
	token := ts.GetAuthToken(t, "test@example.com", "password123")
	require.NotEmpty(t, token)

	t.Run("api key operations", func(t *testing.T) {
		// 1. Generate API key
		w := ts.MakeRequest(http.MethodPost, "/api/v1/auth/apikey", nil,
			map[string]string{"Authorization": "Bearer " + token})
		require.Equal(t, http.StatusOK, w.Code)
		var keyResult struct {
			Data struct {
				APIKey string `json:"api_key"`
			} `json:"data"`
		}
		require.NoError(t, json.NewDecoder(w.Body).Decode(&keyResult))
		assert.NotEmpty(t, keyResult.Data.APIKey)

		// 2. Access endpoint with API key
		w = ts.MakeRequest(http.MethodGet, "/api/v1/tracks", nil,
			map[string]string{"X-API-Key": keyResult.Data.APIKey})
		require.Equal(t, http.StatusOK, w.Code)

		// 3. Try invalid API key
		w = ts.MakeRequest(http.MethodGet, "/api/v1/tracks", nil,
			map[string]string{"X-API-Key": "invalid-key"})
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestRoleBasedAccess(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Close()

	// Create admin user
	adminUser := ts.CreateTestUser(t, "admin@example.com", "password123", domain.RoleAdmin)
	require.NotEmpty(t, adminUser.ID)

	// Get admin token
	adminToken := ts.GetAuthToken(t, "admin@example.com", "password123")
	require.NotEmpty(t, adminToken)

	// Create regular user
	userUser := ts.CreateTestUser(t, "user@example.com", "password123", domain.RoleUser)
	require.NotEmpty(t, userUser.ID)

	// Get regular user token
	userToken := ts.GetAuthToken(t, "user@example.com", "password123")
	require.NotEmpty(t, userToken)

	t.Run("role-based permissions", func(t *testing.T) {
		// 1. Admin can access admin endpoints
		w := ts.MakeRequest(http.MethodPost, "/api/v1/tracks/batch", nil,
			map[string]string{"Authorization": "Bearer " + adminToken})
		require.Equal(t, http.StatusOK, w.Code)

		// 2. Regular user cannot access admin endpoints
		w = ts.MakeRequest(http.MethodPost, "/api/v1/tracks/batch", nil,
			map[string]string{"Authorization": "Bearer " + userToken})
		require.Equal(t, http.StatusForbidden, w.Code)

		// 3. Both can access regular endpoints
		w = ts.MakeRequest(http.MethodGet, "/api/v1/tracks", nil,
			map[string]string{"Authorization": "Bearer " + adminToken})
		require.Equal(t, http.StatusOK, w.Code)

		w = ts.MakeRequest(http.MethodGet, "/api/v1/tracks", nil,
			map[string]string{"Authorization": "Bearer " + userToken})
		require.Equal(t, http.StatusOK, w.Code)
	})
}
