package hr

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== Mock HTTP Client =====

// mockHTTPClient is a mock implementation of HTTPClient for testing
type mockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return nil, errors.New("DoFunc not implemented")
}

// Helper function to create a mock response
func mockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

// ===== MocoClient Tests =====

func TestNewMocoClient(t *testing.T) {
	client := NewMocoClient("test-domain", "test-token")
	assert.NotNil(t, client)
	assert.Equal(t, "test-domain", client.domain)
	assert.Equal(t, "test-token", client.token)
	assert.NotNil(t, client.httpClient)
}

func TestNewMocoClientWithHTTP(t *testing.T) {
	mockClient := &mockHTTPClient{}
	client := NewMocoClientWithHTTP("test-domain", "test-token", mockClient)
	assert.NotNil(t, client)
	assert.Equal(t, mockClient, client.httpClient)
}

func TestMocoClient_Projects_Success(t *testing.T) {
	mockProjects := []MocoProject{
		{
			Id:         1,
			Name:       "Project Alpha",
			Active:     true,
			Billable:   true,
			Identifier: "ALPHA",
			StartDate:  MocoDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		},
		{
			Id:         2,
			Name:       "Project Beta",
			Active:     true,
			Billable:   false,
			Identifier: "BETA",
		},
	}
	mockBody, _ := json.Marshal(mockProjects)

	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "https://test-domain.mocoapp.com/api/v1/projects.json", req.URL.String())
			assert.Equal(t, "Token token=test-token", req.Header.Get("Authorization"))
			assert.Equal(t, "application/json", req.Header.Get("Accept"))
			return mockResponse(http.StatusOK, string(mockBody)), nil
		},
	}

	client := NewMocoClientWithHTTP("test-domain", "test-token", mockClient)
	projects, err := client.Projects()

	require.NoError(t, err)
	assert.Len(t, projects, 2)
	assert.Equal(t, "Project Alpha", projects[0].Name)
	assert.Equal(t, "Project Beta", projects[1].Name)
}

func TestMocoClient_Projects_HTTPError(t *testing.T) {
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		},
	}

	client := NewMocoClientWithHTTP("test-domain", "test-token", mockClient)
	projects, err := client.Projects()

	assert.Error(t, err)
	assert.Nil(t, projects)
	assert.Contains(t, err.Error(), "request failed")
}

func TestMocoClient_Projects_UnauthorizedError(t *testing.T) {
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return mockResponse(http.StatusUnauthorized, `{"error":"Unauthorized"}`), nil
		},
	}

	client := NewMocoClientWithHTTP("test-domain", "test-token", mockClient)
	projects, err := client.Projects()

	assert.Error(t, err)
	assert.Nil(t, projects)
	assert.Contains(t, err.Error(), "401")
}

func TestMocoClient_Projects_InvalidJSON(t *testing.T) {
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return mockResponse(http.StatusOK, `{invalid json}`), nil
		},
	}

	client := NewMocoClientWithHTTP("test-domain", "test-token", mockClient)
	projects, err := client.Projects()

	assert.Error(t, err)
	assert.Nil(t, projects)
	assert.Contains(t, err.Error(), "failed to parse response")
}

func TestMocoClient_ProjectContracts_Success(t *testing.T) {
	mockContracts := []MocoUserGroup{
		{
			Id:        1,
			UserId:    100,
			FirstName: "John",
			LastName:  "Doe",
			Billable:  true,
			Active:    true,
		},
	}
	mockBody, _ := json.Marshal(mockContracts)

	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "https://test-domain.mocoapp.com/api/v1/projects/123/contracts.json", req.URL.String())
			return mockResponse(http.StatusOK, string(mockBody)), nil
		},
	}

	client := NewMocoClientWithHTTP("test-domain", "test-token", mockClient)
	contracts, err := client.ProjectContracts(123)

	require.NoError(t, err)
	assert.Len(t, contracts, 1)
	assert.Equal(t, "John", contracts[0].FirstName)
}

