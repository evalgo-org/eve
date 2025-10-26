// Package hr provides testable HTTP clients for Moco and Personio APIs.
// This file implements dependency injection pattern for better testability.
package hr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	eve "eve.evalgo.org/common"
)

// HTTPClient is an interface for making HTTP requests.
// This allows for easy mocking in tests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// MocoClient is a client for interacting with the Moco API.
// It uses dependency injection for the HTTP client to enable testing.
type MocoClient struct {
	domain     string
	token      string
	httpClient HTTPClient
}

// NewMocoClient creates a new Moco API client.
//
// Parameters:
//   - domain: The Moco account domain (e.g., "mycompany" for mycompany.mocoapp.com)
//   - token: API token for authentication
//
// Returns:
//   - *MocoClient: Configured Moco client
func NewMocoClient(domain, token string) *MocoClient {
	return &MocoClient{
		domain:     domain,
		token:      token,
		httpClient: http.DefaultClient,
	}
}

// NewMocoClientWithHTTP creates a new Moco API client with a custom HTTP client.
// This is primarily useful for testing with mock HTTP clients.
//
// Parameters:
//   - domain: The Moco account domain
//   - token: API token for authentication
//   - httpClient: Custom HTTP client implementation
//
// Returns:
//   - *MocoClient: Configured Moco client
func NewMocoClientWithHTTP(domain, token string, httpClient HTTPClient) *MocoClient {
	return &MocoClient{
		domain:     domain,
		token:      token,
		httpClient: httpClient,
	}
}

