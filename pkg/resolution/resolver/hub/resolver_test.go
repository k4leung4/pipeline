/*
Copyright 2022 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package hub

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	resolutioncommon "github.com/tektoncd/pipeline/pkg/resolution/common"
	frtesting "github.com/tektoncd/pipeline/pkg/resolution/resolver/framework/testing"
	"github.com/tektoncd/pipeline/test/diff"
)

func TestGetSelector(t *testing.T) {
	resolver := Resolver{}
	sel := resolver.GetSelector(resolverContext())
	if typ, has := sel[resolutioncommon.LabelKeyResolverType]; !has {
		t.Fatalf("unexpected selector: %v", sel)
	} else if typ != LabelValueHubResolverType {
		t.Fatalf("unexpected type: %q", typ)
	}
}

func TestValidateParams(t *testing.T) {
	resolver := Resolver{}

	paramsWithTask := map[string]string{
		ParamKind:    "task",
		ParamName:    "foo",
		ParamVersion: "bar",
		ParamCatalog: "baz",
	}
	if err := resolver.ValidateParams(resolverContext(), toParams(paramsWithTask)); err != nil {
		t.Fatalf("unexpected error validating params: %v", err)
	}

	paramsWithPipeline := map[string]string{
		ParamKind:    "pipeline",
		ParamName:    "foo",
		ParamVersion: "bar",
		ParamCatalog: "baz",
	}
	if err := resolver.ValidateParams(resolverContext(), toParams(paramsWithPipeline)); err != nil {
		t.Fatalf("unexpected error validating params: %v", err)
	}
}

func TestValidateParamsDisabled(t *testing.T) {
	resolver := Resolver{}

	var err error

	params := map[string]string{
		ParamKind:    "task",
		ParamName:    "foo",
		ParamVersion: "bar",
		ParamCatalog: "baz",
	}
	err = resolver.ValidateParams(context.Background(), toParams(params))
	if err == nil {
		t.Fatalf("expected missing name err")
	}

	if d := cmp.Diff(disabledError, err.Error()); d != "" {
		t.Errorf("unexpected error: %s", diff.PrintWantGot(d))
	}
}

func TestValidateParamsMissing(t *testing.T) {
	resolver := Resolver{}

	var err error

	paramsMissingName := map[string]string{
		ParamKind:    "foo",
		ParamVersion: "bar",
	}
	err = resolver.ValidateParams(resolverContext(), toParams(paramsMissingName))
	if err == nil {
		t.Fatalf("expected missing name err")
	}

	paramsMissingVersion := map[string]string{
		ParamKind: "foo",
		ParamName: "bar",
	}
	err = resolver.ValidateParams(resolverContext(), toParams(paramsMissingVersion))
	if err == nil {
		t.Fatalf("expected missing version err")
	}
}

func TestValidateParamsConflictingKindName(t *testing.T) {
	resolver := Resolver{}
	params := map[string]string{
		ParamKind:    "not-taskpipeline",
		ParamName:    "foo",
		ParamVersion: "bar",
		ParamCatalog: "baz",
	}
	err := resolver.ValidateParams(resolverContext(), toParams(params))
	if err == nil {
		t.Fatalf("expected err due to conflicting kind param")
	}
}

func TestResolveDisabled(t *testing.T) {
	resolver := Resolver{}

	var err error

	params := map[string]string{
		ParamKind:    "task",
		ParamName:    "foo",
		ParamVersion: "bar",
		ParamCatalog: "baz",
	}
	_, err = resolver.Resolve(context.Background(), toParams(params))
	if err == nil {
		t.Fatalf("expected missing name err")
	}

	if d := cmp.Diff(disabledError, err.Error()); d != "" {
		t.Errorf("unexpected error: %s", diff.PrintWantGot(d))
	}
}

func TestResolve(t *testing.T) {
	testCases := []struct {
		name        string
		kind        string
		imageName   string
		version     string
		catalog     string
		input       string
		expectedRes []byte
		expectedErr error
	}{
		{
			name:        "valid response from hub",
			kind:        "task",
			imageName:   "foo",
			version:     "baz",
			catalog:     "tekton",
			input:       `{"data":{"yaml":"some content"}}`,
			expectedRes: []byte("some content"),
		},
		{
			name:        "not-found response from hub",
			kind:        "task",
			imageName:   "foo",
			version:     "baz",
			catalog:     "tekton",
			input:       `{"name":"not-found","id":"aaaaaaaa","message":"resource not found","temporary":false,"timeout":false,"fault":false}`,
			expectedRes: []byte(""),
		},
		{
			name:        "response with bad formatting error",
			kind:        "task",
			imageName:   "foo",
			version:     "baz",
			catalog:     "tekton",
			input:       `value`,
			expectedErr: fmt.Errorf("error unmarshalling json response: invalid character 'v' looking for beginning of value"),
		},
		{
			name:        "response with empty body error",
			kind:        "task",
			imageName:   "foo",
			version:     "baz",
			catalog:     "tekton",
			expectedErr: fmt.Errorf("error unmarshalling json response: unexpected end of JSON input"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, tc.input)
			}))

			resolver := &Resolver{HubURL: svr.URL + "/" + YamlEndpoint}

			params := map[string]string{
				ParamKind:    tc.kind,
				ParamName:    tc.imageName,
				ParamVersion: tc.version,
				ParamCatalog: tc.catalog,
			}

			output, err := resolver.Resolve(resolverContext(), toParams(params))
			if tc.expectedErr != nil {
				if err == nil {
					t.Fatalf("expected err '%v' but didn't get one", tc.expectedErr)
				}
				if d := cmp.Diff(tc.expectedErr.Error(), err.Error()); d != "" {
					t.Fatalf("expected err '%v' but got '%v'", tc.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error resolving: %v", err)
				}

				expectedResource := &ResolvedHubResource{
					Content: tc.expectedRes,
				}

				if d := cmp.Diff(expectedResource, output); d != "" {
					t.Errorf("unexpected resource from Resolve: %s", diff.PrintWantGot(d))
				}
			}
		})
	}
}

func resolverContext() context.Context {
	return frtesting.ContextWithHubResolverEnabled(context.Background())
}

func toParams(m map[string]string) []pipelinev1beta1.Param {
	var params []pipelinev1beta1.Param

	for k, v := range m {
		params = append(params, pipelinev1beta1.Param{
			Name:  k,
			Value: *pipelinev1beta1.NewStructuredValues(v),
		})
	}

	return params
}