func TestMocoClient_Users_Success(t *testing.T) {
	mockUsers := []MocoUser{
		{
			Id:        1,
			FirstName: "Alice",
			LastName:  "Smith",
			Email:     "alice@example.com",
			Active:    true,
			Extern:    false,
		},
		{
			Id:        2,
			FirstName: "Bob",
			LastName:  "Jones",
			Email:     "bob@example.com",
			Active:    true,
			Extern:    true,
		},
	}
	mockBody, _ := json.Marshal(mockUsers)

	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "https://test-domain.mocoapp.com/api/v1/users.json", req.URL.String())
			return mockResponse(http.StatusOK, string(mockBody)), nil
		},
	}

	client := NewMocoClientWithHTTP("test-domain", "test-token", mockClient)
	users, err := client.Users()

	require.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "alice@example.com", users[0].Email)
	assert.True(t, users[1].Extern)
}

func TestMocoClient_UserEmployments_Success(t *testing.T) {
	mockEmployments := []MocoUserEmployment{
		{
			Id:                1,
			WeeklyTargetHours: 40.0,
			From:              MocoDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			To:                MocoDate(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		},
	}
	mockBody, _ := json.Marshal(mockEmployments)

	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "https://test-domain.mocoapp.com/api/v1/users/employments.json?user_id=456", req.URL.String())
			return mockResponse(http.StatusOK, string(mockBody)), nil
		},
	}

	client := NewMocoClientWithHTTP("test-domain", "test-token", mockClient)
	employments, err := client.UserEmployments(456)

	require.NoError(t, err)
	assert.Len(t, employments, 1)
	assert.Equal(t, 40.0, employments[0].WeeklyTargetHours)
}

func TestMocoClient_ProjectTasks_Success(t *testing.T) {
	mockTasks := []MocoTask{
		{
			Id:       1,
			Name:     "Development",
			Billable: true,
			Active:   true,
		},
		{
			Id:       2,
			Name:     "Testing",
			Billable: false,
			Active:   true,
		},
	}
	mockBody, _ := json.Marshal(mockTasks)

	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "https://test-domain.mocoapp.com/api/v1/projects/789/tasks.json", req.URL.String())
			return mockResponse(http.StatusOK, string(mockBody)), nil
		},
	}

	client := NewMocoClientWithHTTP("test-domain", "test-token", mockClient)
	tasks, err := client.ProjectTasks(789)

	require.NoError(t, err)
	assert.Len(t, tasks, 2)
	assert.Equal(t, "Development", tasks[0].Name)
	assert.True(t, tasks[0].Billable)
	assert.False(t, tasks[1].Billable)
}

func TestMocoClient_Activities_Success(t *testing.T) {
	mockActivities := []MocoActivity{
		{
			Id:          1,
			Date:        MocoDate(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)),
			Description: "Fixed bug #123",
			ProjectId:   10,
			TaskId:      20,
			Seconds:     3600,
			Billable:    true,
			Hours:       1.0,
		},
	}
	mockBody, _ := json.Marshal(mockActivities)

	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "https://test-domain.mocoapp.com/api/v1/activities.json?project_id=10&task_id=20", req.URL.String())
			return mockResponse(http.StatusOK, string(mockBody)), nil
		},
	}

	client := NewMocoClientWithHTTP("test-domain", "test-token", mockClient)
	activities, err := client.Activities(10, 20)

	require.NoError(t, err)
	assert.Len(t, activities, 1)
	assert.Equal(t, "Fixed bug #123", activities[0].Description)
	assert.Equal(t, 3600, activities[0].Seconds)
}

