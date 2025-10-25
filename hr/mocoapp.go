// Package hr provides integration with the Moco time tracking API.
// It allows querying projects, tasks, activities, and users, as well as booking time entries.
// Moco is a comprehensive time tracking and project management tool.
//
// Features:
//   - Retrieve projects, tasks, and activities
//   - Manage user employments and contracts
//   - Book time entries for users
//   - Impersonate users when booking time
//   - Delete time entries
//   - Custom date handling for Moco's API format
//
// All functions require a valid Moco domain and API token for authentication.
// The package handles JSON serialization/deserialization with Moco's API format.
package hr

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	eve "eve.evalgo.org/common"
)

// MocoDate represents a date in Moco's format (YYYY-MM-DD).
// It implements json.Marshaler and json.Unmarshaler for proper JSON handling.
type MocoDate time.Time

// MarshalJSON implements the json.Marshaler interface for MocoDate.
// It formats the date as "YYYY-MM-DD" for JSON serialization.
func (d MocoDate) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(d).Format("2006-01-02") + `"`), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for MocoDate.
// It parses dates in "YYYY-MM-DD" format from JSON.
func (d *MocoDate) UnmarshalJSON(b []byte) error {
	value := strings.Trim(string(b), `"`)
	if value == "" || value == "null" {
		return nil
	}
	t, err := time.Parse("2006-01-02", value)
	if err != nil {
		return err
	}
	*d = MocoDate(t)
	return nil
}

// MocoTask represents a task in Moco.
// Tasks are associated with projects and can be used for time tracking.
type MocoTask struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	Billable bool   `json:"billable"`
	Active   bool   `json:"active"`
}

// MocoProject represents a project in Moco.
// Projects contain tasks and can have contracts with users.
type MocoProject struct {
	Id             int        `json:"id"`
	Identifier     string     `json:"identifier"`
	Name           string     `json:"name"`
	Active         bool       `json:"active"`
	Billable       bool       `json:"billable"`
	BillingVariant string     `json:"billing_variant"`
	StartDate      MocoDate   `json:"start_date"`
	FinishDate     MocoDate   `json:"finish_date"`
	Currency       string     `json:"currency"`
	Tasks          []MocoTask `json:"tasks"`
}

// MocoActivity represents a time entry in Moco.
// Activities track time spent on specific tasks within projects.
type MocoActivity struct {
	Id          int         `json:"id"`
	Date        MocoDate    `json:"date"`
	Description string      `json:"description"`
	ProjectId   int         `json:"project_id"`
	TaskId      int         `json:"task_id"`
	Seconds     int         `json:"seconds"`
	Billable    bool        `json:"billable"`
	Hours       float64     `json:"hours"`
	Project     MocoProject `json:"project"`
	User        MocoUser    `json:"user"`
}

// MocoUser represents a user in Moco.
// Users can book time, be assigned to projects, and have employments.
type MocoUser struct {
	Id        int    `json:"id"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Active    bool   `json:"active"`
	Extern    bool   `json:"extern"`
	Email     string `json:"email"`
}

// MocoUserEmployment represents an employment period for a user in Moco.
// Employments define working hours and active periods for users.
type MocoUserEmployment struct {
	Id                int      `json:"id"`
	WeeklyTargetHours float64  `json:"weekly_target_hours"`
	From              MocoDate `json:"from"`
	To                MocoDate `json:"to"`
	User              MocoUser `json:"user"`
}

// MocoUserGroup represents a user contract for a project in Moco.
// This shows which users are assigned to which projects.
type MocoUserGroup struct {
	Id        int    `json:"id"`
	UserId    int    `json:"user_id"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Billable  bool   `json:"billable"`
	Active    bool   `json:"active"`
}

// MocoUserContract represents a contract between a user and a project.
// Used when adding users to projects.
type MocoUserContract struct {
	UserId int `json:"user_id"`
}

