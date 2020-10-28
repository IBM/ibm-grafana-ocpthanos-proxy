//
// Copyright 2020 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package nsparser

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

/**********************************************
***** NSParser implementation: type definitions
***********************************************/
type ibmCommonServiceNSParser struct {
	uidURL      string
	userInfoURL string
	client      *http.Client
}
type nsListParser struct {
	namespaces []string
}

/**********************************************
***** NSParser implementation: interface methods
***********************************************/

//ParseNamespaces get the namespaces for the request
func (p *nsListParser) ParseNamespaces(req *http.Request) ([]string, error) {
	copied := make([]string, len(p.namespaces))
	copy(copied, p.namespaces)
	return copied, nil
}

//ParseNamespaces get the namespaces for the request
func (p *ibmCommonServiceNSParser) ParseNamespaces(req *http.Request) ([]string, error) {
	p.init()
	token, err := p.getToken(req)
	if err != nil {
		return []string{}, err
	}
	var uid string
	uid, err = p.getUserID(token)
	if err != nil {
		return []string{}, err
	}
	var namespaces []string
	namespaces, err = p.getUserNamespaces(token, uid)
	if err != nil {
		return []string{}, err
	}
	return namespaces, nil
}

/**********************************************
***** NSParser implementation: helper methods
***********************************************/

func (p *ibmCommonServiceNSParser) getUserID(token string) (string, error) {
	targetURL := p.uidURL + "/v1/auth/userInfo"

	formData := url.Values{"access_token": {token}}
	formBody := strings.NewReader(formData.Encode())

	req, err := http.NewRequest(
		http.MethodPost,
		targetURL,
		formBody)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		return "", err
	}

	var resp *http.Response
	if resp, err = p.client.Do(req); err != nil {
		return "", err
	}
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("Status:" + resp.Status)
	}
	respbytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var respObj map[string]interface{}

	err = json.Unmarshal(respbytes, &respObj)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	return respObj["sub"].(string), nil
}
func (p *ibmCommonServiceNSParser) queryUserInfo(url string, token string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	var resp *http.Response
	if resp, err = p.client.Do(req); err != nil {
		return "", err
	}
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("Status:" + resp.Status)
	}
	respbytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	respString := string(respbytes)
	return strings.Trim(respString, "\""), nil
}
func (p *ibmCommonServiceNSParser) getUserNamespaces(token string, uid string) ([]string, error) {
	//get namespaces accessible to the user
	nsURL := p.userInfoURL + "/identity/api/v1/users/" + uid + "/getTeamResources" + "?resourceType=namespace"
	nsRawString, err := p.queryUserInfo(nsURL, token)
	if err != nil {
		return []string{}, fmt.Errorf("failed to get namespaces accessible to user. details: " + err.Error())
	}
	var nsObj []map[string]string
	if err := json.Unmarshal([]byte(nsRawString), &nsObj); err != nil {
		return []string{}, fmt.Errorf("failed to parse common service IAM service response")
	}
	var namespaces []string
	for _, ns := range nsObj {
		namespaces = append(namespaces, ns["namespaceId"])
	}
	var role string
	if len(namespaces) > 0 {
		role = nsObj[0]["highestRole"]

	}
	if role == "ClusterAdministrator" {
		return []string{AllNamespaces}, nil
	}
	if role == "" {
		return namespaces, fmt.Errorf("no namespace accessible to user")
	}

	return namespaces, nil

}

func (p *ibmCommonServiceNSParser) getToken(req *http.Request) (string, error) {
	cookie, err := req.Cookie("cfc-access-token-cookie")
	if err == nil {
		return cookie.Value, err
	}
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("failed to get token")
	}
	if len(authHeader) < len("Bearer ") {
		return "", fmt.Errorf("no token")
	}
	token := strings.Split(authHeader, "Bearer ")[1]
	return token, nil
}

func (p *ibmCommonServiceNSParser) init() {
	if p.client != nil {
		return
	}
	// create Client. it is http.DefaultTransport with extra tls Config
	tlsConfig := &tls.Config{
		//nolint:gosec
		InsecureSkipVerify: true,
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       tlsConfig,
	}
	p.client = &http.Client{Transport: transport}

}
