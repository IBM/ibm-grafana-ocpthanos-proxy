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

package proxy

import (
	"crypto/tls"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	promparser "github.com/prometheus/prometheus/promql/parser"

	"github.com/IBM/ibm-grafana-ocpthanos-proxy/pkg/nsparser"
)

type routes struct {
	//handler is instance of httputil.ReverseProxy
	handler http.Handler
	mux     *http.ServeMux

	nsparser        nsparser.NSParser
	nsLabelName     string
	thanosTokenFile string
	thanosURL       *url.URL
}

func (r *routes) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	//serve Get method only
	if req.Method != http.MethodGet {
		http.NotFound(w, req)
		return
	}
	r.mux.ServeHTTP(w, req)
}

//1. custom httputil.ReverseProxy
//2. register handler functions
func (r *routes) init() {
	proxy := httputil.NewSingleHostReverseProxy(r.thanosURL)
	// it is http.DefaultTransport with extra tls Config
	if r.thanosURL.Scheme == "https" {
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
		proxy.Transport = transport
	}

	//modify default director to add thanos access token and update req.Host
	director := func(req *http.Request) {
		req.URL.Scheme = r.thanosURL.Scheme
		req.URL.Host = r.thanosURL.Host
		//update Host so that it can pass OCP Router when testing locally
		req.Host = r.thanosURL.Host
		targetQuery := r.thanosURL.RawQuery
		req.URL.Path = singleJoiningSlash(r.thanosURL.Path, req.URL.Path)
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
		//add Authorization heander for token
		tokenBytes, err := ioutil.ReadFile(r.thanosTokenFile)
		if err != nil {
			tokenBytes = []byte("")
		}
		thanosToken := string(tokenBytes)
		req.Header.Set("Authorization", "Bearer "+thanosToken)
	}
	proxy.Director = director
	r.handler = proxy

	//add handler for different endpoints to meet requirements from Grafana
	mux := http.NewServeMux()
	r.mux = mux
	mux.Handle("/api/v1/query", r.wrapMethod(r.query))
	mux.Handle("/api/v1/query_range", r.wrapMethod(r.query))
	mux.Handle("/api/v1/series", r.wrapMethod(r.query))
	mux.Handle("/api/v1/label/", r.wrapMethod(r.forward))

}

//query injects namespaces into PromQL query string
func (r *routes) query(w http.ResponseWriter, req *http.Request) {
	namespaces, err := r.nsparser.ParseNamespaces(req)
	if err != nil {
		http.Error(w, "No namespace accessible for user. details: "+err.Error(), http.StatusForbidden)
		return

	}
	if len(namespaces) == 0 {
		http.Error(w, "No namespace accessible for user.", http.StatusForbidden)
		return
	}
	for _, ns := range namespaces {
		if ns == nsparser.AllNamespaces {
			r.handler.ServeHTTP(w, req)
			return
		}
	}
	var queryKey string
	var query string
	if query = req.FormValue("query"); query != "" {
		queryKey = "query"
	} else if query = req.FormValue("match[]"); query != "" {
		queryKey = "match[]"
	}
	if err != nil {
		http.Error(w, "failed to parse namespaces. Details: "+err.Error(), http.StatusBadRequest)
		return
	}
	expr, err := promparser.ParseExpr(query)
	if err != nil {
		http.Error(w, "failed to parse query string", http.StatusBadRequest)
		return
	}
	err = setRecursive(expr, r.nsLabelName, namespaces)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	updatedQuery := expr.String()
	q := req.URL.Query()
	q.Set(queryKey, updatedQuery)
	req.URL.RawQuery = q.Encode()
	r.handler.ServeHTTP(w, req)
}

//wrap our function as http handler
func (r *routes) wrapMethod(h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		h(w, req)
	})
}

//forward forwards request directly without any change
func (r *routes) forward(w http.ResponseWriter, req *http.Request) {
	r.handler.ServeHTTP(w, req)
}

//copied from httputil for customizing Director of http.ReverseProxy
func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