func TestMocoClient_BookActivity_Success(t *testing.T) {
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "POST", req.Method)
			assert.Equal(t, "https://test-domain.mocoapp.com/api/v1/activities", req.URL.String())
			assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

			// Read and verify request body
			bodyBytes, _ := io.ReadAll(req.Body)
			var reqData map[string]interface{}
			json.Unmarshal(bodyBytes, &reqData)
			assert.Equal(t, "2024-01-15", reqData["date"])
			assert.Equal(t, float64(10), reqData["project_id"])
			assert.Equal(t, float64(20), reqData["task_id"])
			assert.Equal(t, float64(3600), reqData["seconds"])
			assert.Equal(t, "Test activity", reqData["description"])

			return mockResponse(http.StatusCreated, `{"id":123}`), nil
		},
	}

	client := NewMocoClientWithHTTP("test-domain", "test-token", mockClient)
	err := client.BookActivity("2024-01-15", 10, 20, 3600, "Test activity")

	assert.NoError(t, err)
}

func TestMocoClient_BookActivity_InvalidDate(t *testing.T) {
	client := NewMocoClient("test-domain", "test-token")
	err := client.BookActivity("invalid-date", 10, 20, 3600, "Test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid date format")
}

func TestMocoClient_BookActivity_APIError(t *testing.T) {
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return mockResponse(http.StatusBadRequest, `{"error":"Invalid project"}`), nil
		},
	}

	client := NewMocoClientWithHTTP("test-domain", "test-token", mockClient)
	err := client.BookActivity("2024-01-15", 10, 20, 3600, "Test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestMocoClient_BookActivityImpersonate_Success(t *testing.T) {
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "POST", req.Method)
			assert.Equal(t, "100", req.Header.Get("X-IMPERSONATE-USER-ID"))

			bodyBytes, _ := io.ReadAll(req.Body)
			var reqData map[string]interface{}
			json.Unmarshal(bodyBytes, &reqData)
			// User ID is in header, not in body
			assert.Equal(t, "2024-01-15", reqData["date"])
			assert.Equal(t, float64(10), reqData["project_id"])

			return mockResponse(http.StatusCreated, `{"id":456}`), nil
		},
	}

	client := NewMocoClientWithHTTP("test-domain", "test-token", mockClient)
	err := client.BookActivityImpersonate("2024-01-15", 10, 20, 3600, "Test", 100)

	assert.NoError(t, err)
}

func TestMocoClient_DeleteActivity_Success(t *testing.T) {
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "DELETE", req.Method)
			assert.Equal(t, "https://test-domain.mocoapp.com/api/v1/activities/789", req.URL.String())
			return mockResponse(http.StatusNoContent, ""), nil
		},
	}

	client := NewMocoClientWithHTTP("test-domain", "test-token", mockClient)
	err := client.DeleteActivity(789)

	assert.NoError(t, err)
}

func TestMocoClient_DeleteActivity_NotFound(t *testing.T) {
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return mockResponse(http.StatusNotFound, `{"error":"Activity not found"}`), nil
		},
	}

	client := NewMocoClientWithHTTP("test-domain", "test-token", mockClient)
	err := client.DeleteActivity(999)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestMocoClient_AddProjectContract_Success(t *testing.T) {
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "POST", req.Method)
			assert.Equal(t, "https://test-domain.mocoapp.com/api/v1/projects/10/contracts", req.URL.String())

			bodyBytes, _ := io.ReadAll(req.Body)
			var reqData map[string]interface{}
			json.Unmarshal(bodyBytes, &reqData)
			assert.Equal(t, float64(100), reqData["user_id"])

			return mockResponse(http.StatusCreated, `{"id":1}`), nil
		},
	}

	client := NewMocoClientWithHTTP("test-domain", "test-token", mockClient)
	err := client.AddProjectContract(10, 100)

	assert.NoError(t, err)
}

// ===== Backward Compatibility Wrapper Tests =====

func TestMocoAppProjects_Wrapper(t *testing.T) {
	// This tests the backward compatibility wrapper
	// It will fail to connect but tests the wrapper logic
	projects := MocoAppProjects("test-domain", "test-token")
	// Should return nil on network error (logged internally)
	assert.Nil(t, projects)
}

