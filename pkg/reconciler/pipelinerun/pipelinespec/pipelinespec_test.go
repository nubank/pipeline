/*
Copyright 2019 The Tekton Authors

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

package pipelinespec

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	cfgtesting "github.com/tektoncd/pipeline/pkg/apis/config/testing"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test/diff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetPipelineSpec_Ref(t *testing.T) {
	pipeline := &v1beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name: "orchestrate",
		},
		Spec: v1beta1.PipelineSpec{
			Tasks: []v1beta1.PipelineTask{{
				Name: "mytask",
				TaskRef: &v1beta1.TaskRef{
					Name: "mytask",
				},
			}},
		},
	}
	pr := &v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mypipelinerun",
		},
		Spec: v1beta1.PipelineRunSpec{
			PipelineRef: &v1beta1.PipelineRef{
				Name: "orchestrate",
			},
		},
	}
	gt := func(ctx context.Context, n string) (*v1beta1.Pipeline, *v1beta1.RefSource, error) {
		return pipeline, nil, nil
	}
	resolvedObjectMeta, pipelineSpec, err := GetPipelineData(context.Background(), pr, gt)

	if err != nil {
		t.Fatalf("Did not expect error getting pipeline spec but got: %s", err)
	}

	if resolvedObjectMeta.Name != "orchestrate" {
		t.Errorf("Expected pipeline name to be `orchestrate` but was %q", resolvedObjectMeta.Name)
	}

	if len(pipelineSpec.Tasks) != 1 || pipelineSpec.Tasks[0].Name != "mytask" {
		t.Errorf("Pipeline Spec not resolved as expected, expected referenced Pipeline spec but got: %v", pipelineSpec)
	}

	if resolvedObjectMeta.RefSource != nil {
		t.Errorf("Expected resolved refSource is nil, but got %v", resolvedObjectMeta.RefSource)
	}
}

func TestGetPipelineSpec_Embedded(t *testing.T) {
	pr := &v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mypipelinerun",
		},
		Spec: v1beta1.PipelineRunSpec{
			PipelineSpec: &v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{{
					Name: "mytask",
					TaskRef: &v1beta1.TaskRef{
						Name: "mytask",
					},
				}},
			},
		},
	}
	gt := func(ctx context.Context, n string) (*v1beta1.Pipeline, *v1beta1.RefSource, error) {
		return nil, nil, errors.New("shouldn't be called")
	}
	resolvedObjectMeta, pipelineSpec, err := GetPipelineData(context.Background(), pr, gt)

	if err != nil {
		t.Fatalf("Did not expect error getting pipeline spec but got: %s", err)
	}

	if resolvedObjectMeta.Name != "mypipelinerun" {
		t.Errorf("Expected pipeline name for embedded pipeline to default to name of pipeline run but was %q", resolvedObjectMeta.Name)
	}

	if len(pipelineSpec.Tasks) != 1 || pipelineSpec.Tasks[0].Name != "mytask" {
		t.Errorf("Pipeline Spec not resolved as expected, expected embedded Pipeline spec but got: %v", pipelineSpec)
	}

	if resolvedObjectMeta.RefSource != nil {
		t.Errorf("Expected resolved refSource is nil, but got %v", resolvedObjectMeta.RefSource)
	}
}

func TestGetPipelineSpec_Invalid(t *testing.T) {
	tr := &v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mypipelinerun",
		},
	}
	gt := func(ctx context.Context, n string) (*v1beta1.Pipeline, *v1beta1.RefSource, error) {
		return nil, nil, errors.New("shouldn't be called")
	}
	_, _, err := GetPipelineData(context.Background(), tr, gt)
	if err == nil {
		t.Fatalf("Expected error resolving spec with no embedded or referenced pipeline spec but didn't get error")
	}
}

func TestGetPipelineData_ResolutionSuccess(t *testing.T) {
	sourceMeta := &metav1.ObjectMeta{
		Name: "pipeline",
	}
	refSource := &v1beta1.RefSource{
		URI:        "abc.com",
		Digest:     map[string]string{"sha1": "a123"},
		EntryPoint: "foo/bar",
	}

	tests := []struct {
		name         string
		pr           *v1beta1.PipelineRun
		sourceMeta   *metav1.ObjectMeta
		sourceSpec   *v1beta1.PipelineSpec
		refSource    *v1beta1.RefSource
		expectedSpec *v1beta1.PipelineSpec
		defaults     map[string]string
	}{
		{
			name:       "resolve remote task with taskRef Name",
			sourceMeta: sourceMeta,
			refSource:  refSource,
			pr: &v1beta1.PipelineRun{
				Spec: v1beta1.PipelineRunSpec{
					PipelineRef: &v1beta1.PipelineRef{
						ResolverRef: v1beta1.ResolverRef{
							Resolver: "foo",
						},
					},
				},
			},
			sourceSpec: &v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{{
					Name: "pt1",
					TaskRef: &v1beta1.TaskRef{
						Kind: "Task",
						Name: "tref",
					},
				}},
			},
			expectedSpec: &v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{{
					Name: "pt1",
					TaskRef: &v1beta1.TaskRef{
						Kind: "Task",
						Name: "tref",
					},
				}},
			},
		},
		{
			name:       "resolve remote task with taskRef resolver - default resolver configured",
			sourceMeta: sourceMeta,
			refSource:  refSource,
			pr: &v1beta1.PipelineRun{
				Spec: v1beta1.PipelineRunSpec{
					PipelineRef: &v1beta1.PipelineRef{
						ResolverRef: v1beta1.ResolverRef{
							Resolver: "foo",
						},
					},
				},
			},
			sourceSpec: &v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{{
					Name:    "pt1",
					TaskRef: &v1beta1.TaskRef{},
				}},
			},
			expectedSpec: &v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{{
					Name: "pt1",
					TaskRef: &v1beta1.TaskRef{
						Kind: "Task",
						ResolverRef: v1beta1.ResolverRef{
							Resolver: "foo",
						},
					},
				}},
			},
			defaults: map[string]string{
				"default-resolver-type": "foo",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := cfgtesting.SetDefaults(context.Background(), t, tc.defaults)
			getPipeline := func(ctx context.Context, n string) (*v1beta1.Pipeline, *v1beta1.RefSource, error) {
				return &v1beta1.Pipeline{
					ObjectMeta: *tc.sourceMeta.DeepCopy(),
					Spec:       *tc.sourceSpec.DeepCopy(),
				}, tc.refSource.DeepCopy(), nil
			}

			resolvedObjectMeta, resolvedPipelineSpec, err := GetPipelineData(ctx, tc.pr, getPipeline)
			if err != nil {
				t.Fatalf("did not expect error getting pipeline spec but got: %s", err)
			}

			if sourceMeta.Name != resolvedObjectMeta.Name {
				t.Errorf("expected name %q but resolved to %q", sourceMeta.Name, resolvedObjectMeta.Name)
			}
			if d := cmp.Diff(tc.refSource, resolvedObjectMeta.RefSource); d != "" {
				t.Errorf("refSource did not match: %s", diff.PrintWantGot(d))
			}
			if d := cmp.Diff(tc.expectedSpec, resolvedPipelineSpec); d != "" {
				t.Errorf("pipelineSpec did not match: %s", diff.PrintWantGot(d))
			}
		})
	}
}

func TestGetPipelineSpec_Error(t *testing.T) {
	tr := &v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mypipelinerun",
		},
		Spec: v1beta1.PipelineRunSpec{
			PipelineRef: &v1beta1.PipelineRef{
				Name: "orchestrate",
			},
		},
	}
	gt := func(ctx context.Context, n string) (*v1beta1.Pipeline, *v1beta1.RefSource, error) {
		return nil, nil, errors.New("something went wrong")
	}
	_, _, err := GetPipelineData(context.Background(), tr, gt)
	if err == nil {
		t.Fatalf("Expected error when unable to find referenced Pipeline but got none")
	}
}

func TestGetPipelineData_ResolutionError(t *testing.T) {
	pr := &v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mypipelinerun",
		},
		Spec: v1beta1.PipelineRunSpec{
			PipelineRef: &v1beta1.PipelineRef{
				ResolverRef: v1beta1.ResolverRef{
					Resolver: "git",
				},
			},
		},
	}
	getPipeline := func(ctx context.Context, n string) (*v1beta1.Pipeline, *v1beta1.RefSource, error) {
		return nil, nil, errors.New("something went wrong")
	}
	ctx := context.Background()
	_, _, err := GetPipelineData(ctx, pr, getPipeline)
	if err == nil {
		t.Fatalf("Expected error when unable to find referenced Pipeline but got none")
	}
}

func TestGetPipelineData_ResolvedNilPipeline(t *testing.T) {
	pr := &v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mypipelinerun",
		},
		Spec: v1beta1.PipelineRunSpec{
			PipelineRef: &v1beta1.PipelineRef{
				ResolverRef: v1beta1.ResolverRef{
					Resolver: "git",
				},
			},
		},
	}
	getPipeline := func(ctx context.Context, n string) (*v1beta1.Pipeline, *v1beta1.RefSource, error) {
		return nil, nil, nil
	}
	ctx := context.Background()
	_, _, err := GetPipelineData(ctx, pr, getPipeline)
	if err == nil {
		t.Fatalf("Expected error when unable to find referenced Pipeline but got none")
	}
}
