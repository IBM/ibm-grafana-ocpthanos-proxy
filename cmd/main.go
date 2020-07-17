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

package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/IBM/ibm-grafana-ocpthanos-proxy/pkg/nsparser"
	"github.com/IBM/ibm-grafana-ocpthanos-proxy/pkg/proxy"
)

type config struct {
	listeningAddr   string
	urlPrefix       string
	thanosAddr      string
	nsParserConf    string
	thanosTokenFile string
	nsLabelName     string
}

func main() {
	cfg := config{}
	flagset := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagset.StringVar(&cfg.listeningAddr, "listen-address", "127.0.0.1:9096", "The address ibm-grafana-ocpthanos-proxy should listen on.")
	flagset.StringVar(&cfg.urlPrefix, "url-prefix", "/", "url prefix of the proxy")
	flagset.StringVar(&cfg.thanosAddr, "thanos-address", "https://thanos-querier.openshift-monitoring.svc:9091", "The address of thanos-querier service")
	flagset.StringVar(&cfg.nsParserConf, "ns-parser-conf", "/etc/conf/ns-config.yaml", "NSParser configurate file location")
	flagset.StringVar(&cfg.thanosTokenFile, "thanos-token-file", "/var/run/secrets/kubernetes.io/serviceaccount/token", "The token file passed to OCP thanos-querier service for authentication")
	flagset.StringVar(&cfg.nsLabelName, "ns-label-name", "namespace", "The name of metrics' namespace label")

	flagset.Parse(os.Args[1:])

	nsparser := nsparser.NewNSParser(cfg.nsParserConf)
	if nsparser == nil {
		os.Exit(1)
	}
	errCh := make(chan error)
	server, err := proxy.StartAndServe(cfg.listeningAddr, cfg.urlPrefix, cfg.thanosAddr, cfg.thanosTokenFile, nsparser, cfg.nsLabelName, errCh)
	if err != nil {
		os.Exit(1)
	}
	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)
	select {
	case <-term:
		log.Print("Received SIGTERM, exiting gracefully...")
		server.Close()
	case err := <-errCh:
		if err != http.ErrServerClosed {
			log.Printf("Server stopped with %v", err)
		}
		os.Exit(1)
	}

}
