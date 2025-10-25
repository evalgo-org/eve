// Package hr provides integration with the Personio HR management system API.
// It allows querying employee data, attendance periods, and booking time entries.
// Personio is a comprehensive HR management platform for modern companies.
//
// Features:
//   - Authentication with Personio using OAuth2 client credentials
//   - Retrieval of employee information
//   - Querying attendance periods and approval statuses
//   - Booking new attendance periods
//   - Custom date/time handling for Personio's API format
//
// All functions require valid Personio credentials and environment variables:
//   - PERSONIO_PARTNER_ID: Your Personio partner ID
//   - PERSONIO_APP_ID: Your Personio application ID
package hr

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	eve "eve.evalgo.org/common"
)

// PersonioDateTime represents a date and time in Personio's format (YYYY-MM-DDTHH:MM:SS).
// It implements json.Marshaler and json.Unmarshaler for proper JSON handling.
type PersonioDateTime time.Time

// MarshalJSON implements the json.Marshaler interface for PersonioDateTime.
// It formats the date/time as "YYYY-MM-DDTHH:MM:SS" for JSON serialization.
func (d PersonioDateTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(d).Format("2006-01-02T15:04:05") + `"`), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for PersonioDateTime.
// It parses dates in "YYYY-MM-DDTHH:MM:SS" format from JSON.
func (d *PersonioDateTime) UnmarshalJSON(b []byte) error {
	value := strings.Trim(string(b), `"`)
	if value == "" || value == "null" {
		return nil
	}
	t, err := time.Parse("2006-01-02T15:04:05", value)
	if err != nil {
		return err
	}
	*d = PersonioDateTime(t)
	return nil
}

// PersonioDate represents a date with time in Personio's API format.
// Used for attendance period start and end times.
type PersonioDate struct {
	DateTime PersonioDateTime `json:"date_time"`
}

