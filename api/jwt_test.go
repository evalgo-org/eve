// Package api provides comprehensive testing for JWT authentication handlers.
// This file contains unit tests for token generation endpoints, validating
// request handling, error conditions, and proper JWT token creation.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	eve "eve.evalgo.org/common"
	"eve.evalgo.org/security"
)

// TestGenerateToken_Success tests successful JWT token generation with valid user ID.
func TestGenerateToken_Success(t *testing.T) {
	// Setup
	e := echo.New()
	jwtService := security.NewJWTService("test-secret-key")
	handlers := &Handlers{
		JWT: jwtService,
	}

	// Create request with valid user ID
	requestBody := `{"user_id":"user123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/token", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err := handlers.GenerateToken(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify response structure
	var response TokenResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotEmpty(t, response.Token, "Token should not be empty")

	// Verify token is valid
	token, err := jwtService.ValidateToken(response.Token)
	require.NoError(t, err, "Generated token should be valid")
	assert.Equal(t, "user123", token.Subject(), "Token subject should match user ID")
}

// TestGenerateToken_EmptyUserID tests token generation with empty user ID.
func TestGenerateToken_EmptyUserID(t *testing.T) {
	// Setup
	e := echo.New()
	jwtService := security.NewJWTService("test-secret-key")
	handlers := &Handlers{
		JWT: jwtService,
	}

	// Create request with empty user ID
	requestBody := `{"user_id":""}`
	req := httptest.NewRequest(http.MethodPost, "/auth/token", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err := handlers.GenerateToken(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify error message
	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "user_id is required", response["error"])
}

// TestGenerateToken_MissingUserID tests token generation with missing user_id field.
func TestGenerateToken_MissingUserID(t *testing.T) {
	// Setup
	e := echo.New()
	jwtService := security.NewJWTService("test-secret-key")
	handlers := &Handlers{
		JWT: jwtService,
	}

	// Create request without user_id field
	requestBody := `{}`
	req := httptest.NewRequest(http.MethodPost, "/auth/token", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err := handlers.GenerateToken(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify error message
	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "user_id is required", response["error"])
}

// TestGenerateToken_InvalidJSON tests token generation with malformed JSON.
func TestGenerateToken_InvalidJSON(t *testing.T) {
	// Setup
	e := echo.New()
	jwtService := security.NewJWTService("test-secret-key")
	handlers := &Handlers{
		JWT: jwtService,
	}

	// Create request with invalid JSON
	requestBody := `{"user_id":"user123"`
	req := httptest.NewRequest(http.MethodPost, "/auth/token", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err := handlers.GenerateToken(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify error message
	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Invalid request", response["error"])
}

// TestGenerateToken_SpecialCharactersUserID tests token generation with special characters in user ID.
func TestGenerateToken_SpecialCharactersUserID(t *testing.T) {
	// Setup
	e := echo.New()
	jwtService := security.NewJWTService("test-secret-key")
	handlers := &Handlers{
		JWT: jwtService,
	}

	testCases := []struct {
		name   string
		userID string
	}{
		{"Email format", "user@example.com"},
		{"UUID format", "550e8400-e29b-41d4-a716-446655440000"},
		{"With spaces", "user 123"},
		{"With special chars", "user!@#$%^&*()"},
		{"Unicode characters", "用户123"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create request
			requestBody := `{"user_id":"` + tc.userID + `"}`
			req := httptest.NewRequest(http.MethodPost, "/auth/token", strings.NewReader(requestBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Execute
			err := handlers.GenerateToken(c)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)

			// Verify token contains correct user ID
			var response TokenResponse
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			require.NoError(t, err)

			token, err := jwtService.ValidateToken(response.Token)
			require.NoError(t, err)
			assert.Equal(t, tc.userID, token.Subject())
		})
	}
}

// TestGenerateToken_TokenExpiration tests that generated tokens have proper expiration.
func TestGenerateToken_TokenExpiration(t *testing.T) {
	// Setup
	e := echo.New()
	jwtService := security.NewJWTService("test-secret-key")
	handlers := &Handlers{
		JWT: jwtService,
	}

	// Create request
	requestBody := `{"user_id":"user123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/token", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Record time before generation
	beforeGeneration := time.Now()

	// Execute
	err := handlers.GenerateToken(c)

	// Record time after generation
	afterGeneration := time.Now()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify token expiration
	var response TokenResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	token, err := jwtService.ValidateToken(response.Token)
	require.NoError(t, err)

	// Verify expiration is approximately 24 hours from now
	expectedExpiration := beforeGeneration.Add(24 * time.Hour)
	actualExpiration := token.Expiration()

	// Allow 5 second variance for test execution time
	timeDiff := actualExpiration.Sub(expectedExpiration).Abs()
	assert.True(t, timeDiff < 5*time.Second, "Token expiration should be ~24 hours from generation")

	// Verify token is not already expired
	assert.True(t, actualExpiration.After(afterGeneration), "Token should not be expired immediately after generation")
}

// TestGenerateToken_TokenClaims tests that generated tokens have all required claims.
func TestGenerateToken_TokenClaims(t *testing.T) {
	// Setup
	e := echo.New()
	jwtService := security.NewJWTService("test-secret-key")
	handlers := &Handlers{
		JWT: jwtService,
	}

	// Create request
	requestBody := `{"user_id":"user123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/token", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	beforeGeneration := time.Now()

	// Execute
	err := handlers.GenerateToken(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Parse token and verify claims
	var response TokenResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	token, err := jwtService.ValidateToken(response.Token)
	require.NoError(t, err)

	// Verify Subject claim
	assert.Equal(t, "user123", token.Subject(), "Token should have correct subject")

	// Verify IssuedAt claim
	issuedAt := token.IssuedAt()
	assert.False(t, issuedAt.IsZero(), "Token should have issued-at time")
	assert.True(t, issuedAt.After(beforeGeneration.Add(-1*time.Second)), "Issued-at should be recent")

	// Verify Expiration claim
	expiration := token.Expiration()
	assert.False(t, expiration.IsZero(), "Token should have expiration time")
	assert.True(t, expiration.After(issuedAt), "Expiration should be after issued-at")
}

// TestGenerateToken_MultipleRequests tests that multiple token generation requests work correctly.
func TestGenerateToken_MultipleRequests(t *testing.T) {
	// Setup
	e := echo.New()
	jwtService := security.NewJWTService("test-secret-key")
	handlers := &Handlers{
		JWT: jwtService,
	}

	tokens := make([]string, 3)

	// Generate multiple tokens for the same user
	for i := 0; i < 3; i++ {
		requestBody := `{"user_id":"user123"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/token", strings.NewReader(requestBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handlers.GenerateToken(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var response TokenResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		tokens[i] = response.Token

		// Delay to ensure different IssuedAt times (JWT uses second precision)
		time.Sleep(1100 * time.Millisecond)
	}

	// Verify all tokens are different (due to different IssuedAt times)
	for i := 0; i < len(tokens); i++ {
		for j := i + 1; j < len(tokens); j++ {
			assert.NotEqual(t, tokens[i], tokens[j], "Each token should be unique")
		}
	}

	// Verify all tokens are valid
	for i, tokenStr := range tokens {
		token, err := jwtService.ValidateToken(tokenStr)
		require.NoError(t, err, "Token %d should be valid", i)
		assert.Equal(t, "user123", token.Subject())
	}
}

// TestGenerateToken_DifferentSecrets tests that tokens from different secrets cannot be validated.
func TestGenerateToken_DifferentSecrets(t *testing.T) {
	// Setup with first secret
	e := echo.New()
	jwtService1 := security.NewJWTService("secret-1")
	handlers := &Handlers{
		JWT: jwtService1,
	}

	// Generate token with first secret
	requestBody := `{"user_id":"user123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/token", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handlers.GenerateToken(c)
	require.NoError(t, err)

	var response TokenResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify token is valid with original secret
	_, err = jwtService1.ValidateToken(response.Token)
	assert.NoError(t, err, "Token should be valid with original secret")

	// Verify token is invalid with different secret
	jwtService2 := security.NewJWTService("secret-2")
	_, err = jwtService2.ValidateToken(response.Token)
	assert.Error(t, err, "Token should be invalid with different secret")
}

// TestGenerateToken_LongUserID tests token generation with very long user IDs.
func TestGenerateToken_LongUserID(t *testing.T) {
	// Setup
	e := echo.New()
	jwtService := security.NewJWTService("test-secret-key")
	handlers := &Handlers{
		JWT: jwtService,
	}

	// Create very long user ID (1000 characters)
	longUserID := strings.Repeat("a", 1000)
	requestBody := `{"user_id":"` + longUserID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/token", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err := handlers.GenerateToken(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify token is valid and contains correct user ID
	var response TokenResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	token, err := jwtService.ValidateToken(response.Token)
	require.NoError(t, err)
	assert.Equal(t, longUserID, token.Subject())
}

// TestTokenRequest_Validation tests the TokenRequest struct binding.
func TestTokenRequest_Validation(t *testing.T) {
	testCases := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Valid request",
			requestBody:    `{"user_id":"user123"}`,
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "Empty user_id",
			requestBody:    `{"user_id":""}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "user_id is required",
		},
		{
			name:           "Null user_id",
			requestBody:    `{"user_id":null}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "user_id is required",
		},
		{
			name:           "Extra fields ignored",
			requestBody:    `{"user_id":"user123","extra":"field"}`,
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "Whitespace only user_id",
			requestBody:    `{"user_id":"   "}`,
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			jwtService := security.NewJWTService("test-secret-key")
			handlers := &Handlers{
				JWT: jwtService,
			}

			req := httptest.NewRequest(http.MethodPost, "/auth/token", strings.NewReader(tc.requestBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Execute
			err := handlers.GenerateToken(c)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, rec.Code)

			if tc.expectedError != "" {
				var response map[string]string
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tc.expectedError, response["error"])
			}
		})
	}
}

// TestTokenResponse_Structure tests the TokenResponse struct structure.
func TestTokenResponse_Structure(t *testing.T) {
	// Setup
	e := echo.New()
	jwtService := security.NewJWTService("test-secret-key")
	handlers := &Handlers{
		JWT: jwtService,
	}

	requestBody := `{"user_id":"user123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/token", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err := handlers.GenerateToken(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify response structure
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Should only have "token" field
	assert.Len(t, response, 1, "Response should have exactly one field")
	token, exists := response["token"]
	assert.True(t, exists, "Response should have 'token' field")
	assert.IsType(t, "", token, "Token field should be a string")
	assert.NotEmpty(t, token, "Token should not be empty")
}

// BenchmarkGenerateToken benchmarks the token generation handler.
func BenchmarkGenerateToken(b *testing.B) {
	// Setup
	e := echo.New()
	jwtService := security.NewJWTService("test-secret-key")
	handlers := &Handlers{
		JWT: jwtService,
	}

	requestBody := `{"user_id":"user123"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/auth/token", strings.NewReader(requestBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handlers.GenerateToken(c)
	}
}

// Mock implementations for testing handlers

// MockMessagePublisher is a mock message publisher for testing
type MockMessagePublisher struct {
	PublishMessageFunc func(eve.FlowProcessMessage) error
	CloseFunc          func() error
}

func (m *MockMessagePublisher) PublishMessage(msg eve.FlowProcessMessage) error {
	if m.PublishMessageFunc != nil {
		return m.PublishMessageFunc(msg)
	}
	return nil
}

func (m *MockMessagePublisher) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// MockDocumentStore is a mock document store for testing
type MockDocumentStore struct {
	GetDocumentFunc         func(string) (*eve.FlowProcessDocument, error)
	GetAllDocumentsFunc     func() ([]eve.FlowProcessDocument, error)
	GetDocumentsByStateFunc func(eve.FlowProcessState) ([]eve.FlowProcessDocument, error)
	SaveDocumentFunc        func(eve.FlowProcessDocument) (*eve.FlowCouchDBResponse, error)
	DeleteDocumentFunc      func(string, string) error
	CloseFunc               func() error
}

func (m *MockDocumentStore) GetDocument(id string) (*eve.FlowProcessDocument, error) {
	if m.GetDocumentFunc != nil {
		return m.GetDocumentFunc(id)
	}
	return nil, nil
}

func (m *MockDocumentStore) GetAllDocuments() ([]eve.FlowProcessDocument, error) {
	if m.GetAllDocumentsFunc != nil {
		return m.GetAllDocumentsFunc()
	}
	return []eve.FlowProcessDocument{}, nil
}

func (m *MockDocumentStore) GetDocumentsByState(state eve.FlowProcessState) ([]eve.FlowProcessDocument, error) {
	if m.GetDocumentsByStateFunc != nil {
		return m.GetDocumentsByStateFunc(state)
	}
	return []eve.FlowProcessDocument{}, nil
}

func (m *MockDocumentStore) SaveDocument(doc eve.FlowProcessDocument) (*eve.FlowCouchDBResponse, error) {
	if m.SaveDocumentFunc != nil {
		return m.SaveDocumentFunc(doc)
	}
	return &eve.FlowCouchDBResponse{}, nil
}

func (m *MockDocumentStore) DeleteDocument(id, rev string) error {
	if m.DeleteDocumentFunc != nil {
		return m.DeleteDocumentFunc(id, rev)
	}
	return nil
}

func (m *MockDocumentStore) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// TestPublishMessage_Success tests successful message publishing
func TestPublishMessage_Success(t *testing.T) {
	e := echo.New()
	mockPublisher := &MockMessagePublisher{}
	handlers := &Handlers{
		RabbitMQ: mockPublisher,
	}

	requestBody := `{
		"process_id": "proc-123",
		"state": "running",
		"description": "Test process"
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/api/publish", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handlers.PublishMessage(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.Equal(t, "Message published successfully", response["status"])
}

// TestPublishMessage_MissingProcessID tests publishing without process ID
func TestPublishMessage_MissingProcessID(t *testing.T) {
	e := echo.New()
	mockPublisher := &MockMessagePublisher{}
	handlers := &Handlers{
		RabbitMQ: mockPublisher,
	}

	requestBody := `{
		"state": "running",
		"description": "Test"
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/api/publish", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handlers.PublishMessage(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.Equal(t, "process_id is required", response["error"])
}

// TestPublishMessage_MissingState tests publishing without state
func TestPublishMessage_MissingState(t *testing.T) {
	e := echo.New()
	mockPublisher := &MockMessagePublisher{}
	handlers := &Handlers{
		RabbitMQ: mockPublisher,
	}

	requestBody := `{
		"process_id": "proc-123",
		"description": "Test"
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/api/publish", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handlers.PublishMessage(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.Equal(t, "state is required", response["error"])
}

// TestPublishMessage_InvalidJSON tests publishing with malformed JSON
func TestPublishMessage_InvalidJSON(t *testing.T) {
	e := echo.New()
	mockPublisher := &MockMessagePublisher{}
	handlers := &Handlers{
		RabbitMQ: mockPublisher,
	}

	requestBody := `{"process_id": "invalid json`

	req := httptest.NewRequest(http.MethodPost, "/v1/api/publish", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handlers.PublishMessage(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// TestPublishMessage_PublishError tests error handling when publishing fails
func TestPublishMessage_PublishError(t *testing.T) {
	e := echo.New()
	mockPublisher := &MockMessagePublisher{
		PublishMessageFunc: func(msg eve.FlowProcessMessage) error {
			return assert.AnError
		},
	}
	handlers := &Handlers{
		RabbitMQ: mockPublisher,
	}

	requestBody := `{
		"process_id": "proc-123",
		"state": "running"
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/api/publish", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handlers.PublishMessage(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.Equal(t, "Failed to publish message", response["error"])
}

// TestGetProcess_Success tests successful process retrieval
func TestGetProcess_Success(t *testing.T) {
	e := echo.New()
	mockStore := &MockDocumentStore{
		GetDocumentFunc: func(id string) (*eve.FlowProcessDocument, error) {
			return &eve.FlowProcessDocument{
				ID:        id,
				ProcessID: id,
				State:     eve.StateRunning,
			}, nil
		},
	}
	handlers := &Handlers{
		CouchDB: mockStore,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/api/processes/proc-123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("proc-123")

	err := handlers.GetProcess(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestGetProcess_NotFound tests process not found scenario
func TestGetProcess_NotFound(t *testing.T) {
	e := echo.New()
	mockStore := &MockDocumentStore{
		GetDocumentFunc: func(id string) (*eve.FlowProcessDocument, error) {
			return nil, fmt.Errorf("document not found")
		},
	}
	handlers := &Handlers{
		CouchDB: mockStore,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/api/processes/nonexistent", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("nonexistent")

	err := handlers.GetProcess(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var response map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.Equal(t, "Process not found", response["error"])
}

// TestGetProcess_EmptyID tests process retrieval with empty ID
func TestGetProcess_EmptyID(t *testing.T) {
	e := echo.New()
	mockStore := &MockDocumentStore{}
	handlers := &Handlers{
		CouchDB: mockStore,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/api/processes/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("")

	err := handlers.GetProcess(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// TestGetProcess_DatabaseError tests database error handling
func TestGetProcess_DatabaseError(t *testing.T) {
	e := echo.New()
	mockStore := &MockDocumentStore{
		GetDocumentFunc: func(id string) (*eve.FlowProcessDocument, error) {
			return nil, fmt.Errorf("database connection error")
		},
	}
	handlers := &Handlers{
		CouchDB: mockStore,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/api/processes/proc-123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("proc-123")

	err := handlers.GetProcess(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.Equal(t, "Failed to retrieve process", response["error"])
}

// TestGetProcessesByState_AllProcesses tests getting all processes without filter
func TestGetProcessesByState_AllProcesses(t *testing.T) {
	e := echo.New()
	mockStore := &MockDocumentStore{
		GetAllDocumentsFunc: func() ([]eve.FlowProcessDocument, error) {
			return []eve.FlowProcessDocument{
				{ProcessID: "proc-1", State: eve.StateRunning},
				{ProcessID: "proc-2", State: eve.StateSuccessful},
			}, nil
		},
	}
	handlers := &Handlers{
		CouchDB: mockStore,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/api/processes", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handlers.GetProcessesByState(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.Equal(t, float64(2), response["count"])
}

// TestGetProcessesByState_FilterByState tests filtering processes by state
func TestGetProcessesByState_FilterByState(t *testing.T) {
	e := echo.New()
	mockStore := &MockDocumentStore{
		GetDocumentsByStateFunc: func(state eve.FlowProcessState) ([]eve.FlowProcessDocument, error) {
			return []eve.FlowProcessDocument{
				{ProcessID: "proc-1", State: state},
			}, nil
		},
	}
	handlers := &Handlers{
		CouchDB: mockStore,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/api/processes?state=running", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handlers.GetProcessesByState(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.Equal(t, float64(1), response["count"])
}

// TestGetProcessesByState_InvalidState tests filtering with invalid state
func TestGetProcessesByState_InvalidState(t *testing.T) {
	e := echo.New()
	mockStore := &MockDocumentStore{}
	handlers := &Handlers{
		CouchDB: mockStore,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/api/processes?state=invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handlers.GetProcessesByState(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.Equal(t, "Invalid state value", response["error"])
}

// TestGetProcessesByState_DatabaseError tests database error when getting all docs
func TestGetProcessesByState_DatabaseError(t *testing.T) {
	e := echo.New()
	mockStore := &MockDocumentStore{
		GetAllDocumentsFunc: func() ([]eve.FlowProcessDocument, error) {
			return nil, fmt.Errorf("database error")
		},
	}
	handlers := &Handlers{
		CouchDB: mockStore,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/api/processes", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handlers.GetProcessesByState(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// TestGetProcessesByState_FilteredDatabaseError tests database error when filtering
func TestGetProcessesByState_FilteredDatabaseError(t *testing.T) {
	e := echo.New()
	mockStore := &MockDocumentStore{
		GetDocumentsByStateFunc: func(state eve.FlowProcessState) ([]eve.FlowProcessDocument, error) {
			return nil, fmt.Errorf("database error")
		},
	}
	handlers := &Handlers{
		CouchDB: mockStore,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/api/processes?state=running", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handlers.GetProcessesByState(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
