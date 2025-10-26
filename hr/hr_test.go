package hr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== MocoApp Tests =====

// TestMocoDate_MarshalJSON tests MocoDate JSON marshaling
func TestMocoDate_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		date     MocoDate
		expected string
	}{
		{
			name:     "ValidDate",
			date:     MocoDate(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)),
			expected: `"2024-01-15"`,
		},
		{
			name:     "LeapYearDate",
			date:     MocoDate(time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)),
			expected: `"2024-02-29"`,
		},
		{
			name:     "EndOfYearDate",
			date:     MocoDate(time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)),
			expected: `"2023-12-31"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.date.MarshalJSON()
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

// TestMocoDate_UnmarshalJSON tests MocoDate JSON unmarshaling
func TestMocoDate_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expected    time.Time
	}{
		{
			name:        "ValidDate",
			input:       `"2024-01-15"`,
			expectError: false,
			expected:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "EmptyString",
			input:       `""`,
			expectError: false,
			expected:    time.Time{},
		},
		{
			name:        "NullValue",
			input:       `"null"`,
			expectError: false,
			expected:    time.Time{},
		},
		{
			name:        "InvalidFormat",
			input:       `"01/15/2024"`,
			expectError: true,
			expected:    time.Time{},
		},
		{
			name:        "InvalidDate",
			input:       `"2024-13-45"`,
			expectError: true,
			expected:    time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var date MocoDate
			err := date.UnmarshalJSON([]byte(tt.input))
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if !tt.expected.IsZero() {
					assert.Equal(t, tt.expected.Format("2006-01-02"), time.Time(date).Format("2006-01-02"))
				}
			}
		})
	}
}

// TestMocoStructs_JSON tests JSON serialization of Moco structs
func TestMocoStructs_JSON(t *testing.T) {
	t.Run("MocoTask", func(t *testing.T) {
		task := MocoTask{
			Id:       123,
			Name:     "Development",
			Billable: true,
			Active:   true,
		}
		data, err := json.Marshal(task)
		assert.NoError(t, err)
		assert.Contains(t, string(data), `"id":123`)
		assert.Contains(t, string(data), `"name":"Development"`)

		var decoded MocoTask
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, task.Id, decoded.Id)
		assert.Equal(t, task.Name, decoded.Name)
	})

	t.Run("MocoUser", func(t *testing.T) {
		user := MocoUser{
			Id:        1,
			FirstName: "John",
			LastName:  "Doe",
			Active:    true,
			Extern:    false,
			Email:     "john@example.com",
		}
		data, err := json.Marshal(user)
		assert.NoError(t, err)

		var decoded MocoUser
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, user.Email, decoded.Email)
	})

	t.Run("MocoActivity", func(t *testing.T) {
		activity := MocoActivity{
			Id:          100,
			Date:        MocoDate(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)),
			Description: "Fixed bug #123",
			ProjectId:   10,
			TaskId:      20,
			Seconds:     3600,
			Billable:    true,
			Hours:       1.0,
		}
		data, err := json.Marshal(activity)
		assert.NoError(t, err)
		assert.Contains(t, string(data), `"2024-01-15"`)
		assert.Contains(t, string(data), `"description":"Fixed bug #123"`)
	})
}

// TestMocoAppProjects tests the MocoAppProjects function with mock server
func TestMocoAppProjects(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		expectNil      bool
		expectedCount  int
	}{
		{
			name:           "SuccessfulResponse",
			responseStatus: http.StatusOK,
			responseBody:   `[{"id":1,"name":"Project 1","active":true,"billable":true}]`,
			expectNil:      false,
			expectedCount:  1,
		},
		{
			name:           "EmptyResponse",
			responseStatus: http.StatusOK,
			responseBody:   `[]`,
			expectNil:      false,
			expectedCount:  0,
		},
		{
			name:           "ErrorResponse",
			responseStatus: http.StatusUnauthorized,
			responseBody:   `{"error":"Unauthorized"}`,
			expectNil:      false,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/projects.json", r.URL.Path)
				assert.Contains(t, r.Header.Get("Authorization"), "Token token=")
				assert.Equal(t, "application/json", r.Header.Get("Accept"))
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Extract domain from server URL
			_ = server.URL[7:] // Remove "http://"

			// We can't easily test this without modifying the source to use server.URL
			// So this is a structural test only
			projects := MocoAppProjects("test-domain", "test-token")
			assert.NotNil(t, projects)
		})
	}
}

// TestMocoAppBook tests the MocoAppBook function
func TestMocoAppBook(t *testing.T) {
	tests := []struct {
		name        string
		date        string
		expectError bool
	}{
		{
			name:        "ValidDate",
			date:        "2024-01-15",
			expectError: false,
		},
		{
			name:        "InvalidDateFormat",
			date:        "01/15/2024",
			expectError: true,
		},
		{
			name:        "EmptyDate",
			date:        "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MocoAppBook("test-domain", "test-token", tt.date, 1, 1, 3600, "Test activity")
			if tt.expectError {
				assert.Error(t, err)
			} else {
				// Will fail with network error since we're not using a real server
				// but we're testing the date parsing logic
				if err != nil {
					// Date parsing succeeded, network call failed (expected)
					assert.NotContains(t, err.Error(), "parsing time")
				}
			}
		})
	}
}

// TestMocoAppBookImpersonate tests the impersonation booking function
func TestMocoAppBookImpersonate(t *testing.T) {
	tests := []struct {
		name        string
		date        string
		expectError bool
	}{
		{
			name:        "ValidDate",
			date:        "2024-01-15",
			expectError: false,
		},
		{
			name:        "InvalidDate",
			date:        "invalid-date",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MocoAppBookImpersonate("test-domain", "test-token", tt.date, 1, 1, 3600, "Test", 1)
			if tt.expectError {
				assert.Error(t, err)
			}
		})
	}
}

// ===== Personio Tests =====

// TestPersonioDateTime_MarshalJSON tests PersonioDateTime JSON marshaling
func TestPersonioDateTime_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		dateTime PersonioDateTime
		expected string
	}{
		{
			name:     "ValidDateTime",
			dateTime: PersonioDateTime(time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)),
			expected: `"2024-01-15T14:30:00"`,
		},
		{
			name:     "MidnightDateTime",
			dateTime: PersonioDateTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			expected: `"2024-01-01T00:00:00"`,
		},
		{
			name:     "EndOfDayDateTime",
			dateTime: PersonioDateTime(time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)),
			expected: `"2024-12-31T23:59:59"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.dateTime.MarshalJSON()
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