// PersonioToken represents an OAuth2 access token from Personio.
// Used for authenticating API requests.
type PersonioToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// PersonioPerson represents an employee in Personio.
// Contains basic employee information.
type PersonioPerson struct {
	Id        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// PersonioApproval represents the approval status of an attendance period.
type PersonioApproval struct {
	Status string `json:"status"`
}

// PersonioAttendance represents an attendance period in Personio.
// Contains information about the type, timing, and approval status of the attendance.
type PersonioAttendance struct {
	Id       string           `json:"id"`
	Start    PersonioDate     `json:"start"`
	End      PersonioDate     `json:"end"`
	AType    string           `json:"type"`
	Approval PersonioApproval `json:"approval"`
}

// PersonioPersonResponse represents the API response for person queries.
// Contains a list of persons and metadata.
type PersonioPersonResponse struct {
	Data []PersonioPerson                  `json:"_data"`
	Meta map[string]map[string]interface{} `json:"_meta"`
}

// PersonioAttendanceResponse represents the API response for attendance queries.
// Contains a list of attendance periods and metadata.
type PersonioAttendanceResponse struct {
	Data []PersonioAttendance              `json:"_data"`
	Meta map[string]map[string]interface{} `json:"_meta"`
}

// personioObtainToken obtains an OAuth2 access token from Personio using client credentials.
// This token is required for authenticating with the Personio API.
//
// Parameters:
//   - clientId: The client ID for authentication
//   - clientSecret: The client secret for authentication
//
// Returns:
//   - PersonioToken: The access token and related information
//
// Environment Variables:
//   - PERSONIO_PARTNER_ID: Required for the API request
//   - PERSONIO_APP_ID: Required for the API request
func personioObtainToken(clientId string, clientSecret string) PersonioToken {
	url := "https://api.personio.de/v2/auth/token"
	payload := strings.NewReader("grant_type=client_credentials&client_id=" + clientId + "&client_secret=" + clientSecret)

	req, _ := http.NewRequest("POST", url, payload)
	req.Header.Add("X-Personio-Partner-ID", os.Getenv("PERSONIO_PARTNER_ID"))
	req.Header.Add("X-Personio-App-ID", os.Getenv("PERSONIO_APP_ID"))
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
		return PersonioToken{}
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info(string(body))

	token := PersonioToken{}
	json.Unmarshal(body, &token)
	return token
}

// PersonioUser retrieves information about a specific person from Personio.
//
// Parameters:
//   - token: Valid Personio access token
//   - personId: ID of the person to retrieve
//
// Returns:
//   - PersonioPerson: The person's information
//
// Environment Variables:
//   - PERSONIO_PARTNER_ID: Required for the API request
//   - PERSONIO_APP_ID: Required for the API request
func PersonioUser(token PersonioToken, personId string) PersonioPerson {
	tgt_url := "https://api.personio.de/v2/persons/" + personId

	req, _ := http.NewRequest("GET", tgt_url, nil)
	req.Header.Add("X-Personio-Partner-ID", os.Getenv("PERSONIO_PARTNER_ID"))
	req.Header.Add("X-Personio-App-ID", os.Getenv("PERSONIO_APP_ID"))
	req.Header.Add("Authorization", "Bearer "+token.AccessToken)
	req.Header.Add("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
		return PersonioPerson{}
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	person := PersonioPerson{}
	err = json.Unmarshal(body, &person)
	if err != nil {
		eve.Logger.Info(err)
		return PersonioPerson{}
	}

	return person
}

// PersonioAttendancesPeriods retrieves all attendance periods from Personio.
// This function is currently a placeholder that logs the raw API response.
//
// Parameters:
//   - token: Valid Personio access token
//
// Returns:
//   - error: If the request fails
//
// Environment Variables:
//   - PERSONIO_PARTNER_ID: Required for the API request
//   - PERSONIO_APP_ID: Required for the API request
func PersonioAttendancesPeriods(token PersonioToken) error {
	tgt_url := "https://api.personio.de/v2/attendance-periods"

	req, _ := http.NewRequest("GET", tgt_url, nil)
	req.Header.Add("X-Personio-Partner-ID", os.Getenv("PERSONIO_PARTNER_ID"))
	req.Header.Add("X-Personio-App-ID", os.Getenv("PERSONIO_APP_ID"))
	req.Header.Add("Authorization", "Bearer "+token.AccessToken)
	req.Header.Add("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
		return err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info("PersonioAttendances", string(body))
	return nil
}

// PersonioAttendancesPerson retrieves attendance periods for a specific person.
// This function is currently a placeholder that logs the raw API response.
//
// Parameters:
//   - token: Valid Personio access token
//   - personId: ID of the person whose attendances to retrieve
//
// Returns:
//   - []PersonioAttendance: Empty slice (function needs implementation)
//
// Environment Variables:
//   - PERSONIO_PARTNER_ID: Required for the API request
//   - PERSONIO_APP_ID: Required for the API request
func PersonioAttendancesPerson(token PersonioToken, personId string) []PersonioAttendance {
	tgt_url := "https://api.personio.de/v2/attendance-periods/" + personId

	req, _ := http.NewRequest("GET", tgt_url, nil)
	req.Header.Add("X-Personio-Partner-ID", os.Getenv("PERSONIO_PARTNER_ID"))
	req.Header.Add("X-Personio-App-ID", os.Getenv("PERSONIO_APP_ID"))
	req.Header.Add("Authorization", "Bearer "+token.AccessToken)
	req.Header.Add("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
		return []PersonioAttendance{}
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info("PersonioAttendances", string(body))
	return []PersonioAttendance{}
}

// PersonioAttendances retrieves attendance periods for a specific person with filtering.
// This function retrieves pending attendance periods for a specific person and logs them.
//
// Parameters:
//   - token: Valid Personio access token
//   - personId: ID of the person whose attendances to retrieve
//
// Returns:
//   - error: If the request or parsing fails
//
// Environment Variables:
//   - PERSONIO_PARTNER_ID: Required for the API request
//   - PERSONIO_APP_ID: Required for the API request
func PersonioAttendances(token PersonioToken, personId string) error {
	tgt_url := "https://api.personio.de/v2/attendance-periods"

	req, _ := http.NewRequest("GET", tgt_url, nil)
	values := req.URL.Query()
	values.Add("limit", "50")
	values.Add("person.id", personId)
	values.Add("status", "PENDING")
	req.URL.RawQuery = values.Encode()

	req.Header.Add("X-Personio-Partner-ID", os.Getenv("PERSONIO_PARTNER_ID"))
	req.Header.Add("X-Personio-App-ID", os.Getenv("PERSONIO_APP_ID"))
	req.Header.Add("Authorization", "Bearer "+token.AccessToken)
	req.Header.Add("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
		return err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info("=============>>>>>>>>>>>>>>>>>>>>>>", string(body))

	resp := PersonioAttendanceResponse{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}

	for _, attendancePeriod := range resp.Data {
		eve.Logger.Info(attendancePeriod.Id, ": ", attendancePeriod.Start, "=>",
			attendancePeriod.End, ">", attendancePeriod.AType, ">",
			attendancePeriod.Approval.Status)
	}

	return nil
}

// PersonioBook books a new attendance period in Personio.
// This function creates a new WORK type attendance period for a specific person.
//
// Parameters:
//   - clientId: The client ID for authentication
//   - clientSecret: The client secret for authentication
//   - personId: ID of the person for whom to book the attendance
//   - start: Start date/time in "YYYY-MM-DDTHH:MM:SS" format
//   - end: End date/time in "YYYY-MM-DDTHH:MM:SS" format
//
// Returns:
//   - error: If the request fails
//
// Environment Variables:
//   - PERSONIO_PARTNER_ID: Required for the API request
//   - PERSONIO_APP_ID: Required for the API request
func PersonioBook(clientId string, clientSecret string, personId string, start string, end string) error {
	token := personioObtainToken(clientId, clientSecret)
	url := "https://api.personio.de/v2/attendance-periods"

	payload := strings.NewReader(`{"type":"WORK","person": {"id":"` + personId + `"},"start": {"date_time":"` + start + `"},"end": {"date_time":"` + end + `"}}`)

	req, _ := http.NewRequest("POST", url, payload)
	values := req.URL.Query()
	values.Add("skip_approval", "false")
	req.URL.RawQuery = values.Encode()

	req.Header.Add("X-Personio-Partner-ID", os.Getenv("PERSONIO_PARTNER_ID"))
	req.Header.Add("X-Personio-App-ID", os.Getenv("PERSONIO_APP_ID"))
	req.Header.Add("Authorization", "Bearer "+token.AccessToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Beta", "true")
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
		return err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info(string(body))
	return nil
}

// PersonioUsers retrieves all users from Personio.
// This function retrieves a list of users and logs their information.
// For users named "Francisc Simon", it also retrieves their attendance records.
//
// Parameters:
//   - clientId: The client ID for authentication
//   - clientSecret: The client secret for authentication
//
// Returns:
//   - error: If the request or processing fails
//
// Environment Variables:
//   - PERSONIO_PARTNER_ID: Required for the API request
//   - PERSONIO_APP_ID: Required for the API request
func PersonioUsers(clientId string, clientSecret string) error {
	token := personioObtainToken(clientId, clientSecret)
	tgt_url := "https://api.personio.de/v2/persons?limit=50"

	req, _ := http.NewRequest("GET", tgt_url, nil)
	req.Header.Add("X-Personio-Partner-ID", os.Getenv("PERSONIO_PARTNER_ID"))
	req.Header.Add("X-Personio-App-ID", os.Getenv("PERSONIO_APP_ID"))
	req.Header.Add("Authorization", "Bearer "+token.AccessToken)
	req.Header.Add("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
		return err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	resp := PersonioPersonResponse{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}

	for _, r := range resp.Data {
		eve.Logger.Info(r.Id, ": ", r.FirstName, " ", r.LastName, " (", r.Email, ") ")
		if r.FirstName == "Francisc" && r.LastName == "Simon" {
			eve.Logger.Info(PersonioAttendances(token, r.Id))
		}
	}

	return nil
}