// MocoAppProjects retrieves all projects from a Moco account.
//
// Parameters:
//   - domain: The Moco account domain (e.g., "mycompany" for mycompany.mocoapp.com)
//   - token: API token for authentication
//
// Returns:
//   - []MocoProject: List of all projects
func MocoAppProjects(domain string, token string) []MocoProject {
	tgt_url := "https://" + domain + ".mocoapp.com/api/v1/projects.json"
	req, _ := http.NewRequest("GET", tgt_url, nil)
	req.Header.Add("Authorization", "Token token="+token)
	req.Header.Add("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
		return nil
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info(string(body))

	projects := []MocoProject{}
	json.Unmarshal(body, &projects)
	return projects
}

// MocoAppProjectsContracts retrieves all user contracts for a specific project.
//
// Parameters:
//   - domain: The Moco account domain
//   - token: API token for authentication
//   - project_id: ID of the project to query
//
// Returns:
//   - []MocoUserGroup: List of user contracts for the project
func MocoAppProjectsContracts(domain string, token string, project_id int) []MocoUserGroup {
	tgt_url := "https://" + domain + ".mocoapp.com/api/v1/projects/" + strconv.Itoa(project_id) + "/contracts.json"
	req, _ := http.NewRequest("GET", tgt_url, nil)
	req.Header.Add("Authorization", "Token token="+token)
	req.Header.Add("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
		return nil
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	users := []MocoUserGroup{}
	json.Unmarshal(body, &users)
	return users
}

// MocoAppUsers retrieves all users from a Moco account.
//
// Parameters:
//   - domain: The Moco account domain
//   - token: API token for authentication
//
// Returns:
//   - []MocoUser: List of all users
func MocoAppUsers(domain string, token string) []MocoUser {
	tgt_url := "https://" + domain + ".mocoapp.com/api/v1/users.json"
	req, _ := http.NewRequest("GET", tgt_url, nil)
	req.Header.Add("Authorization", "Token token="+token)
	req.Header.Add("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
		return nil
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info(string(body))

	users := []MocoUser{}
	json.Unmarshal(body, &users)
	return users
}

// MocoUserEmployments retrieves all employment periods for a specific user.
//
// Parameters:
//   - domain: The Moco account domain
//   - token: API token for authentication
//   - user_id: ID of the user to query
//
// Returns:
//   - []MocoUserEmployment: List of employment periods for the user
func MocoUserEmployments(domain string, token string, user_id int) []MocoUserEmployment {
	tgt_url := "https://" + domain + ".mocoapp.com/api/v1/users/employments.json?user_id=" + strconv.Itoa(user_id)
	eve.Logger.Info(tgt_url)

	req, _ := http.NewRequest("GET", tgt_url, nil)
	req.Header.Add("Authorization", "Token token="+token)
	req.Header.Add("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
		return nil
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info(string(body))

	employments := []MocoUserEmployment{}
	json.Unmarshal(body, &employments)
	return employments
}

// MocoAppProjectsTasks retrieves all tasks for a specific project.
//
// Parameters:
//   - domain: The Moco account domain
//   - token: API token for authentication
//   - project_id: ID of the project to query
//
// Returns:
//   - []MocoTask: List of tasks for the project
func MocoAppProjectsTasks(domain string, token string, project_id int) []MocoTask {
	tgt_url := "https://" + domain + ".mocoapp.com/api/v1/projects/" + strconv.Itoa(project_id) + "/tasks.json"
	eve.Logger.Info(tgt_url)

	req, _ := http.NewRequest("GET", tgt_url, nil)
	req.Header.Add("Authorization", "Token token="+token)
	req.Header.Add("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
		return nil
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info(string(body))

	tasks := []MocoTask{}
	json.Unmarshal(body, &tasks)
	return tasks
}

// MocoAppActivities retrieves all activities for a specific project and task.
//
// Parameters:
//   - domain: The Moco account domain
//   - token: API token for authentication
//   - project_id: ID of the project to query
//   - task_id: ID of the task to query
//
// Returns:
//   - []MocoActivity: List of activities for the project and task
func MocoAppActivities(domain string, token string, project_id int, task_id int) []MocoActivity {
	tgt_url := "https://" + domain + ".mocoapp.com/api/v1/activities.json?project_id=" +
		strconv.Itoa(project_id) + "&task_id=" + strconv.Itoa(task_id)
	eve.Logger.Info(tgt_url)

	req, _ := http.NewRequest("GET", tgt_url, nil)
	req.Header.Add("Authorization", "Token token="+token)
	req.Header.Add("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
		return nil
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info(string(body))

	activities := []MocoActivity{}
	json.Unmarshal(body, &activities)
	return activities
}

// MocoAppBookImpersonate books time for a task on behalf of another user (impersonation).
//
// Parameters:
//   - domain: The Moco account domain
//   - token: API token for authentication (must have admin rights)
//   - date: Date of the activity in "YYYY-MM-DD" format
//   - project_id: ID of the project
//   - task_id: ID of the task
//   - seconds: Duration of the activity in seconds
//   - description: Description of the activity
//   - impersonate: ID of the user to impersonate
//
// Returns:
//   - error: If the booking fails
func MocoAppBookImpersonate(domain string, token string, date string, project_id int,
	task_id int, seconds int, description string, impersonate int) error {

	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		eve.Logger.Info(err)
		return err
	}

	activity := MocoActivity{
		Date:        MocoDate(t),
		ProjectId:   project_id,
		TaskId:      task_id,
		Description: description,
		Seconds:     seconds,
		Billable:    true,
		Hours:       float64(seconds) / 3600.0,
	}

	tgt_url := "https://" + domain + ".mocoapp.com/api/v1/activities"
	jsonStr, err := json.Marshal(activity)
	if err != nil {
		eve.Logger.Info(err)
		return err
	}

	req, _ := http.NewRequest("POST", tgt_url, bytes.NewBuffer(jsonStr))
	req.Header.Add("X-IMPERSONATE-USER-ID", strconv.Itoa(impersonate))
	req.Header.Add("Authorization", "Token token="+token)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Info(err)
		return err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info(string(body))
	return nil
}

// MocoAppBook books time for a task.
// The authenticated user will be the owner of the time entry.
//
// Parameters:
//   - domain: The Moco account domain
//   - token: API token for authentication
//   - date: Date of the activity in "YYYY-MM-DD" format
//   - project_id: ID of the project
//   - task_id: ID of the task
//   - seconds: Duration of the activity in seconds
//   - description: Description of the activity
//
// Returns:
//   - error: If the booking fails
func MocoAppBook(domain string, token string, date string, project_id int,
	task_id int, seconds int, description string) error {

	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		eve.Logger.Info(err)
		return err
	}

	activity := MocoActivity{
		Date:        MocoDate(t),
		ProjectId:   project_id,
		TaskId:      task_id,
		Description: description,
		Seconds:     seconds,
		Billable:    true,
		Hours:       float64(seconds) / 3600.0,
	}

	tgt_url := "https://" + domain + ".mocoapp.com/api/v1/activities"
	jsonStr, err := json.Marshal(activity)
	if err != nil {
		eve.Logger.Info(err)
		return err
	}

	req, _ := http.NewRequest("POST", tgt_url, bytes.NewBuffer(jsonStr))
	req.Header.Add("Authorization", "Token token="+token)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Info(err)
		return err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info(string(body))
	return nil
}

// MocoAppBookDelete deletes a time entry from Moco.
//
// Parameters:
//   - domain: The Moco account domain
//   - token: API token for authentication
//   - activityId: ID of the activity to delete
//
// Returns:
//   - error: If the deletion fails
func MocoAppBookDelete(domain string, token string, activityId int) error {
	tgt_url := "https://" + domain + ".mocoapp.com/api/v1/activities/" + strconv.Itoa(activityId)
	req, _ := http.NewRequest("DELETE", tgt_url, nil)
	req.Header.Add("Authorization", "Token token="+token)
	req.Header.Add("Accept", "application/json")

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

// MocoAppNewProjectContract adds a user to a project as a contractor.
//
// Parameters:
//   - domain: The Moco account domain
//   - token: API token for authentication
//   - project_id: ID of the project
//   - user: MocoUserContract containing the user ID to add
//
// Returns:
//   - error: If the operation fails
func MocoAppNewProjectContract(domain string, token string, project_id int, user MocoUserContract) error {
	tgt_url := "https://" + domain + ".mocoapp.com/api/v1/projects/" + strconv.Itoa(project_id) + "/contracts"
	jsonStr, err := json.Marshal(user)
	if err != nil {
		eve.Logger.Info(err)
		return err
	}

	req, _ := http.NewRequest("POST", tgt_url, bytes.NewBuffer(jsonStr))
	req.Header.Add("Authorization", "Token token="+token)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Info(err)
		return err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info(string(body))
	return nil
}
