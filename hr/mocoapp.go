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
	"strings"
	"time"
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

// Note: MocoAppProjects and other Moco functions have been moved to client.go
// with proper dependency injection for testability. The old function signatures
// are preserved as backward-compatible wrappers in client.go.
