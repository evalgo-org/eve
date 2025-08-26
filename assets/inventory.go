package assets

import (
	// "fmt"
	"io"
	"net/http"

	eve "eve.evalgo.org/common"
)

func InvComponents(url string, token string) string {
	tgt_url := url + "/api/v1/components?limit=50&offset=0&order_number=null&sort=created_at&order=desc&expand=false"
	req, _ := http.NewRequest("GET", tgt_url, nil)
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	return string(body)
}
