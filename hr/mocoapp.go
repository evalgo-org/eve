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

type MocoDate time.Time

func (d MocoDate) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(d).Format("2006-01-02") + `"`), nil
}

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

type MocoTask struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	Billable bool   `json:"billable"`
	Active   bool   `json:"active"`
}

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

type MocoUser struct {
	Id        int    `json:"id"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Active    bool   `json:"active"`
	Extern    bool   `json:"extern"`
	Email     string `json:"email"`
}

type MocoUserEmployment struct {
	Id                int      `json:"id"`
	WeeklyTargetHours float64  `json:"weekly_target_hours"`
	From              MocoDate `json:"from"`
	To                MocoDate `json:"to"`
	User              MocoUser `json:"user"`
}

type MocoUserGroup struct {
	Id        int    `json:"id"`
	UserId    int    `json:"user_id"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Billable  bool   `json:"billable"`
	Active    bool   `json:"active"`
}

type MocoUserContract struct {
	UserId int `json:"user_id"`
}

func MocoAppProjects(domain string, token string) []MocoProject {
	tgt_url := "https://" + domain + ".mocoapp.com/api/v1/projects.json"
	req, _ := http.NewRequest("GET", tgt_url, nil)
	req.Header.Add("Authorization", "Token token="+token)
	req.Header.Add("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info(string(body))
	projects := []MocoProject{}
	eve.Logger.Info(json.Unmarshal(body, &projects))
	return projects
}

func MocoAppProjectsContracts(domain string, token string, project_id int) []MocoUserGroup {
	tgt_url := "https://" + domain + ".mocoapp.com/api/v1/projects/" + strconv.Itoa(project_id) + "/contracts.json"
	req, _ := http.NewRequest("GET", tgt_url, nil)
	req.Header.Add("Authorization", "Token token="+token)
	req.Header.Add("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	// eve.Logger.Info(string(body))
	users := []MocoUserGroup{}
	eve.Logger.Info(json.Unmarshal(body, &users))
	return users
}

func MocoAppUsers(domain string, token string) []MocoUser {
	tgt_url := "https://" + domain + ".mocoapp.com/api/v1/users.json"
	req, _ := http.NewRequest("GET", tgt_url, nil)
	req.Header.Add("Authorization", "Token token="+token)
	req.Header.Add("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info(string(body))
	users := []MocoUser{}
	eve.Logger.Info(json.Unmarshal(body, &users))
	return users
}

func MocoUserEmployments(domain string, token string, user_id int) []MocoUserEmployment {
	tgt_url := "https://" + domain + ".mocoapp.com/api/v1/users/employments.json?user_id=" + strconv.Itoa(user_id)
	eve.Logger.Info(tgt_url)
	req, _ := http.NewRequest("GET", tgt_url, nil)
	req.Header.Add("Authorization", "Token token="+token)
	req.Header.Add("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info(string(body))
	employments := []MocoUserEmployment{}
	eve.Logger.Info(json.Unmarshal(body, &employments))
	return employments
}

func MocoAppProjectsTasks(domain string, token string, project_id int) []MocoTask {
	tgt_url := "https://" + domain + ".mocoapp.com/api/v1/projects/" + strconv.Itoa(project_id) + "/tasks.json"
	eve.Logger.Info(tgt_url)
	req, _ := http.NewRequest("GET", tgt_url, nil)
	req.Header.Add("Authorization", "Token token="+token)
	req.Header.Add("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info(string(body))
	tasks := []MocoTask{}
	eve.Logger.Info(json.Unmarshal(body, &tasks))
	return tasks
}

func MocoAppActivities(domain string, token string, project_id int, task_id int) []MocoActivity {
	tgt_url := "https://" + domain + ".mocoapp.com/api/v1/activities.json?project_id=" + strconv.Itoa(project_id) + "&task_id=" + strconv.Itoa(task_id)
	eve.Logger.Info(tgt_url)
	req, _ := http.NewRequest("GET", tgt_url, nil)
	req.Header.Add("Authorization", "Token token="+token)
	req.Header.Add("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info(string(body))
	activities := []MocoActivity{}
	eve.Logger.Info(json.Unmarshal(body, &activities))
	return activities
}

func MocoAppBookImpersonate(domain string, token string, date string, project_id int, task_id int, seconds int, description string, impersonate int) error {
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
		Hours:       1.0,
	}
	tgt_url := "https://" + domain + ".mocoapp.com/api/v1/activities"
	jsonStr, err := json.Marshal(activity)
	if err != nil {
		eve.Logger.Info(err)
		return err
	}
	// eve.Logger.Info(string(jsonStr))
	// return nil
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
	eve.Logger.Info(res)
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info(string(body))
	return nil
	// projects := []MocoProject{}
	// eve.Logger.Info(json.Unmarshal(body, &projects))
	// return projects
}

func MocoAppBook(domain string, token string, date string, project_id int, task_id int, seconds int, description string) error {
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
		Hours:       1.0,
	}
	tgt_url := "https://" + domain + ".mocoapp.com/api/v1/activities"
	jsonStr, err := json.Marshal(activity)
	if err != nil {
		eve.Logger.Info(err)
		return err
	}
	// eve.Logger.Info(string(jsonStr))
	// return nil
	req, _ := http.NewRequest("POST", tgt_url, bytes.NewBuffer(jsonStr))
	req.Header.Add("Authorization", "Token token="+token)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Info(err)
		return err
	}
	eve.Logger.Info(res)
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info(string(body))
	return nil
	// projects := []MocoProject{}
	// eve.Logger.Info(json.Unmarshal(body, &projects))
	// return projects
}

func MocoAppBookDelete(domain string, token string, activityId int) error {
	tgt_url := "https://" + domain + ".mocoapp.com/api/v1/activities/" + strconv.Itoa(activityId)
	req, _ := http.NewRequest("DELETE", tgt_url, nil)
	req.Header.Add("Authorization", "Token token="+token)
	req.Header.Add("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info(string(body))
	return nil
}

func MocoAppNewProjectContract(domain string, token string, project_id int, user MocoUserContract) error {
	tgt_url := "https://" + domain + ".mocoapp.com/api/v1/projects/" + strconv.Itoa(project_id) + "/contracts"
	jsonStr, err := json.Marshal(user)
	if err != nil {
		eve.Logger.Info(err)
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
