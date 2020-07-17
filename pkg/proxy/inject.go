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
	"fmt"
	"log"
	"strings"

	promlabels "github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/promql/parser"
	promparser "github.com/prometheus/prometheus/promql/parser"
)

func setRecursive(node promparser.Node, nsLabelname string, namespaces []string) (err error) {
	switch n := node.(type) {
	case *parser.EvalStmt:
		if err := setRecursive(n.Expr, nsLabelname, namespaces); err != nil {
			return err
		}

	case parser.Expressions:
		for _, e := range n {
			if err := setRecursive(e, nsLabelname, namespaces); err != nil {
				return err
			}
		}
	case *parser.AggregateExpr:
		if err := setRecursive(n.Expr, nsLabelname, namespaces); err != nil {
			return err
		}

	case *parser.BinaryExpr:
		if err := setRecursive(n.LHS, nsLabelname, namespaces); err != nil {
			return err
		}
		if err := setRecursive(n.RHS, nsLabelname, namespaces); err != nil {
			return err
		}

	case *parser.Call:
		if err := setRecursive(n.Args, nsLabelname, namespaces); err != nil {
			return err
		}

	case *parser.ParenExpr:
		if err := setRecursive(n.Expr, nsLabelname, namespaces); err != nil {
			return err
		}

	case *parser.UnaryExpr:
		if err := setRecursive(n.Expr, nsLabelname, namespaces); err != nil {
			return err
		}

	case *parser.SubqueryExpr:
		if err := setRecursive(n.Expr, nsLabelname, namespaces); err != nil {
			return err
		}

	case *parser.NumberLiteral, *parser.StringLiteral:
		// nothing to do

	case *parser.MatrixSelector:
		// inject labelselector
		if vs, ok := n.VectorSelector.(*parser.VectorSelector); ok {
			vs.LabelMatchers = enforceLabelMatcher(vs.LabelMatchers, nsLabelname, namespaces)
		}

	case *parser.VectorSelector:
		// inject labelselector
		n.LabelMatchers = enforceLabelMatcher(n.LabelMatchers, nsLabelname, namespaces)

	default:
		panic(fmt.Errorf("promql.Walk: unhandled node type %T", node))
	}

	return err
}

//This is where really injection is done.
//It limits original query's namespace matcher
//1. no namespace mather at all
//2. use Equal matcher
//3. use MatchRegexp but can only be simple pattern "name1|name2|name3". there should be no any kind of wildcard in names
//otherwise it will return empty data by injecting noDataMatcher
func enforceLabelMatcher(matchers []*promlabels.Matcher, nsLabelname string, namespaces []string) []*promlabels.Matcher {
	res := []*promlabels.Matcher{}
	var nsMatcher *promlabels.Matcher
	noDataMatcher := &promlabels.Matcher{
		Type:  promlabels.MatchEqual,
		Name:  nsLabelname,
		Value: "__ibm-ocpthanos-proxy-no-data-namespace__",
	}
	for _, m := range matchers {
		if m.Name == nsLabelname {
			nsMatcher = m
			continue
		}
		res = append(res, m)
	}

	if nsMatcher == nil {
		//create namespace matcher if raw expression does not containe one
		nsMatcher = &promlabels.Matcher{
			Type:  promlabels.MatchRegexp,
			Name:  nsLabelname,
			Value: generateRegExpr(namespaces),
		}
		return append(res, nsMatcher)
	}
	//combine to existing namespace selector
	if nsMatcher.Type == promlabels.MatchEqual {
		allowed := false
		for _, ns := range namespaces {
			if ns == nsMatcher.Value {
				allowed = true
			}
		}
		if allowed {
			return append(res, nsMatcher)
		}

	}
	//namespace matcher value in query string is assumed in the format: 'name1|name2|name3'
	//if any one (name1, name2, name3) is not accessible to user, noDataMatcher will be injected
	if nsMatcher.Type == promlabels.MatchRegexp {
		allowed := true
		origNamespaces := strings.Split(nsMatcher.Value, "|")
		for _, ons := range origNamespaces {
			found := false
			for _, ns := range namespaces {
				if ons == ns {
					found = true
				}
			}
			if !found {
				allowed = false
				break
			}
		}
		if allowed {
			return append(res, nsMatcher)
		}

	}

	log.Printf("no data matcher is injected query. namespace matcher in query: %s. allowed namespaces: %s",
		nsMatcher.Value, strings.Join(namespaces, ","))
	return append(res, noDataMatcher)

}
func generateRegExpr(namespaces []string) string {

	for i, ns := range namespaces {
		namespaces[i] = "^" + ns + "$"
	}
	return strings.Join(namespaces, "|")

}