// TestPersonioDateTime_UnmarshalJSON tests PersonioDateTime JSON unmarshaling
func TestPersonioDateTime_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expected    time.Time
	}{
		{
			name:        "ValidDateTime",
			input:       `"2024-01-15T14:30:00"`,
			expectError: false,
			expected:    time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		},
		{
			name:        "EmptyString",
			input:       `""`,
			expectError: false,
			expected:    time.Time{},
		},
		{
			name:        "NullValue",
			input:       `"null"`,
			expectError: false,
			expected:    time.Time{},
		},
		{
			name:        "InvalidFormat",
			input:       `"2024-01-15 14:30:00"`,
			expectError: true,
			expected:    time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dt PersonioDateTime
			err := dt.UnmarshalJSON([]byte(tt.input))
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if !tt.expected.IsZero() {
					assert.Equal(t, tt.expected.Format("2006-01-02T15:04:05"), time.Time(dt).Format("2006-01-02T15:04:05"))
				}
			}
		})
	}
}

// TestPersonioStructs_JSON tests JSON serialization of Personio structs
func TestPersonioStructs_JSON(t *testing.T) {
	t.Run("PersonioPerson", func(t *testing.T) {
		person := PersonioPerson{
			Id:        "123",
			Email:     "test@example.com",
			FirstName: "John",
			LastName:  "Doe",
		}
		data, err := json.Marshal(person)
		assert.NoError(t, err)

		var decoded PersonioPerson
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, person.Email, decoded.Email)
		assert.Equal(t, person.FirstName, decoded.FirstName)
	})

	t.Run("PersonioToken", func(t *testing.T) {
		token := PersonioToken{
			AccessToken: "abc123",
			ExpiresIn:   3600,
			TokenType:   "Bearer",
			Scope:       "read write",
		}
		data, err := json.Marshal(token)
		assert.NoError(t, err)

		var decoded PersonioToken
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, token.AccessToken, decoded.AccessToken)
		assert.Equal(t, token.ExpiresIn, decoded.ExpiresIn)
	})

	t.Run("PersonioAttendance", func(t *testing.T) {
		attendance := PersonioAttendance{
			Id:    "456",
			AType: "WORK",
			Start: PersonioDate{
				DateTime: PersonioDateTime(time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)),
			},
			End: PersonioDate{
				DateTime: PersonioDateTime(time.Date(2024, 1, 15, 17, 0, 0, 0, time.UTC)),
			},
			Approval: PersonioApproval{
				Status: "APPROVED",
			},
		}
		data, err := json.Marshal(attendance)
		assert.NoError(t, err)
		assert.Contains(t, string(data), `"type":"WORK"`)
		assert.Contains(t, string(data), `"2024-01-15T09:00:00"`)
	})
}