func TestMocoAppUsers_Wrapper(t *testing.T) {
	users := MocoAppUsers("test-domain", "test-token")
	assert.Nil(t, users)
}

func TestMocoUserEmployments_Wrapper(t *testing.T) {
	employments := MocoUserEmployments("test-domain", "test-token", 123)
	assert.Nil(t, employments)
}

func TestMocoAppProjectsTasks_Wrapper(t *testing.T) {
	tasks := MocoAppProjectsTasks("test-domain", "test-token", 456)
	assert.Nil(t, tasks)
}

func TestMocoAppActivities_Wrapper(t *testing.T) {
	activities := MocoAppActivities("test-domain", "test-token", 10, 20)
	assert.Nil(t, activities)
}

func TestMocoAppProjectsContracts_Wrapper(t *testing.T) {
	contracts := MocoAppProjectsContracts("test-domain", "test-token", 789)
	assert.Nil(t, contracts)
}

// ===== PersonioClient Tests =====

func TestNewPersonioClient(t *testing.T) {
	client := NewPersonioClient("client-id", "client-secret")
	assert.NotNil(t, client)
	assert.Equal(t, "client-id", client.clientID)
	assert.Equal(t, "client-secret", client.clientSecret)
}

func TestNewPersonioClientWithHTTP(t *testing.T) {
	mockClient := &mockHTTPClient{}
	client := NewPersonioClientWithHTTP("client-id", "client-secret", "partner-id", "app-id", mockClient)
	assert.NotNil(t, client)
	assert.Equal(t, mockClient, client.httpClient)
	assert.Equal(t, "partner-id", client.partnerID)
	assert.Equal(t, "app-id", client.appID)
}

func TestPersonioClient_ObtainToken_Success(t *testing.T) {
	os.Setenv("PERSONIO_PARTNER_ID", "test-partner")
	os.Setenv("PERSONIO_APP_ID", "test-app")
	defer os.Unsetenv("PERSONIO_PARTNER_ID")
	defer os.Unsetenv("PERSONIO_APP_ID")

	mockToken := PersonioToken{
		AccessToken: "abc123token",
		ExpiresIn:   3600,
		TokenType:   "Bearer",
		Scope:       "read write",
	}
	mockBody, _ := json.Marshal(mockToken)

	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "POST", req.Method)
			assert.Equal(t, "https://api.personio.de/v2/auth/token", req.URL.String())
			assert.Equal(t, "test-partner", req.Header.Get("X-Personio-Partner-ID"))
			assert.Equal(t, "test-app", req.Header.Get("X-Personio-App-ID"))
			assert.Equal(t, "application/x-www-form-urlencoded", req.Header.Get("Content-Type"))

			// Verify request body - it's form data, not JSON
			bodyBytes, _ := io.ReadAll(req.Body)
			bodyStr := string(bodyBytes)
			assert.Contains(t, bodyStr, "grant_type=client_credentials")
			assert.Contains(t, bodyStr, "client_id=client-id-test")
			assert.Contains(t, bodyStr, "client_secret=client-secret-test")

			return mockResponse(http.StatusOK, string(mockBody)), nil
		},
	}

	client := NewPersonioClientWithHTTP("client-id-test", "client-secret-test", "test-partner", "test-app", mockClient)
	err := client.obtainToken()

	require.NoError(t, err)
	assert.NotNil(t, client.token)
	assert.Equal(t, "abc123token", client.token.AccessToken)
	assert.Equal(t, 3600, client.token.ExpiresIn)
}

