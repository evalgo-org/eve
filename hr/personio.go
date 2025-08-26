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

type PersonioDateTime time.Time

type PersonioDate struct {
	DateTime PersonioDateTime `json:"date_time"`
}

func (d PersonioDateTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(d).Format("2006-01-02T15:04:05") + `"`), nil
}

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

type PersonioToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type PersonioPerson struct {
	Id        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type PersonioApproval struct {
	Status string `json:"status"`
}

type PersonioAttendance struct {
	Id       string           `json:"id"`
	Start    PersonioDate     `json:"start"`
	End      PersonioDate     `json:"end"`
	AType    string           `json:"type"`
	Approval PersonioApproval `json:"approval"`
}

type PersonioPersonResponse struct {
	Data []PersonioPerson                  `json:"_data"`
	Meta map[string]map[string]interface{} `json:"_meta"`
}

type PersonioAttendanceResponse struct {
	Data []PersonioAttendance              `json:"_data"`
	Meta map[string]map[string]interface{} `json:"_meta"`
}

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
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info(string(body))
	token := PersonioToken{}
	eve.Logger.Info(json.Unmarshal(body, &token))
	return token
}

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
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	person := PersonioPerson{}
	err = json.Unmarshal(body, &person)
	if err != nil {
		eve.Logger.Info(err)
	}
	return person
}

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
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info("PersonioAttendances", string(body))
	return nil
}

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
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info("PersonioAttendances", string(body))
	return []PersonioAttendance{}
}

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
		eve.Logger.Info(attendancePeriod.Id, ": ", attendancePeriod.Start, "=>", attendancePeriod.End, ">", attendancePeriod.AType, ">", attendancePeriod.Approval.Status)
	}
	return nil
}

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
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	eve.Logger.Info(string(body))
	return nil
}

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