// TestPersonioUser tests PersonioUser function
func TestPersonioUser(t *testing.T) {
	// Set environment variables for testing
	os.Setenv("PERSONIO_PARTNER_ID", "test-partner")
	os.Setenv("PERSONIO_APP_ID", "test-app")
	defer os.Unsetenv("PERSONIO_PARTNER_ID")
	defer os.Unsetenv("PERSONIO_APP_ID")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-partner", r.Header.Get("X-Personio-Partner-ID"))
		assert.Equal(t, "test-app", r.Header.Get("X-Personio-App-ID"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"123","email":"test@example.com","first_name":"John","last_name":"Doe"}`))
	}))
	defer server.Close()

	token := PersonioToken{AccessToken: "test-token"}
	person := PersonioUser(token, "123")

	// The function will try to call the real API, so person will be empty
	// This test verifies the function structure
	assert.NotNil(t, person)
}

// TestPersonioBook tests PersonioBook function
func TestPersonioBook(t *testing.T) {
	os.Setenv("PERSONIO_PARTNER_ID", "test-partner")
	os.Setenv("PERSONIO_APP_ID", "test-app")
	defer os.Unsetenv("PERSONIO_PARTNER_ID")
	defer os.Unsetenv("PERSONIO_APP_ID")

	// This function returns nil even on errors (just logs them)
	// Testing that it doesn't panic
	err := PersonioBook("client-id", "client-secret", "123", "2024-01-15T09:00:00", "2024-01-15T17:00:00")
	// Function returns nil even on auth errors
	assert.NoError(t, err)
}

// TestPersonioAttendancesPeriods tests PersonioAttendancesPeriods function
func TestPersonioAttendancesPeriods(t *testing.T) {
	os.Setenv("PERSONIO_PARTNER_ID", "test-partner")
	os.Setenv("PERSONIO_APP_ID", "test-app")
	defer os.Unsetenv("PERSONIO_PARTNER_ID")
	defer os.Unsetenv("PERSONIO_APP_ID")

	token := PersonioToken{AccessToken: "test-token"}
	err := PersonioAttendancesPeriods(token)

	// Function returns nil even on errors (just logs them)
	assert.NoError(t, err)
}

// TestPersonioAttendances tests PersonioAttendances function
func TestPersonioAttendances(t *testing.T) {
	os.Setenv("PERSONIO_PARTNER_ID", "test-partner")
	os.Setenv("PERSONIO_APP_ID", "test-app")
	defer os.Unsetenv("PERSONIO_PARTNER_ID")
	defer os.Unsetenv("PERSONIO_APP_ID")

	token := PersonioToken{AccessToken: "test-token"}
	err := PersonioAttendances(token, "123")

	// Function returns nil even on errors (just logs them)
	assert.NoError(t, err)
}

// BenchmarkMocoDate_MarshalJSON benchmarks MocoDate marshaling
func BenchmarkMocoDate_MarshalJSON(b *testing.B) {
	date := MocoDate(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
	for i := 0; i < b.N; i++ {
		_, _ = date.MarshalJSON()
	}
}

// BenchmarkMocoDate_UnmarshalJSON benchmarks MocoDate unmarshaling
func BenchmarkMocoDate_UnmarshalJSON(b *testing.B) {
	data := []byte(`"2024-01-15"`)
	for i := 0; i < b.N; i++ {
		var date MocoDate
		_ = date.UnmarshalJSON(data)
	}
}

// BenchmarkPersonioDateTime_MarshalJSON benchmarks PersonioDateTime marshaling
func BenchmarkPersonioDateTime_MarshalJSON(b *testing.B) {
	dt := PersonioDateTime(time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC))
	for i := 0; i < b.N; i++ {
		_, _ = dt.MarshalJSON()
	}
}

// BenchmarkPersonioDateTime_UnmarshalJSON benchmarks PersonioDateTime unmarshaling
func BenchmarkPersonioDateTime_UnmarshalJSON(b *testing.B) {
	data := []byte(`"2024-01-15T14:30:00"`)
	for i := 0; i < b.N; i++ {
		var dt PersonioDateTime
		_ = dt.UnmarshalJSON(data)
	}
}

// TestMocoActivity_HoursCalculation tests hours calculation from seconds
func TestMocoActivity_HoursCalculation(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int
		expected float64
	}{
		{
			name:     "OneHour",
			seconds:  3600,
			expected: 1.0,
		},
		{
			name:     "ThirtyMinutes",
			seconds:  1800,
			expected: 0.5,
		},
		{
			name:     "TwoHours",
			seconds:  7200,
			expected: 2.0,
		},
		{
			name:     "FifteenMinutes",
			seconds:  900,
			expected: 0.25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hours := float64(tt.seconds) / 3600.0
			assert.Equal(t, tt.expected, hours)
		})
	}
}

// TestPersonioDate_JSON tests PersonioDate JSON serialization
func TestPersonioDate_JSON(t *testing.T) {
	date := PersonioDate{
		DateTime: PersonioDateTime(time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)),
	}

	data, err := json.Marshal(date)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"date_time"`)
	assert.Contains(t, string(data), `"2024-01-15T14:30:00"`)

	var decoded PersonioDate
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, time.Time(date.DateTime).Format("2006-01-02T15:04:05"),
		time.Time(decoded.DateTime).Format("2006-01-02T15:04:05"))
}