func TestPersonioClient_ObtainToken_EmptyCredentials(t *testing.T) {
	// Test that empty partner/app IDs result in API error
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Empty header values
			assert.Equal(t, "", req.Header.Get("X-Personio-Partner-ID"))
			assert.Equal(t, "", req.Header.Get("X-Personio-App-ID"))
			return mockResponse(http.StatusBadRequest, `{"error":"invalid_client"}`), nil
		},
	}

	client := NewPersonioClientWithHTTP("client-id", "client-secret", "", "", mockClient)
	err := client.obtainToken()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestPersonioClient_ObtainToken_HTTPError(t *testing.T) {
	os.Setenv("PERSONIO_PARTNER_ID", "test-partner")
	os.Setenv("PERSONIO_APP_ID", "test-app")
	defer os.Unsetenv("PERSONIO_PARTNER_ID")
	defer os.Unsetenv("PERSONIO_APP_ID")

	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		},
	}

	client := NewPersonioClientWithHTTP("client-id", "client-secret", "test-partner", "test-app", mockClient)
	err := client.obtainToken()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")
}

func TestPersonioClient_ObtainToken_Unauthorized(t *testing.T) {
	os.Setenv("PERSONIO_PARTNER_ID", "test-partner")
	os.Setenv("PERSONIO_APP_ID", "test-app")
	defer os.Unsetenv("PERSONIO_PARTNER_ID")
	defer os.Unsetenv("PERSONIO_APP_ID")

	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return mockResponse(http.StatusUnauthorized, `{"error":"Invalid credentials"}`), nil
		},
	}

	client := NewPersonioClientWithHTTP("client-id", "client-secret", "test-partner", "test-app", mockClient)
	err := client.obtainToken()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestPersonioClient_GetPerson_Success(t *testing.T) {
	os.Setenv("PERSONIO_PARTNER_ID", "test-partner")
	os.Setenv("PERSONIO_APP_ID", "test-app")
	defer os.Unsetenv("PERSONIO_PARTNER_ID")
	defer os.Unsetenv("PERSONIO_APP_ID")

	mockPerson := PersonioPerson{
		Id:        "123",
		Email:     "john@example.com",
		FirstName: "John",
		LastName:  "Doe",
	}
	mockBody, _ := json.Marshal(mockPerson)

	callCount := 0
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				// First call: obtain token
				token := PersonioToken{AccessToken: "test-token", ExpiresIn: 3600}
				tokenBody, _ := json.Marshal(token)
				return mockResponse(http.StatusOK, string(tokenBody)), nil
			}
			// Second call: get person
			assert.Equal(t, "https://api.personio.de/v2/persons/123", req.URL.String())
			assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
			return mockResponse(http.StatusOK, string(mockBody)), nil
		},
	}

	client := NewPersonioClientWithHTTP("client-id", "client-secret", "test-partner", "test-app", mockClient)
	person, err := client.GetPerson("123")

	require.NoError(t, err)
	assert.Equal(t, "123", person.Id)
	assert.Equal(t, "john@example.com", person.Email)
	assert.Equal(t, "John", person.FirstName)
}

func TestPersonioClient_GetPerson_TokenError(t *testing.T) {
	os.Unsetenv("PERSONIO_PARTNER_ID")
	os.Unsetenv("PERSONIO_APP_ID")

	client := NewPersonioClient("client-id", "client-secret")
	person, err := client.GetPerson("123")

	assert.Error(t, err)
	assert.Empty(t, person.Id)
}

func TestPersonioClient_BookAttendance_Success(t *testing.T) {
	os.Setenv("PERSONIO_PARTNER_ID", "test-partner")
	os.Setenv("PERSONIO_APP_ID", "test-app")
	defer os.Unsetenv("PERSONIO_PARTNER_ID")
	defer os.Unsetenv("PERSONIO_APP_ID")

	callCount := 0
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				// First call: obtain token
				token := PersonioToken{AccessToken: "test-token", ExpiresIn: 3600}
				tokenBody, _ := json.Marshal(token)
				return mockResponse(http.StatusOK, string(tokenBody)), nil
			}
			// Second call: book attendance
			assert.Equal(t, "POST", req.Method)
			assert.Equal(t, "https://api.personio.de/v2/attendance-periods?skip_approval=false", req.URL.String())
			assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
			assert.Equal(t, "true", req.Header.Get("Beta"))

			// Verify request body - the payload is a string, not JSON array
			bodyBytes, _ := io.ReadAll(req.Body)
			assert.Contains(t, string(bodyBytes), `"type":"WORK"`)
			assert.Contains(t, string(bodyBytes), `"id":"123"`)
			assert.Contains(t, string(bodyBytes), `"date_time":"2024-01-15T09:00:00"`)
			assert.Contains(t, string(bodyBytes), `"date_time":"2024-01-15T17:00:00"`)

			return mockResponse(http.StatusCreated, `{"id":"456"}`), nil
		},
	}

	client := NewPersonioClientWithHTTP("client-id", "client-secret", "test-partner", "test-app", mockClient)
	err := client.BookAttendance("123", "2024-01-15T09:00:00", "2024-01-15T17:00:00")

	assert.NoError(t, err)
}

