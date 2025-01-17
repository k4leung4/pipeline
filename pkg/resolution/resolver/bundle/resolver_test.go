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

package bundle

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	resolutioncommon "github.com/tektoncd/pipeline/pkg/resolution/common"
	frtesting "github.com/tektoncd/pipeline/pkg/resolution/resolver/framework/testing"
	"github.com/tektoncd/pipeline/test/diff"
)

func TestGetSelector(t *testing.T) {
	resolver := Resolver{}
	sel := resolver.GetSelector(context.Background())
	if typ, has := sel[resolutioncommon.LabelKeyResolverType]; !has {
		t.Fatalf("unexpected selector: %v", sel)
	} else if typ != LabelValueBundleResolverType {
		t.Fatalf("unexpected type: %q", typ)
	}
}

func TestValidateParams(t *testing.T) {
	resolver := Resolver{}

	paramsWithTask := []pipelinev1beta1.Param{{
		Name:  ParamKind,
		Value: *pipelinev1beta1.NewStructuredValues("task"),
	}, {
		Name:  ParamName,
		Value: *pipelinev1beta1.NewStructuredValues("foo"),
	}, {
		Name:  ParamBundle,
		Value: *pipelinev1beta1.NewStructuredValues("bar"),
	}, {
		Name:  ParamServiceAccount,
		Value: *pipelinev1beta1.NewStructuredValues("baz"),
	}}

	if err := resolver.ValidateParams(resolverContext(), paramsWithTask); err != nil {
		t.Fatalf("unexpected error validating params: %v", err)
	}

	paramsWithPipeline := []pipelinev1beta1.Param{{
		Name:  ParamKind,
		Value: *pipelinev1beta1.NewStructuredValues("pipeline"),
	}, {
		Name:  ParamName,
		Value: *pipelinev1beta1.NewStructuredValues("foo"),
	}, {
		Name:  ParamBundle,
		Value: *pipelinev1beta1.NewStructuredValues("bar"),
	}, {
		Name:  ParamServiceAccount,
		Value: *pipelinev1beta1.NewStructuredValues("baz"),
	}}
	if err := resolver.ValidateParams(resolverContext(), paramsWithPipeline); err != nil {
		t.Fatalf("unexpected error validating params: %v", err)
	}
}

func TestValidateParamsDisabled(t *testing.T) {
	resolver := Resolver{}

	var err error

	params := []pipelinev1beta1.Param{{
		Name:  ParamKind,
		Value: *pipelinev1beta1.NewStructuredValues("task"),
	}, {
		Name:  ParamName,
		Value: *pipelinev1beta1.NewStructuredValues("foo"),
	}, {
		Name:  ParamBundle,
		Value: *pipelinev1beta1.NewStructuredValues("bar"),
	}, {
		Name:  ParamServiceAccount,
		Value: *pipelinev1beta1.NewStructuredValues("baz"),
	}}
	err = resolver.ValidateParams(context.Background(), params)
	if err == nil {
		t.Fatalf("expected disabled err")
	}

	if d := cmp.Diff(disabledError, err.Error()); d != "" {
		t.Errorf("unexpected error: %s", diff.PrintWantGot(d))
	}
}

func TestValidateParamsMissing(t *testing.T) {
	resolver := Resolver{}

	var err error

	paramsMissingBundle := []pipelinev1beta1.Param{{
		Name:  ParamKind,
		Value: *pipelinev1beta1.NewStructuredValues("task"),
	}, {
		Name:  ParamName,
		Value: *pipelinev1beta1.NewStructuredValues("foo"),
	}, {
		Name:  ParamServiceAccount,
		Value: *pipelinev1beta1.NewStructuredValues("baz"),
	}}
	err = resolver.ValidateParams(resolverContext(), paramsMissingBundle)
	if err == nil {
		t.Fatalf("expected missing kind err")
	}

	paramsMissingName := []pipelinev1beta1.Param{{
		Name:  ParamKind,
		Value: *pipelinev1beta1.NewStructuredValues("task"),
	}, {
		Name:  ParamBundle,
		Value: *pipelinev1beta1.NewStructuredValues("bar"),
	}, {
		Name:  ParamServiceAccount,
		Value: *pipelinev1beta1.NewStructuredValues("baz"),
	}}
	err = resolver.ValidateParams(resolverContext(), paramsMissingName)
	if err == nil {
		t.Fatalf("expected missing name err")
	}

}

func TestResolveDisabled(t *testing.T) {
	resolver := Resolver{}

	var err error

	params := []pipelinev1beta1.Param{{
		Name:  ParamKind,
		Value: *pipelinev1beta1.NewStructuredValues("task"),
	}, {
		Name:  ParamName,
		Value: *pipelinev1beta1.NewStructuredValues("foo"),
	}, {
		Name:  ParamBundle,
		Value: *pipelinev1beta1.NewStructuredValues("bar"),
	}, {
		Name:  ParamServiceAccount,
		Value: *pipelinev1beta1.NewStructuredValues("baz"),
	}}
	_, err = resolver.Resolve(context.Background(), params)
	if err == nil {
		t.Fatalf("expected disabled err")
	}

	if d := cmp.Diff(disabledError, err.Error()); d != "" {
		t.Errorf("unexpected error: %s", diff.PrintWantGot(d))
	}
}

func resolverContext() context.Context {
	return frtesting.ContextWithBundlesResolverEnabled(context.Background())
}
