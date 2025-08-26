package network

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	sdk_golang "github.com/openziti/sdk-golang"
	"github.com/openziti/sdk-golang/ziti"

	eve "eve.evalgo.org/common"
)

type ZitiServiceConfig struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ZitiServiceConfigsResult struct {
	Data []ZitiServiceConfig `json:"data"`
}

type ZitiToken struct {
	Token string `json:"token"`
	ID    string `json:"id"`
}

type ZitiResult struct {
	Data ZitiToken `json:"data"`
}

func ZitiClient(id string) *http.Client {
	cfg, _ := ziti.NewConfigFromFile(id)
	ctx, _ := ziti.NewContext(cfg)
	return sdk_golang.NewHttpClient(ctx, nil)
}

func postWithAuthMap(url, token string, payload map[string]interface{}) (string, error) {
	eve.Logger.Info(url, " <> ", token)
	data, _ := json.Marshal(payload)
	eve.Logger.Info(string(data))
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	// req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Zt-Session", token)
	// req.Header.Set("Authorization", "Bearer " + token)
	var client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	eve.Logger.Info(resp.StatusCode)
	if resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			eve.Logger.Fatal(err)
		}
		eve.Logger.Info(string(body))
		return "", errors.New(resp.Status)
	}
	result := ZitiResult{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Data.ID, nil
}

func ZitiAuthenticate(url, user, pass string) (string, error) {
	payload := map[string]string{
		"username": user,
		"password": pass,
	}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url+"/authenticate?method=password", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	// req.Header.Set("Authorization", "Bearer "+token)
	var client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result ZitiResult
	// body, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	eve.Logger.Fatal(err)
	// }
	json.NewDecoder(resp.Body).Decode(&result)
	// eve.Logger.Info(result)
	return result.Data.Token, nil
}

func ZitiCreateService(url, token, name, hostV1, interceptV1 string) (string, error) {
	body := map[string]interface{}{
		"configs":            []string{hostV1, interceptV1},
		"encryptionRequired": true,
		"name":               name,
	}
	return postWithAuthMap(url+"/edge/management/v1/services", token, body)
}

func ZitiCreateServicePolicy(url, token, name, policyType, serviceID, identity string) (string, error) {
	body := map[string]interface{}{
		"name":          name,
		"type":          policyType,
		"identityRoles": []string{identity},
		"serviceRoles":  []string{serviceID},
		"semantic":      "AnyOf",
	}
	pID, err := postWithAuthMap(url+"/edge/management/v1/service-policies", token, body)
	return pID, err
}

func ZitiCreateServiceConfig(url, token, name, configType string, config map[string]interface{}) (string, error) {
	payload := map[string]interface{}{
		"name":         name,
		"configTypeId": configType,
		"data":         config,
	}
	eve.Logger.Info(payload)
	confID, err := postWithAuthMap(url+"/edge/management/v1/configs", token, payload)
	return confID, err
}

func ZitiCreateEdgeRouterPolicy(url, token, name string, routers, services, roles []string) error {
	body := map[string]interface{}{
		"name":            name,
		"edgeRouterRoles": routers,
		"serviceRoles":    services,
		"semantic":        "AllOf",
		"permissions":     roles,
	}
	_, err := postWithAuthMap(url+"/edge/v1/service-edge-router-policies", token, body)
	return err
}

func ZitiGetConfigTypes(url, token, name string) (string, error) {
	req, _ := http.NewRequest("GET", url+"/edge/management/v1/config-types", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Zt-Session", token)
	var client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	// body, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	eve.Logger.Fatal(err)
	// }
	var result ZitiServiceConfigsResult
	json.NewDecoder(resp.Body).Decode(&result)
	for _, conf := range result.Data {
		if conf.Name == name {
			return conf.ID, nil
		}
	}
	return "", errors.New("could not find config: " + name)
}

func ZitiServicePolicies(url, token string) {
	req, _ := http.NewRequest("GET", url+"/edge/management/v1/service-policies", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Zt-Session", token)
	var client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		eve.Logger.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		eve.Logger.Fatal(err)
	}
	// var result ZitiServiceConfigsResult
	// json.NewDecoder(resp.Body).Decode(&result)
	// for _,conf := range result.Data {
	// 	if conf.Name == name {
	// 		return conf.ID, nil
	// 	}
	// }
	// return "", errors.New("could not find config: " + name)
	eve.Logger.Info(string(body))
}

func ZitiIdentities(urlSrc, token string) {
	q := url.Values{}
	q.Add("limit", "10000")
	tgtURL := urlSrc + "/edge/management/v1/identities"
	parsedURL, err := url.Parse(tgtURL)
	if err != nil {
		panic(err)
	}
	parsedURL.RawQuery = q.Encode()
	req, _ := http.NewRequest("GET", parsedURL.String(), nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Zt-Session", token)
	var client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		eve.Logger.Fatal(err)
	}
	defer resp.Body.Close()
	// body, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	eve.Logger.Fatal(err)
	// }
	var result ZitiServiceConfigsResult
	json.NewDecoder(resp.Body).Decode(&result)
	for _, conf := range result.Data {
		eve.Logger.Info(conf.ID, " <> ", conf.Name)
		// if conf.Name == name {
		// 	return conf.ID, nil
		// }
	}
	// return "", errors.New("could not find config: " + name)
	// eve.Logger.Info(string(body))
}

func ZitiGetIdentity(urlSrc, token, name string) (string, error) {
	q := url.Values{}
	q.Add("limit", "10000")
	tgtURL := urlSrc + "/edge/management/v1/identities"
	parsedURL, err := url.Parse(tgtURL)
	if err != nil {
		return "", err
	}
	parsedURL.RawQuery = q.Encode()
	req, _ := http.NewRequest("GET", parsedURL.String(), nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Zt-Session", token)
	var client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result ZitiServiceConfigsResult
	json.NewDecoder(resp.Body).Decode(&result)
	for _, ident := range result.Data {
		if ident.Name == name {
			return ident.ID, nil
		}
	}
	return "", errors.New("could not find identity with the name: " + name)
}
