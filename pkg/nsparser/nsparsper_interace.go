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
	"io/ioutil"
	"log"
	"net/http"

	"github.com/ghodss/yaml"
)

//NSParserType defines type of namespace parser
type NSParserType string

//NSParserTypeBedrock this namespace parser gets namespaces by querying IBM Bedrock IAM service
const NSParserTypeBedrock NSParserType = "ibm-bedrock-iam"

//NSParserTypeNSList use this kind of namespace parser, user can configure namespace list in configuration file
const NSParserTypeNSList NSParserType = "ns-list"

//AllNamespaces means user can access all namespaces
const AllNamespaces = "ALL"

//NSParser define interface for namespace parser
type NSParser interface {
	ParseNamespaces(req *http.Request) ([]string, error)
}

//NewNSParser create NSParser instance according to configration file.
//The configuration file should be in format:
//type: typename
//paras:
//  pname1: pvalue1
//  pname2: pvalue2
//only ibm-bedrock type of parser is implemented for now
func NewNSParser(cfgFile string) NSParser {
	b, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		log.Fatalf("failed to parse namespace parser configuration file: " + cfgFile)
		log.Printf(err.Error())
		return nil
	}
	var cfg map[string]interface{}
	err = yaml.Unmarshal(b, &cfg)
	if err != nil {
		return nil
	}
	ptype, ok := cfg["type"]
	if !ok {
		log.Fatalf("something is wrong in namespace parser configuration file: " + cfgFile)
		return nil
	}
	switch NSParserType(ptype.(string)) {
	case NSParserTypeBedrock:
		paras, ok := cfg["paras"].(map[string]interface{})
		if !ok {
			log.Fatalf("something is wrong in namespace parser configuration file: " + cfgFile)
			return nil
		}
		parser := ibmBedrockNSParser{
			uidURL:      paras["uidURL"].(string),
			userInfoURL: paras["userInfoURL"].(string),
		}
		log.Printf("namespace parser created. type: " + string(NSParserTypeBedrock))
		return &parser
	case NSParserTypeNSList:
		paras, ok := cfg["paras"].(map[string]interface{})
		if !ok {
			log.Fatalf("something is wrong in namespace parser configuration file: " + cfgFile)
			return nil
		}
		objects := paras["namespaces"].([]interface{})
		namespaces := []string{}
		for _, ns := range objects {
			if ns != "" {
				namespaces = append(namespaces, ns.(string))
			}

		}
		parser := nsListParser{namespaces}
		log.Printf("namespace parser created. type: " + string(NSParserTypeNSList))
		return &parser

	default:
		log.Fatalf("unsupported namespace parser: " + ptype.(string))
		return nil
	}
}