// Projects retrieves all projects from the Moco account.
//
// Returns:
//   - []MocoProject: List of all projects
//   - error: If the request fails
func (c *MocoClient) Projects() ([]MocoProject, error) {
	url := fmt.Sprintf("https://%s.mocoapp.com/api/v1/projects.json", c.domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", "Token token="+c.token)
	req.Header.Add("Accept", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("API returned status %d: %s", res.StatusCode, string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var projects []MocoProject
	if err := json.Unmarshal(body, &projects); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return projects, nil
}

// ProjectContracts retrieves all user contracts for a specific project.
//
// Parameters:
//   - projectID: ID of the project to query
//
// Returns:
//   - []MocoUserGroup: List of user contracts for the project
//   - error: If the request fails
func (c *MocoClient) ProjectContracts(projectID int) ([]MocoUserGroup, error) {
	url := fmt.Sprintf("https://%s.mocoapp.com/api/v1/projects/%d/contracts.json", c.domain, projectID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", "Token token="+c.token)
	req.Header.Add("Accept", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("API returned status %d: %s", res.StatusCode, string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var users []MocoUserGroup
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return users, nil
}

// Users retrieves all users from the Moco account.
//
// Returns:
//   - []MocoUser: List of all users
//   - error: If the request fails
func (c *MocoClient) Users() ([]MocoUser, error) {
	url := fmt.Sprintf("https://%s.mocoapp.com/api/v1/users.json", c.domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", "Token token="+c.token)
	req.Header.Add("Accept", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("API returned status %d: %s", res.StatusCode, string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var users []MocoUser
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return users, nil
}

// UserEmployments retrieves all employment periods for a specific user.
//
// Parameters:
//   - userID: ID of the user to query
//
// Returns:
//   - []MocoUserEmployment: List of employment periods
//   - error: If the request fails
func (c *MocoClient) UserEmployments(userID int) ([]MocoUserEmployment, error) {
	url := fmt.Sprintf("https://%s.mocoapp.com/api/v1/users/employments.json?user_id=%d", c.domain, userID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", "Token token="+c.token)
	req.Header.Add("Accept", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("API returned status %d: %s", res.StatusCode, string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var employments []MocoUserEmployment
	if err := json.Unmarshal(body, &employments); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return employments, nil
}

// ProjectTasks retrieves all tasks for a specific project.
//
// Parameters:
//   - projectID: ID of the project to query
//
// Returns:
//   - []MocoTask: List of tasks
//   - error: If the request fails
func (c *MocoClient) ProjectTasks(projectID int) ([]MocoTask, error) {
	url := fmt.Sprintf("https://%s.mocoapp.com/api/v1/projects/%d/tasks.json", c.domain, projectID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", "Token token="+c.token)
	req.Header.Add("Accept", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("API returned status %d: %s", res.StatusCode, string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var tasks []MocoTask
	if err := json.Unmarshal(body, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return tasks, nil
}

// Activities retrieves activities for a specific project and task.
//
// Parameters:
//   - projectID: ID of the project
//   - taskID: ID of the task
//
// Returns:
//   - []MocoActivity: List of activities
//   - error: If the request fails
func (c *MocoClient) Activities(projectID, taskID int) ([]MocoActivity, error) {
	url := fmt.Sprintf("https://%s.mocoapp.com/api/v1/activities.json?project_id=%d&task_id=%d",
		c.domain, projectID, taskID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", "Token token="+c.token)
	req.Header.Add("Accept", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("API returned status %d: %s", res.StatusCode, string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var activities []MocoActivity
	if err := json.Unmarshal(body, &activities); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return activities, nil
}

// BookActivity books a time entry for a task.
//
// Parameters:
//   - date: Date in "YYYY-MM-DD" format
//   - projectID: ID of the project
//   - taskID: ID of the task
//   - seconds: Duration in seconds
//   - description: Activity description
//
// Returns:
//   - error: If the booking fails
func (c *MocoClient) BookActivity(date string, projectID, taskID, seconds int, description string) error {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("invalid date format: %w", err)
	}

	activity := MocoActivity{
		Date:        MocoDate(t),
		ProjectId:   projectID,
		TaskId:      taskID,
		Description: description,
		Seconds:     seconds,
		Billable:    true,
		Hours:       float64(seconds) / 3600.0,
	}

	url := fmt.Sprintf("https://%s.mocoapp.com/api/v1/activities", c.domain)
	jsonStr, err := json.Marshal(activity)
	if err != nil {
		return fmt.Errorf("failed to marshal activity: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", "Token token="+c.token)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("API returned status %d: %s", res.StatusCode, string(body))
	}

	return nil
}

// BookActivityImpersonate books a time entry on behalf of another user.
//
// Parameters:
//   - date: Date in "YYYY-MM-DD" format
//   - projectID: ID of the project
//   - taskID: ID of the task
//   - seconds: Duration in seconds
//   - description: Activity description
//   - impersonateUserID: ID of user to impersonate
//
// Returns:
//   - error: If the booking fails
func (c *MocoClient) BookActivityImpersonate(date string, projectID, taskID, seconds int, description string, impersonateUserID int) error {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("invalid date format: %w", err)
	}

	activity := MocoActivity{
		Date:        MocoDate(t),
		ProjectId:   projectID,
		TaskId:      taskID,
		Description: description,
		Seconds:     seconds,
		Billable:    true,
		Hours:       float64(seconds) / 3600.0,
	}

	url := fmt.Sprintf("https://%s.mocoapp.com/api/v1/activities", c.domain)
	jsonStr, err := json.Marshal(activity)
	if err != nil {
		return fmt.Errorf("failed to marshal activity: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("X-IMPERSONATE-USER-ID", strconv.Itoa(impersonateUserID))
	req.Header.Add("Authorization", "Token token="+c.token)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("API returned status %d: %s", res.StatusCode, string(body))
	}

	return nil
}

// DeleteActivity deletes a time entry.
//
// Parameters:
//   - activityID: ID of the activity to delete
//
// Returns:
//   - error: If the deletion fails
func (c *MocoClient) DeleteActivity(activityID int) error {
	url := fmt.Sprintf("https://%s.mocoapp.com/api/v1/activities/%d", c.domain, activityID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", "Token token="+c.token)
	req.Header.Add("Accept", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("API returned status %d: %s", res.StatusCode, string(body))
	}

	return nil
}

// AddProjectContract adds a user to a project as a contractor.
//
// Parameters:
//   - projectID: ID of the project
//   - userID: ID of the user to add
//
// Returns:
//   - error: If the operation fails
func (c *MocoClient) AddProjectContract(projectID, userID int) error {
	url := fmt.Sprintf("https://%s.mocoapp.com/api/v1/projects/%d/contracts", c.domain, projectID)
	user := MocoUserContract{UserId: userID}
	jsonStr, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal contract: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", "Token token="+c.token)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("API returned status %d: %s", res.StatusCode, string(body))
	}

	return nil
}

// PersonioClient is a client for interacting with the Personio API.
// It uses dependency injection for the HTTP client to enable testing.
type PersonioClient struct {
	clientID     string
	clientSecret string
	partnerID    string
	appID        string
	httpClient   HTTPClient
	token        *PersonioToken
}

// NewPersonioClient creates a new Personio API client.
// It reads PERSONIO_PARTNER_ID and PERSONIO_APP_ID from environment variables.
//
// Parameters:
//   - clientID: OAuth client ID
//   - clientSecret: OAuth client secret
//
// Returns:
//   - *PersonioClient: Configured Personio client
func NewPersonioClient(clientID, clientSecret string) *PersonioClient {
	return &PersonioClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		partnerID:    os.Getenv("PERSONIO_PARTNER_ID"),
		appID:        os.Getenv("PERSONIO_APP_ID"),
		httpClient:   http.DefaultClient,
	}
}

// NewPersonioClientWithHTTP creates a new Personio API client with a custom HTTP client.
//
// Parameters:
//   - clientID: OAuth client ID
//   - clientSecret: OAuth client secret
//   - partnerID: Personio partner ID
//   - appID: Personio app ID
//   - httpClient: Custom HTTP client implementation
//
// Returns:
//   - *PersonioClient: Configured Personio client
func NewPersonioClientWithHTTP(clientID, clientSecret, partnerID, appID string, httpClient HTTPClient) *PersonioClient {
	return &PersonioClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		partnerID:    partnerID,
		appID:        appID,
		httpClient:   httpClient,
	}
}

// obtainToken obtains an OAuth2 access token from Personio.
func (c *PersonioClient) obtainToken() error {
	url := "https://api.personio.de/v2/auth/token"
	payload := strings.NewReader("grant_type=client_credentials&client_id=" + c.clientID + "&client_secret=" + c.clientSecret)

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("X-Personio-Partner-ID", c.partnerID)
	req.Header.Add("X-Personio-App-ID", c.appID)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("token request failed with status %d: %s", res.StatusCode, string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	token := PersonioToken{}
	if err := json.Unmarshal(body, &token); err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	c.token = &token
	return nil
}

// ensureToken ensures we have a valid access token.
func (c *PersonioClient) ensureToken() error {
	if c.token == nil {
		return c.obtainToken()
	}
	return nil
}

// GetPerson retrieves information about a specific person.
//
// Parameters:
//   - personID: ID of the person to retrieve
//
// Returns:
//   - PersonioPerson: The person's information
//   - error: If the request fails
func (c *PersonioClient) GetPerson(personID string) (PersonioPerson, error) {
	if err := c.ensureToken(); err != nil {
		return PersonioPerson{}, err
	}

	url := "https://api.personio.de/v2/persons/" + personID
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return PersonioPerson{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("X-Personio-Partner-ID", c.partnerID)
	req.Header.Add("X-Personio-App-ID", c.appID)
	req.Header.Add("Authorization", "Bearer "+c.token.AccessToken)
	req.Header.Add("Accept", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return PersonioPerson{}, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return PersonioPerson{}, fmt.Errorf("API returned status %d: %s", res.StatusCode, string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return PersonioPerson{}, fmt.Errorf("failed to read response: %w", err)
	}

	var person PersonioPerson
	if err := json.Unmarshal(body, &person); err != nil {
		return PersonioPerson{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return person, nil
}

// BookAttendance books a new attendance period.
//
// Parameters:
//   - personID: ID of the person for whom to book
//   - start: Start date/time in "YYYY-MM-DDTHH:MM:SS" format
//   - end: End date/time in "YYYY-MM-DDTHH:MM:SS" format
//
// Returns:
//   - error: If the booking fails
func (c *PersonioClient) BookAttendance(personID, start, end string) error {
	if err := c.ensureToken(); err != nil {
		return err
	}

	url := "https://api.personio.de/v2/attendance-periods?skip_approval=false"
	payload := strings.NewReader(`{"type":"WORK","person": {"id":"` + personID + `"},"start": {"date_time":"` + start + `"},"end": {"date_time":"` + end + `"}}`)

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("X-Personio-Partner-ID", c.partnerID)
	req.Header.Add("X-Personio-App-ID", c.appID)
	req.Header.Add("Authorization", "Bearer "+c.token.AccessToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Beta", "true")
	req.Header.Add("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("API returned status %d: %s", res.StatusCode, string(body))
	}

	return nil
}

// Backward compatibility wrappers for existing functions
// These maintain the old API while using the new client internally

// MocoAppProjects is a backward-compatible wrapper.
// Deprecated: Use NewMocoClient().Projects() instead.
func MocoAppProjects(domain string, token string) []MocoProject {
	client := NewMocoClient(domain, token)
	projects, err := client.Projects()
	if err != nil {
		eve.Logger.Error(err)
		return nil
	}
	return projects
}

// MocoAppProjectsContracts is a backward-compatible wrapper.
// Deprecated: Use NewMocoClient().ProjectContracts() instead.
func MocoAppProjectsContracts(domain string, token string, project_id int) []MocoUserGroup {
	client := NewMocoClient(domain, token)
	users, err := client.ProjectContracts(project_id)
	if err != nil {
		eve.Logger.Error(err)
		return nil
	}
	return users
}

// MocoAppUsers is a backward-compatible wrapper.
// Deprecated: Use NewMocoClient().Users() instead.
func MocoAppUsers(domain string, token string) []MocoUser {
	client := NewMocoClient(domain, token)
	users, err := client.Users()
	if err != nil {
		eve.Logger.Error(err)
		return nil
	}
	return users
}

// MocoUserEmployments is a backward-compatible wrapper.
// Deprecated: Use NewMocoClient().UserEmployments() instead.
func MocoUserEmployments(domain string, token string, user_id int) []MocoUserEmployment {
	client := NewMocoClient(domain, token)
	employments, err := client.UserEmployments(user_id)
	if err != nil {
		eve.Logger.Error(err)
		return nil
	}
	return employments
}

// MocoAppProjectsTasks is a backward-compatible wrapper.
// Deprecated: Use NewMocoClient().ProjectTasks() instead.
func MocoAppProjectsTasks(domain string, token string, project_id int) []MocoTask {
	client := NewMocoClient(domain, token)
	tasks, err := client.ProjectTasks(project_id)
	if err != nil {
		eve.Logger.Error(err)
		return nil
	}
	return tasks
}

// MocoAppActivities is a backward-compatible wrapper.
// Deprecated: Use NewMocoClient().Activities() instead.
func MocoAppActivities(domain string, token string, project_id int, task_id int) []MocoActivity {
	client := NewMocoClient(domain, token)
	activities, err := client.Activities(project_id, task_id)
	if err != nil {
		eve.Logger.Error(err)
		return nil
	}
	return activities
}

// MocoAppBookImpersonate is a backward-compatible wrapper.
// Deprecated: Use NewMocoClient().BookActivityImpersonate() instead.
func MocoAppBookImpersonate(domain string, token string, date string, project_id int,
	task_id int, seconds int, description string, impersonate int) error {
	client := NewMocoClient(domain, token)
	return client.BookActivityImpersonate(date, project_id, task_id, seconds, description, impersonate)
}

// MocoAppBook is a backward-compatible wrapper.
// Deprecated: Use NewMocoClient().BookActivity() instead.
func MocoAppBook(domain string, token string, date string, project_id int,
	task_id int, seconds int, description string) error {
	client := NewMocoClient(domain, token)
	return client.BookActivity(date, project_id, task_id, seconds, description)
}

// MocoAppBookDelete is a backward-compatible wrapper.
// Deprecated: Use NewMocoClient().DeleteActivity() instead.
func MocoAppBookDelete(domain string, token string, activityId int) error {
	client := NewMocoClient(domain, token)
	return client.DeleteActivity(activityId)
}

// MocoAppNewProjectContract is a backward-compatible wrapper.
// Deprecated: Use NewMocoClient().AddProjectContract() instead.
func MocoAppNewProjectContract(domain string, token string, project_id int, user MocoUserContract) error {
	client := NewMocoClient(domain, token)
	return client.AddProjectContract(project_id, user.UserId)
}