func TestPersonioClient_BookAttendance_APIError(t *testing.T) {
	os.Setenv("PERSONIO_PARTNER_ID", "test-partner")
	os.Setenv("PERSONIO_APP_ID", "test-app")
	defer os.Unsetenv("PERSONIO_PARTNER_ID")
	defer os.Unsetenv("PERSONIO_APP_ID")

	callCount := 0
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				// First call: obtain token
				token := PersonioToken{AccessToken: "test-token", ExpiresIn: 3600}
				tokenBody, _ := json.Marshal(token)
				return mockResponse(http.StatusOK, string(tokenBody)), nil
			}
			// Second call: API error
			return mockResponse(http.StatusBadRequest, `{"error":"Invalid employee"}`), nil
		},
	}

	client := NewPersonioClientWithHTTP("client-id", "client-secret", "test-partner", "test-app", mockClient)
	err := client.BookAttendance("999", "2024-01-15T09:00:00", "2024-01-15T17:00:00")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

// ===== Backward Compatibility Wrapper Tests for Personio =====

func TestPersonioBook_Wrapper(t *testing.T) {
	os.Setenv("PERSONIO_PARTNER_ID", "test-partner")
	os.Setenv("PERSONIO_APP_ID", "test-app")
	defer os.Unsetenv("PERSONIO_PARTNER_ID")
	defer os.Unsetenv("PERSONIO_APP_ID")

	// Will fail to connect but tests wrapper logic
	err := PersonioBook("client-id", "client-secret", "123", "2024-01-15T09:00:00", "2024-01-15T17:00:00")
	// Function swallows errors and returns nil
	assert.NoError(t, err)
}

// ===== Edge Cases and Error Handling =====

func TestMocoClient_EmptyDomain(t *testing.T) {
	client := NewMocoClient("", "test-token")
	_, err := client.Projects()
	// Will fail with URL error
	assert.Error(t, err)
}

func TestMocoClient_EmptyToken(t *testing.T) {
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Verify empty token is sent
			assert.Equal(t, "Token token=", req.Header.Get("Authorization"))
			return mockResponse(http.StatusUnauthorized, `{"error":"Missing token"}`), nil
		},
	}

	client := NewMocoClientWithHTTP("test-domain", "", mockClient)
	_, err := client.Projects()
	assert.Error(t, err)
}

func TestPersonioClient_EmptyCredentials(t *testing.T) {
	client := NewPersonioClient("", "")
	assert.NotNil(t, client)
	assert.Empty(t, client.clientID)
	assert.Empty(t, client.clientSecret)
}

// ===== Concurrent Access Tests =====

func TestMocoClient_ConcurrentProjects(t *testing.T) {
	mockProjects := []MocoProject{{Id: 1, Name: "Test"}}
	mockBody, _ := json.Marshal(mockProjects)

	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return mockResponse(http.StatusOK, string(mockBody)), nil
		},
	}

	client := NewMocoClientWithHTTP("test-domain", "test-token", mockClient)

	// Run 10 concurrent requests
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			projects, err := client.Projects()
			assert.NoError(t, err)
			assert.Len(t, projects, 1)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
