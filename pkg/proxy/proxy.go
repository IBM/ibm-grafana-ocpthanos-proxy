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
	"log"
	"net"
	"net/http"
	"net/url"

	"github.com/IBM/ibm-grafana-ocpthanos-proxy/pkg/nsparser"
)

//StartAndServe start HTTP server and forward request to backend server
func StartAndServe(listenAddr string,
	urlPrefix string,
	thanosAddr string,
	thanosTokenFile string,
	nsparser nsparser.NSParser,
	nsLabelName string,
	errCh chan<- error) (*http.Server, error) {
	url, err := url.Parse(thanosAddr)
	if err != nil {
		return nil, err
	}
	// create handlers
	routes := &routes{
		thanosURL:       url,
		thanosTokenFile: thanosTokenFile,
		nsparser:        nsparser,
		nsLabelName:     nsLabelName,
	}
	routes.init()
	mux := http.NewServeMux()
	mux.Handle(urlPrefix, routes)
	// create server
	server := &http.Server{Handler: mux}
	l, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, err
	}
	go func() {
		log.Printf("Listening insecurely on %v", l.Addr())
		errCh <- server.Serve(l)
	}()
	return server, nil
}
