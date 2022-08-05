// Copyright Project Contour Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dag

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/projectcontour/contour/internal/fixture"
	"github.com/projectcontour/contour/internal/gatewayapi"
	"github.com/projectcontour/contour/internal/status"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	gatewayapi_v1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayapi_v1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func validGatewayStatusUpdate(listenerName string, kind gatewayapi_v1beta1.Kind, attachedRoutes int) []*status.GatewayStatusUpdate {
	return []*status.GatewayStatusUpdate{
		{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionTrue,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonReady),
					Message: status.MessageValidGateway,
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				listenerName: {
					Name:           gatewayapi_v1beta1.SectionName(listenerName),
					AttachedRoutes: int32(attachedRoutes),
					SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
						{
							Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName),
							Kind:  kind,
						},
					},
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionTrue,
							Reason:  "Ready",
							Message: "Valid listener",
						},
					},
				},
			},
		},
	}
}

func TestGatewayAPIHTTPRouteDAGStatus(t *testing.T) {
	type testcase struct {
		objs                    []interface{}
		gateway                 *gatewayapi_v1beta1.Gateway
		wantRouteConditions     []*status.RouteStatusUpdate
		wantGatewayStatusUpdate []*status.GatewayStatusUpdate
	}

	run := func(t *testing.T, desc string, tc testcase) {
		t.Helper()
		t.Run(desc, func(t *testing.T) {
			t.Helper()
			builder := Builder{
				Source: KubernetesCache{
					RootNamespaces: []string{"roots", "marketing"},
					FieldLogger:    fixture.NewTestLogger(t),
					gatewayclass: &gatewayapi_v1beta1.GatewayClass{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-gc",
						},
						Spec: gatewayapi_v1beta1.GatewayClassSpec{
							ControllerName: "projectcontour.io/contour",
						},
						Status: gatewayapi_v1beta1.GatewayClassStatus{
							Conditions: []metav1.Condition{
								{
									Type:   string(gatewayapi_v1beta1.GatewayClassConditionStatusAccepted),
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
					gateway: tc.gateway,
				},
				Processors: []Processor{
					&GatewayAPIProcessor{
						FieldLogger: fixture.NewTestLogger(t),
					},
					&ListenerProcessor{},
				},
			}

			// Set a default gateway if not defined by a test
			if tc.gateway == nil {
				builder.Source.gateway = &gatewayapi_v1beta1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "contour",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.GatewaySpec{
						Listeners: []gatewayapi_v1beta1.Listener{{
							Name:     "http",
							Port:     80,
							Protocol: gatewayapi_v1beta1.HTTPProtocolType,
							AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
								Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
									From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
								},
							},
						}},
					},
				}
			}

			for _, o := range tc.objs {
				builder.Source.Insert(o)
			}
			dag := builder.Build()
			gotRouteUpdates := dag.StatusCache.GetRouteUpdates()
			gotGatewayUpdates := dag.StatusCache.GetGatewayUpdates()

			ops := []cmp.Option{
				cmpopts.IgnoreFields(metav1.Condition{}, "LastTransitionTime"),
				cmpopts.IgnoreFields(status.RouteStatusUpdate{}, "GatewayRef"),
				cmpopts.IgnoreFields(status.RouteStatusUpdate{}, "Generation"),
				cmpopts.IgnoreFields(status.RouteStatusUpdate{}, "TransitionTime"),
				cmpopts.IgnoreFields(status.RouteStatusUpdate{}, "Resource"),
				cmpopts.IgnoreFields(status.GatewayStatusUpdate{}, "ExistingConditions"),
				cmpopts.IgnoreFields(status.GatewayStatusUpdate{}, "Generation"),
				cmpopts.IgnoreFields(status.GatewayStatusUpdate{}, "TransitionTime"),
				cmpopts.SortSlices(func(i, j metav1.Condition) bool {
					return i.Message < j.Message
				}),
				cmpopts.SortSlices(func(i, j *status.RouteStatusUpdate) bool {
					return i.FullName.String() < j.FullName.String()
				}),
			}

			// Since we're using a single static GatewayClass,
			// set the expected controller string here for all
			// test cases.
			for _, u := range tc.wantRouteConditions {
				u.GatewayController = builder.Source.gatewayclass.Spec.ControllerName

				for _, rps := range u.RouteParentStatuses {
					rps.ControllerName = builder.Source.gatewayclass.Spec.ControllerName
				}
			}

			if diff := cmp.Diff(tc.wantRouteConditions, gotRouteUpdates, ops...); diff != "" {
				t.Fatalf("expected route status: %v, got %v", tc.wantRouteConditions, diff)
			}

			if diff := cmp.Diff(tc.wantGatewayStatusUpdate, gotGatewayUpdates, ops...); diff != "" {
				t.Fatalf("expected gateway status: %v, got %v", tc.wantGatewayStatusUpdate, diff)
			}
		})
	}

	kuardService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}

	kuardService2 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard2",
			Namespace: "default",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}

	kuardService3 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard3",
			Namespace: "default",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}

	run(t, "simple httproute", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
						BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{{
						Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
						Status:  metav1.ConditionTrue,
						Reason:  string(gatewayapi_v1beta1.RouteReasonAccepted),
						Message: "Accepted HTTPRoute",
					}},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 1),
	})

	run(t, "simple httproute with backendref namespace matching route's explicitly specified", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
						BackendRefs: []gatewayapi_v1beta1.HTTPBackendRef{
							{
								BackendRef: gatewayapi_v1beta1.BackendRef{
									BackendObjectReference: gatewayapi_v1beta1.BackendObjectReference{
										Kind:      gatewayapi.KindPtr("Service"),
										Namespace: gatewayapi.NamespacePtr(kuardService.Namespace),
										Name:      gatewayapi_v1beta1.ObjectName(kuardService.Name),
										Port:      gatewayapi.PortNumPtr(8080),
									},
									Weight: pointer.Int32(1),
								},
							},
						},
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{{
				ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
				Conditions: []metav1.Condition{
					{
						Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
						Status:  metav1.ConditionTrue,
						Reason:  string(gatewayapi_v1beta1.RouteReasonAccepted),
						Message: "Accepted HTTPRoute",
					},
				},
			}},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 1),
	})

	run(t, "multiple httproutes", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
						BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
					}},
				},
			},
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic-2",
					Namespace: "default",
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
						BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{
			{
				FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
				RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
					{
						ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
						Conditions: []metav1.Condition{
							{
								Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
								Status:  metav1.ConditionTrue,
								Reason:  string(gatewayapi_v1beta1.RouteReasonAccepted),
								Message: "Accepted HTTPRoute",
							},
						},
					},
				},
			},
			{
				FullName: types.NamespacedName{Namespace: "default", Name: "basic-2"},
				RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
					{
						ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
						Conditions: []metav1.Condition{
							{
								Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
								Status:  metav1.ConditionTrue,
								Reason:  string(gatewayapi_v1beta1.RouteReasonAccepted),
								Message: "Accepted HTTPRoute",
							},
						},
					},
				},
			},
		},
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 2),
	})

	run(t, "prefix path match not starting with '/' for httproute", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches: []gatewayapi_v1beta1.HTTPRouteMatch{{
							Path: &gatewayapi_v1beta1.HTTPPathMatch{
								Type:  gatewayapi.PathMatchTypePtr(gatewayapi_v1beta1.PathMatchPathPrefix),
								Value: pointer.StringPtr("doesnt-start-with-slash"),
							},
						}},
						BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.GatewayClassReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
						{
							Type:    string(status.ConditionValidMatches),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonInvalidPathMatch),
							Message: "Match.Path.Value must start with '/'.",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 0),
	})

	run(t, "exact path match not starting with '/' for httproute", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches: []gatewayapi_v1beta1.HTTPRouteMatch{{
							Path: &gatewayapi_v1beta1.HTTPPathMatch{
								Type:  gatewayapi.PathMatchTypePtr(gatewayapi_v1beta1.PathMatchExact),
								Value: pointer.StringPtr("doesnt-start-with-slash"),
							},
						}},
						BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.GatewayClassReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
						{
							Type:    string(status.ConditionValidMatches),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonInvalidPathMatch),
							Message: "Match.Path.Value must start with '/'.",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 0),
	})

	run(t, "prefix path match with consecutive '/' characters for httproute", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches: []gatewayapi_v1beta1.HTTPRouteMatch{{
							Path: &gatewayapi_v1beta1.HTTPPathMatch{
								Type:  gatewayapi.PathMatchTypePtr(gatewayapi_v1beta1.PathMatchPathPrefix),
								Value: pointer.StringPtr("/foo///bar"),
							},
						}},
						BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.GatewayClassReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
						{
							Type:    string(status.ConditionValidMatches),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonInvalidPathMatch),
							Message: "Match.Path.Value must not contain consecutive '/' characters.",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 0),
	})

	run(t, "exact path match with consecutive '/' characters for httproute", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches: []gatewayapi_v1beta1.HTTPRouteMatch{{
							Path: &gatewayapi_v1beta1.HTTPPathMatch{
								Type:  gatewayapi.PathMatchTypePtr(gatewayapi_v1beta1.PathMatchExact),
								Value: pointer.StringPtr("//foo/bar"),
							},
						}},
						BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.GatewayClassReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
						{
							Type:    string(status.ConditionValidMatches),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonInvalidPathMatch),
							Message: "Match.Path.Value must not contain consecutive '/' characters.",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 0),
	})

	run(t, "invalid path match type for httproute", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches: []gatewayapi_v1beta1.HTTPRouteMatch{{
							Path: &gatewayapi_v1beta1.HTTPPathMatch{
								Type:  gatewayapi.PathMatchTypePtr("UNKNOWN"), // <---- unknown type to break the test
								Value: pointer.StringPtr("/"),
							},
						}},
						BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.GatewayClassReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
						{
							Type:    string(status.ConditionNotImplemented),
							Status:  metav1.ConditionTrue,
							Reason:  string(status.ReasonPathMatchType),
							Message: "HTTPRoute.Spec.Rules.PathMatch: Only Prefix match type and Exact match type are supported.",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 0),
	})

	run(t, "regular expression match not yet supported for httproute", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchRegularExpression, "/"),
						BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.GatewayClassReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
						{
							Type:    string(status.ConditionNotImplemented),
							Status:  metav1.ConditionTrue,
							Reason:  string(status.ReasonPathMatchType),
							Message: "HTTPRoute.Spec.Rules.PathMatch: Only Prefix match type and Exact match type are supported.",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 0),
	})

	run(t, "RegularExpression header match not yet supported for httproute", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches: []gatewayapi_v1beta1.HTTPRouteMatch{{
							Path: &gatewayapi_v1beta1.HTTPPathMatch{
								Type:  gatewayapi.PathMatchTypePtr(gatewayapi_v1beta1.PathMatchPathPrefix),
								Value: pointer.StringPtr("/"),
							},
							Headers: []gatewayapi_v1beta1.HTTPHeaderMatch{
								{
									Type:  gatewayapi.HeaderMatchTypePtr(gatewayapi_v1beta1.HeaderMatchRegularExpression), // <---- RegularExpression type not yet supported
									Name:  gatewayapi_v1beta1.HTTPHeaderName("foo"),
									Value: "bar",
								},
							},
						}},
						BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.GatewayClassReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
						{
							Type:    string(status.ConditionNotImplemented),
							Status:  metav1.ConditionTrue,
							Reason:  string(status.ReasonHeaderMatchType),
							Message: "HTTPRoute.Spec.Rules.Matches.Headers: Only Exact match type is supported",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 0),
	})

	run(t, "RegularExpression query param match not supported for httproute", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches: []gatewayapi_v1beta1.HTTPRouteMatch{{
							Path: &gatewayapi_v1beta1.HTTPPathMatch{
								Type:  gatewayapi.PathMatchTypePtr(gatewayapi_v1beta1.PathMatchPathPrefix),
								Value: pointer.StringPtr("/"),
							},
							QueryParams: []gatewayapi_v1beta1.HTTPQueryParamMatch{
								{
									Type:  gatewayapi.QueryParamMatchTypePtr(gatewayapi_v1beta1.QueryParamMatchRegularExpression), // <---- RegularExpression type not yet supported
									Name:  "param-1",
									Value: "value-1",
								},
							},
						}},
						BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.GatewayClassReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
						{
							Type:    string(status.ConditionNotImplemented),
							Status:  metav1.ConditionTrue,
							Reason:  string(status.ReasonQueryParamMatchType),
							Message: "HTTPRoute.Spec.Rules.Matches.QueryParams: Only Exact match type is supported",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 0),
	})

	run(t, "spec.rules.backendRef.name not specified", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
						BackendRefs: []gatewayapi_v1beta1.HTTPBackendRef{
							{
								BackendRef: gatewayapi_v1beta1.BackendRef{
									BackendObjectReference: gatewayapi_v1beta1.BackendObjectReference{
										Kind: gatewayapi.KindPtr("Service"),
										Port: gatewayapi.PortNumPtr(8080),
									},
								},
							},
						},
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonDegraded),
							Message: "Spec.Rules.BackendRef.Name must be specified",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.GatewayClassReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
					},
				},
			},
		}},
		// This still results in an attached route because it returns a 404.
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 1),
	})

	run(t, "spec.rules.backendRef.serviceName invalid on two matches", testcase{
		objs: []interface{}{
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches: []gatewayapi_v1beta1.HTTPRouteMatch{{
							Path: &gatewayapi_v1beta1.HTTPPathMatch{
								Type:  gatewayapi.PathMatchTypePtr(gatewayapi_v1beta1.PathMatchPathPrefix),
								Value: pointer.StringPtr("/"),
							},
						}},
						BackendRefs: gatewayapi.HTTPBackendRef("invalid-one", 8080, 1),
					}, {
						Matches: []gatewayapi_v1beta1.HTTPRouteMatch{{
							Path: &gatewayapi_v1beta1.HTTPPathMatch{
								Type:  gatewayapi.PathMatchTypePtr(gatewayapi_v1beta1.PathMatchPathPrefix),
								Value: pointer.StringPtr("/blog"),
							},
						}},
						BackendRefs: gatewayapi.HTTPBackendRef("invalid-two", 8080, 1),
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.RouteReasonBackendNotFound),
							Message: "service \"invalid-one\" is invalid: service \"default/invalid-one\" not found, service \"invalid-two\" is invalid: service \"default/invalid-two\" not found",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.GatewayClassReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
					},
				},
			},
		}},
		// This still results in an attached route because it returns a 404.
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 1),
	})

	run(t, "spec.rules.backendRef.port not specified", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
						BackendRefs: []gatewayapi_v1beta1.HTTPBackendRef{
							{
								BackendRef: gatewayapi_v1beta1.BackendRef{
									BackendObjectReference: gatewayapi_v1beta1.BackendObjectReference{
										Kind: gatewayapi.KindPtr("Service"),
										Name: "kuard",
									},
								},
							},
						},
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonDegraded),
							Message: "Spec.Rules.BackendRef.Port must be specified",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.GatewayClassReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
					},
				},
			},
		}},
		// This still results in an attached route because it returns a 404.
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 1),
	})

	run(t, "spec.rules.backendRefs not specified", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonDegraded),
							Message: "At least one Spec.Rules.BackendRef must be specified.",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.GatewayClassReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 0),
	})

	run(t, "spec.rules.backendRef.namespace does not match route", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
						BackendRefs: []gatewayapi_v1beta1.HTTPBackendRef{
							{
								BackendRef: gatewayapi_v1beta1.BackendRef{
									BackendObjectReference: gatewayapi_v1beta1.BackendObjectReference{
										Kind:      gatewayapi.KindPtr("Service"),
										Namespace: gatewayapi.NamespacePtr("some-other-namespace"),
										Name:      "service",
										Port:      gatewayapi.PortNumPtr(8080),
									},
								},
							},
						},
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.ListenerReasonRefNotPermitted),
							Message: "Spec.Rules.BackendRef.Namespace must match the route's namespace or be covered by a ReferencePolicy/ReferenceGrant",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.GatewayClassReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
					},
				},
			},
		}},
		// This still results in an attached route because it returns a 404.
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 1),
	})

	// BEGIN TLS CertificateRef + ReferenceGrant tests
	run(t, "Gateway references TLS cert in different namespace, with valid ReferenceGrant", testcase{
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				GatewayClassName: gatewayapi_v1beta1.ObjectName("projectcontour.io/contour"),
				Listeners: []gatewayapi_v1beta1.Listener{{
					Name:     "https",
					Port:     443,
					Protocol: gatewayapi_v1beta1.HTTPSProtocolType,
					TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
						Mode: gatewayapi.TLSModeTypePtr(gatewayapi_v1beta1.TLSModeTerminate),
						CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
							gatewayapi.CertificateRef("secret", "tls-cert-namespace"),
						},
					},
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
				}},
			},
		},
		objs: []interface{}{
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "tls-cert-namespace",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(fixture.CERTIFICATE, fixture.RSA_PRIVATE_KEY),
			},
			&gatewayapi_v1alpha2.ReferenceGrant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-cert-reference-policy",
					Namespace: "tls-cert-namespace",
				},
				Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
					From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
						Group:     gatewayapi_v1alpha2.GroupName,
						Kind:      "Gateway",
						Namespace: gatewayapi_v1alpha2.Namespace("projectcontour"),
					}},
					To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
						Kind: "Secret",
					}},
				},
			},
		},
		wantGatewayStatusUpdate: validGatewayStatusUpdate("https", "HTTPRoute", 0),
	})

	run(t, "Gateway references TLS cert in different namespace, with no ReferenceGrant", testcase{
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				GatewayClassName: gatewayapi_v1beta1.ObjectName("projectcontour.io/contour"),
				Listeners: []gatewayapi_v1beta1.Listener{{
					Name:     "https",
					Port:     443,
					Protocol: gatewayapi_v1beta1.HTTPSProtocolType,
					TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
						Mode: gatewayapi.TLSModeTypePtr(gatewayapi_v1beta1.TLSModeTerminate),
						CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
							gatewayapi.CertificateRef("secret", "tls-cert-namespace"),
						},
					},
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
				}},
			},
		},
		objs: []interface{}{
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "tls-cert-namespace",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(fixture.CERTIFICATE, fixture.RSA_PRIVATE_KEY),
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonListenersNotValid),
					Message: "Listeners are not valid",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"https": {
					Name: "https",
					SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
						{Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName), Kind: "HTTPRoute"},
					},
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionFalse,
							Reason:  "Invalid",
							Message: "Invalid listener, see other listener conditions for details",
						},
						{
							Type:    string(gatewayapi_v1beta1.ListenerConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.ListenerReasonInvalidCertificateRef),
							Message: "Spec.VirtualHost.TLS.CertificateRefs \"secret\" namespace must match the Gateway's namespace or be covered by a ReferencePolicy/ReferenceGrant",
						},
					},
				},
			},
		}},
	})

	run(t, "Gateway references TLS cert in different namespace, with valid ReferenceGrant (secret-specific)", testcase{
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				GatewayClassName: gatewayapi_v1beta1.ObjectName("projectcontour.io/contour"),
				Listeners: []gatewayapi_v1beta1.Listener{{
					Name:     "https",
					Port:     443,
					Protocol: gatewayapi_v1beta1.HTTPSProtocolType,
					TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
						Mode: gatewayapi.TLSModeTypePtr(gatewayapi_v1beta1.TLSModeTerminate),
						CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
							gatewayapi.CertificateRef("secret", "tls-cert-namespace"),
						},
					},
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
				}},
			},
		},
		objs: []interface{}{
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "tls-cert-namespace",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(fixture.CERTIFICATE, fixture.RSA_PRIVATE_KEY),
			},
			&gatewayapi_v1alpha2.ReferenceGrant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-cert-reference-policy",
					Namespace: "tls-cert-namespace",
				},
				Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
					From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
						Group:     gatewayapi_v1alpha2.GroupName,
						Kind:      "Gateway",
						Namespace: gatewayapi_v1alpha2.Namespace("projectcontour"),
					}},
					To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
						Kind: "Secret",
						Name: gatewayapi.ObjectNamePtr("secret"),
					}},
				},
			},
		},
		wantGatewayStatusUpdate: validGatewayStatusUpdate("https", "HTTPRoute", 0),
	})

	run(t, "Gateway references TLS cert in different namespace, with invalid ReferenceGrant (policy in wrong namespace)", testcase{
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				GatewayClassName: gatewayapi_v1beta1.ObjectName("projectcontour.io/contour"),
				Listeners: []gatewayapi_v1beta1.Listener{{
					Name:     "https",
					Port:     443,
					Protocol: gatewayapi_v1beta1.HTTPSProtocolType,
					TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
						Mode: gatewayapi.TLSModeTypePtr(gatewayapi_v1beta1.TLSModeTerminate),
						CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
							gatewayapi.CertificateRef("secret", "tls-cert-namespace"),
						},
					},
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
				}},
			},
		},
		objs: []interface{}{
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "tls-cert-namespace",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(fixture.CERTIFICATE, fixture.RSA_PRIVATE_KEY),
			},
			&gatewayapi_v1alpha2.ReferenceGrant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-cert-reference-policy",
					Namespace: "wrong-namespace",
				},
				Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
					From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
						Group:     gatewayapi_v1alpha2.GroupName,
						Kind:      "Gateway",
						Namespace: gatewayapi_v1alpha2.Namespace("projectcontour"),
					}},
					To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
						Kind: "Secret",
					}},
				},
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonListenersNotValid),
					Message: "Listeners are not valid",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"https": {
					Name: "https",
					SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
						{Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName), Kind: "HTTPRoute"},
					},
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionFalse,
							Reason:  "Invalid",
							Message: "Invalid listener, see other listener conditions for details",
						},
						{
							Type:    string(gatewayapi_v1beta1.ListenerConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.ListenerReasonInvalidCertificateRef),
							Message: "Spec.VirtualHost.TLS.CertificateRefs \"secret\" namespace must match the Gateway's namespace or be covered by a ReferencePolicy/ReferenceGrant",
						},
					},
				},
			},
		}},
	})

	run(t, "Gateway references TLS cert in different namespace, with invalid ReferenceGrant (wrong From namespace)", testcase{
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				GatewayClassName: gatewayapi_v1beta1.ObjectName("projectcontour.io/contour"),
				Listeners: []gatewayapi_v1beta1.Listener{{
					Name:     "https",
					Port:     443,
					Protocol: gatewayapi_v1beta1.HTTPSProtocolType,
					TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
						Mode: gatewayapi.TLSModeTypePtr(gatewayapi_v1beta1.TLSModeTerminate),
						CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
							gatewayapi.CertificateRef("secret", "tls-cert-namespace"),
						},
					},
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
				}},
			},
		},
		objs: []interface{}{
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "tls-cert-namespace",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(fixture.CERTIFICATE, fixture.RSA_PRIVATE_KEY),
			},
			&gatewayapi_v1alpha2.ReferenceGrant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-cert-reference-policy",
					Namespace: "tls-cert-namespace",
				},
				Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
					From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
						Group:     gatewayapi_v1alpha2.GroupName,
						Kind:      "Gateway",
						Namespace: gatewayapi_v1alpha2.Namespace("wrong-namespace"),
					}},
					To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
						Kind: "Secret",
					}},
				},
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonListenersNotValid),
					Message: "Listeners are not valid",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"https": {
					Name: "https",
					SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
						{Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName), Kind: "HTTPRoute"},
					},
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionFalse,
							Reason:  "Invalid",
							Message: "Invalid listener, see other listener conditions for details",
						},
						{
							Type:    string(gatewayapi_v1beta1.ListenerConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.ListenerReasonInvalidCertificateRef),
							Message: "Spec.VirtualHost.TLS.CertificateRefs \"secret\" namespace must match the Gateway's namespace or be covered by a ReferencePolicy/ReferenceGrant",
						},
					},
				},
			},
		}},
	})

	run(t, "Gateway references TLS cert in different namespace, with invalid ReferenceGrant (wrong From kind)", testcase{
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				GatewayClassName: gatewayapi_v1beta1.ObjectName("projectcontour.io/contour"),
				Listeners: []gatewayapi_v1beta1.Listener{{
					Name:     "https",
					Port:     443,
					Protocol: gatewayapi_v1beta1.HTTPSProtocolType,
					TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
						Mode: gatewayapi.TLSModeTypePtr(gatewayapi_v1beta1.TLSModeTerminate),
						CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
							gatewayapi.CertificateRef("secret", "tls-cert-namespace"),
						},
					},
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
				}},
			},
		},
		objs: []interface{}{
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "tls-cert-namespace",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(fixture.CERTIFICATE, fixture.RSA_PRIVATE_KEY),
			},
			&gatewayapi_v1alpha2.ReferenceGrant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-cert-reference-policy",
					Namespace: "tls-cert-namespace",
				},
				Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
					From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
						Group:     gatewayapi_v1alpha2.GroupName,
						Kind:      "WrongKind",
						Namespace: gatewayapi_v1alpha2.Namespace("projectontour"),
					}},
					To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
						Kind: "Secret",
					}},
				},
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonListenersNotValid),
					Message: "Listeners are not valid",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"https": {
					Name: "https",
					SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
						{Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName), Kind: "HTTPRoute"},
					},
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionFalse,
							Reason:  "Invalid",
							Message: "Invalid listener, see other listener conditions for details",
						},
						{
							Type:    string(gatewayapi_v1beta1.ListenerConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.ListenerReasonInvalidCertificateRef),
							Message: "Spec.VirtualHost.TLS.CertificateRefs \"secret\" namespace must match the Gateway's namespace or be covered by a ReferencePolicy/ReferenceGrant",
						},
					},
				},
			},
		}},
	})

	run(t, "Gateway references TLS cert in different namespace, with invalid ReferenceGrant (wrong To kind)", testcase{
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				GatewayClassName: gatewayapi_v1beta1.ObjectName("projectcontour.io/contour"),
				Listeners: []gatewayapi_v1beta1.Listener{{
					Name:     "https",
					Port:     443,
					Protocol: gatewayapi_v1beta1.HTTPSProtocolType,
					TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
						Mode: gatewayapi.TLSModeTypePtr(gatewayapi_v1beta1.TLSModeTerminate),
						CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
							gatewayapi.CertificateRef("secret", "tls-cert-namespace"),
						},
					},
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
				}},
			},
		},
		objs: []interface{}{
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "tls-cert-namespace",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(fixture.CERTIFICATE, fixture.RSA_PRIVATE_KEY),
			},
			&gatewayapi_v1alpha2.ReferenceGrant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-cert-reference-policy",
					Namespace: "tls-cert-namespace",
				},
				Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
					From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
						Group:     gatewayapi_v1alpha2.GroupName,
						Kind:      "Gateway",
						Namespace: gatewayapi_v1alpha2.Namespace("projectcontour"),
					}},
					To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
						Kind: "WrongKind",
					}},
				},
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonListenersNotValid),
					Message: "Listeners are not valid",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"https": {
					Name: "https",
					SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
						{Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName), Kind: "HTTPRoute"},
					},
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionFalse,
							Reason:  "Invalid",
							Message: "Invalid listener, see other listener conditions for details",
						},
						{
							Type:    string(gatewayapi_v1beta1.ListenerConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.ListenerReasonInvalidCertificateRef),
							Message: "Spec.VirtualHost.TLS.CertificateRefs \"secret\" namespace must match the Gateway's namespace or be covered by a ReferencePolicy/ReferenceGrant",
						},
					},
				},
			},
		}},
	})

	run(t, "Gateway references TLS cert in different namespace, with invalid ReferenceGrant (wrong secret name)", testcase{
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				GatewayClassName: gatewayapi_v1beta1.ObjectName("projectcontour.io/contour"),
				Listeners: []gatewayapi_v1beta1.Listener{{
					Name:     "https",
					Port:     443,
					Protocol: gatewayapi_v1beta1.HTTPSProtocolType,
					TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
						Mode: gatewayapi.TLSModeTypePtr(gatewayapi_v1beta1.TLSModeTerminate),
						CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
							gatewayapi.CertificateRef("secret", "tls-cert-namespace"),
						},
					},
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
				}},
			},
		},
		objs: []interface{}{
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "tls-cert-namespace",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(fixture.CERTIFICATE, fixture.RSA_PRIVATE_KEY),
			},
			&gatewayapi_v1alpha2.ReferenceGrant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-cert-reference-policy",
					Namespace: "tls-cert-namespace",
				},
				Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
					From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
						Group:     gatewayapi_v1alpha2.GroupName,
						Kind:      "Gateway",
						Namespace: gatewayapi_v1alpha2.Namespace("projectcontour"),
					}},
					To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
						Kind: "Secret",
						Name: gatewayapi.ObjectNamePtr("wrong-name"),
					}},
				},
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonListenersNotValid),
					Message: "Listeners are not valid",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"https": {
					Name: "https",
					SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
						{Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName), Kind: "HTTPRoute"},
					},
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionFalse,
							Reason:  "Invalid",
							Message: "Invalid listener, see other listener conditions for details",
						},
						{
							Type:    string(gatewayapi_v1beta1.ListenerConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.ListenerReasonInvalidCertificateRef),
							Message: "Spec.VirtualHost.TLS.CertificateRefs \"secret\" namespace must match the Gateway's namespace or be covered by a ReferencePolicy/ReferenceGrant",
						},
					},
				},
			},
		}},
	})

	// END TLS CertificateRef + ReferenceGrant tests

	run(t, "spec.rules.hostname: invalid wildcard", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"*.*.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonDegraded),
							Message: "invalid hostname \"*.*.projectcontour.io\": [a wildcard DNS-1123 subdomain must start with '*.', followed by a valid DNS subdomain, which must consist of lower case alphanumeric characters, '-' or '.' and end with an alphanumeric character (e.g. '*.example.com', regex used for validation is '\\*\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')]",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.RouteReasonNoMatchingListenerHostname),
							Message: "No intersecting hostnames were found between the listener and the route.",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 0),
	})

	run(t, "spec.rules.hostname: invalid hostname", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"#projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonDegraded),
							Message: "invalid hostname \"#projectcontour.io\": [a lowercase RFC 1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character (e.g. 'example.com', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')]",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.RouteReasonNoMatchingListenerHostname),
							Message: "No intersecting hostnames were found between the listener and the route.",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 0),
	})

	run(t, "spec.rules.hostname: invalid hostname, ip address", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"1.2.3.4",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonDegraded),
							Message: "invalid hostname \"1.2.3.4\": must be a DNS name, not an IP address",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.RouteReasonNoMatchingListenerHostname),
							Message: "No intersecting hostnames were found between the listener and the route.",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 0),
	})

	run(t, "two HTTP listeners, route's hostname intersects with one of them", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{"foo.projectcontour.io"},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
						BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
					}},
				},
			}},
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				Listeners: []gatewayapi_v1beta1.Listener{
					{
						Name:     "listener-1",
						Port:     80,
						Protocol: gatewayapi_v1beta1.HTTPProtocolType,
						AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
							Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
								From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
							},
						},
						Hostname: gatewayapi.ListenerHostname("*.projectcontour.io"),
					},
					{
						Name:     "listener-2",
						Port:     80,
						Protocol: gatewayapi_v1beta1.HTTPProtocolType,
						AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
							Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
								From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
							},
						},
						Hostname: gatewayapi.ListenerHostname("specific.hostname.io"),
					},
				},
			},
		},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.RouteReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{
			{
				FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
				Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
					gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
					gatewayapi_v1beta1.GatewayConditionReady: {
						Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
						Status:  metav1.ConditionTrue,
						Reason:  string(gatewayapi_v1beta1.GatewayReasonReady),
						Message: status.MessageValidGateway,
					},
				},
				ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
					"listener-1": {
						Name:           gatewayapi_v1beta1.SectionName("listener-1"),
						AttachedRoutes: int32(1),
						SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
							{
								Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName),
								Kind:  "HTTPRoute",
							},
						},
						Conditions: []metav1.Condition{
							{
								Type:    "Ready",
								Status:  metav1.ConditionTrue,
								Reason:  "Ready",
								Message: "Valid listener",
							},
						},
					},
					"listener-2": {
						Name:           gatewayapi_v1beta1.SectionName("listener-2"),
						AttachedRoutes: int32(0),
						SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
							{
								Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName),
								Kind:  "HTTPRoute",
							},
						},
						Conditions: []metav1.Condition{
							{
								Type:    "Ready",
								Status:  metav1.ConditionTrue,
								Reason:  "Ready",
								Message: "Valid listener",
							},
						},
					},
				},
			},
		},
	})

	run(t, "two HTTP listeners, route's hostname intersects with neither of them", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{"foo.randomdomain.io"},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
						BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
					}},
				},
			}},
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				Listeners: []gatewayapi_v1beta1.Listener{
					{
						Name:     "listener-1",
						Port:     80,
						Protocol: gatewayapi_v1beta1.HTTPProtocolType,
						AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
							Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
								From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
							},
						},
						Hostname: gatewayapi.ListenerHostname("*.projectcontour.io"),
					},
					{
						Name:     "listener-2",
						Port:     80,
						Protocol: gatewayapi_v1beta1.HTTPProtocolType,
						AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
							Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
								From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
							},
						},
						Hostname: gatewayapi.ListenerHostname("specific.hostname.io"),
					},
				},
			},
		},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.RouteReasonNoMatchingListenerHostname),
							Message: "No intersecting hostnames were found between the listener and the route.",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{
			{
				FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
				Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
					gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
					gatewayapi_v1beta1.GatewayConditionReady: {
						Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
						Status:  metav1.ConditionTrue,
						Reason:  string(gatewayapi_v1beta1.GatewayReasonReady),
						Message: status.MessageValidGateway,
					},
				},
				ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
					"listener-1": {
						Name:           gatewayapi_v1beta1.SectionName("listener-1"),
						AttachedRoutes: int32(0),
						SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
							{
								Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName),
								Kind:  "HTTPRoute",
							},
						},
						Conditions: []metav1.Condition{
							{
								Type:    "Ready",
								Status:  metav1.ConditionTrue,
								Reason:  "Ready",
								Message: "Valid listener",
							},
						},
					},
					"listener-2": {
						Name:           gatewayapi_v1beta1.SectionName("listener-2"),
						AttachedRoutes: int32(0),
						SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
							{
								Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName),
								Kind:  "HTTPRoute",
							},
						},
						Conditions: []metav1.Condition{
							{
								Type:    "Ready",
								Status:  metav1.ConditionTrue,
								Reason:  "Ready",
								Message: "Valid listener",
							},
						},
					},
				},
			},
		},
	})

	run(t, "More than one RequestMirror filters in HTTPRoute.Spec.Rules.Filters", testcase{
		objs: []interface{}{
			kuardService,
			kuardService2,
			kuardService3,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
						BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						Filters: []gatewayapi_v1beta1.HTTPRouteFilter{{
							Type: gatewayapi_v1beta1.HTTPRouteFilterRequestMirror,
							RequestMirror: &gatewayapi_v1beta1.HTTPRequestMirrorFilter{
								BackendRef: gatewayapi.ServiceBackendObjectRef("kuard2", 8080),
							},
						}, {
							Type: gatewayapi_v1beta1.HTTPRouteFilterRequestMirror,
							RequestMirror: &gatewayapi_v1beta1.HTTPRequestMirrorFilter{
								BackendRef: gatewayapi.ServiceBackendObjectRef("kuard3", 8080),
							}},
						},
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(status.ConditionNotImplemented),
							Status:  metav1.ConditionTrue,
							Reason:  string(status.ReasonNotImplemented),
							Message: "HTTPRoute.Spec.Rules.Filters: Only one mirror filter is supported.",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.RouteReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
					},
				},
			},
		}},
		// Invalid filters still result in an attached route.
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 1),
	})

	run(t, "Invalid RequestMirror filter due to unspecified backendRef.name", testcase{
		objs: []interface{}{
			kuardService,
			kuardService2,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
						BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						Filters: []gatewayapi_v1beta1.HTTPRouteFilter{{
							Type: gatewayapi_v1beta1.HTTPRouteFilterRequestMirror,
							RequestMirror: &gatewayapi_v1beta1.HTTPRequestMirrorFilter{
								BackendRef: gatewayapi_v1beta1.BackendObjectReference{
									Group: gatewayapi.GroupPtr(""),
									Kind:  gatewayapi.KindPtr("Service"),
									Port:  gatewayapi.PortNumPtr(8080),
								},
							},
						}},
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonDegraded),
							Message: "Spec.Rules.Filters.RequestMirror.BackendRef.Name must be specified",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.RouteReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
					},
				},
			},
		}},
		// This still results in an attached route because it returns a 404.
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 1),
	})

	run(t, "Invalid RequestMirror filter due to unspecified backendRef.port", testcase{
		objs: []interface{}{
			kuardService,
			kuardService2,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
						BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						Filters: []gatewayapi_v1beta1.HTTPRouteFilter{{
							Type: gatewayapi_v1beta1.HTTPRouteFilterRequestMirror,
							RequestMirror: &gatewayapi_v1beta1.HTTPRequestMirrorFilter{
								BackendRef: gatewayapi_v1beta1.BackendObjectReference{
									Group: gatewayapi.GroupPtr(""),
									Kind:  gatewayapi.KindPtr("Service"),
									Name:  gatewayapi_v1beta1.ObjectName("kuard2"),
								},
							},
						}},
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonDegraded),
							Message: "Spec.Rules.Filters.RequestMirror.BackendRef.Port must be specified",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.RouteReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
					},
				},
			},
		}},
		// This still results in an attached route because it returns a 404.
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 1),
	})

	run(t, "Invalid RequestMirror filter due to invalid backendRef.name on two matches", testcase{
		objs: []interface{}{
			kuardService,
			kuardService2,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches: []gatewayapi_v1beta1.HTTPRouteMatch{{
							Path: &gatewayapi_v1beta1.HTTPPathMatch{
								Type:  gatewayapi.PathMatchTypePtr(gatewayapi_v1beta1.PathMatchPathPrefix),
								Value: pointer.StringPtr("/"),
							},
						}},
						BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						Filters: []gatewayapi_v1beta1.HTTPRouteFilter{{
							Type: gatewayapi_v1beta1.HTTPRouteFilterRequestMirror,
							RequestMirror: &gatewayapi_v1beta1.HTTPRequestMirrorFilter{
								BackendRef: gatewayapi.ServiceBackendObjectRef("invalid-one", 8080),
							},
						}},
					}, {
						BackendRefs: gatewayapi.HTTPBackendRef("kuard2", 8080, 1),
						Matches: []gatewayapi_v1beta1.HTTPRouteMatch{{
							Path: &gatewayapi_v1beta1.HTTPPathMatch{
								Type:  gatewayapi.PathMatchTypePtr(gatewayapi_v1beta1.PathMatchPathPrefix),
								Value: pointer.StringPtr("/blog"),
							},
						}},
						Filters: []gatewayapi_v1beta1.HTTPRouteFilter{{
							Type: gatewayapi_v1beta1.HTTPRouteFilterRequestMirror,
							RequestMirror: &gatewayapi_v1beta1.HTTPRequestMirrorFilter{
								BackendRef: gatewayapi.ServiceBackendObjectRef("invalid-two", 8080),
							},
						}},
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.RouteReasonBackendNotFound),
							Message: "service \"invalid-one\" is invalid: service \"default/invalid-one\" not found, service \"invalid-two\" is invalid: service \"default/invalid-two\" not found",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.RouteReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
					},
				},
			},
		}},
		// This still results in an attached route because it returns a 404.
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 1),
	})

	run(t, "Invalid RequestMirror filter due to unmatched backendRef.namespace", testcase{
		objs: []interface{}{
			kuardService,
			kuardService2,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
						BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						Filters: []gatewayapi_v1beta1.HTTPRouteFilter{{
							Type: gatewayapi_v1beta1.HTTPRouteFilterRequestMirror,
							RequestMirror: &gatewayapi_v1beta1.HTTPRequestMirrorFilter{
								BackendRef: gatewayapi_v1beta1.BackendObjectReference{
									Group:     gatewayapi.GroupPtr(""),
									Kind:      gatewayapi.KindPtr("Service"),
									Namespace: gatewayapi.NamespacePtr("some-other-namespace"),
									Name:      gatewayapi_v1beta1.ObjectName("kuard2"),
									Port:      gatewayapi.PortNumPtr(8080),
								},
							},
						}},
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.ListenerReasonRefNotPermitted),
							Message: "Spec.Rules.Filters.RequestMirror.BackendRef.Namespace must match the route's namespace or be covered by a ReferencePolicy/ReferenceGrant",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.RouteReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
					},
				},
			},
		}},
		// This still results in an attached route because it returns a 404.
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 1),
	})

	run(t, "HTTPRouteFilterRequestMirror not yet supported for httproute backendref", testcase{
		objs: []interface{}{

			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
						BackendRefs: []gatewayapi_v1beta1.HTTPBackendRef{
							{
								BackendRef: gatewayapi_v1beta1.BackendRef{
									BackendObjectReference: gatewayapi.ServiceBackendObjectRef("kuard", 8080),
								},
								Filters: []gatewayapi_v1beta1.HTTPRouteFilter{{
									Type: gatewayapi_v1beta1.HTTPRouteFilterRequestMirror, // HTTPRouteFilterRequestMirror is not supported yet.
								}},
							},
						},
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(status.ConditionNotImplemented),
							Status:  metav1.ConditionTrue,
							Reason:  string(status.ReasonHTTPRouteFilterType),
							Message: "HTTPRoute.Spec.Rules.BackendRef.Filters: Only RequestHeaderModifier type is supported.",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.RouteReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
					},
				},
			},
		}},
		// Invalid filters still result in an attached route.
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 1),
	})

	run(t, "Invalid RequestHeaderModifier due to duplicated headers", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
						BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						Filters: []gatewayapi_v1beta1.HTTPRouteFilter{{
							Type: gatewayapi_v1beta1.HTTPRouteFilterRequestHeaderModifier,
							RequestHeaderModifier: &gatewayapi_v1beta1.HTTPRequestHeaderFilter{
								Set: []gatewayapi_v1beta1.HTTPHeader{
									{Name: "custom", Value: "duplicated"},
									{Name: "Custom", Value: "duplicated"},
								},
							},
						}},
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonDegraded),
							Message: "duplicate header addition: \"Custom\" on request headers",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.RouteReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
					},
				},
			},
		}},
		// Invalid filters still result in an attached route.
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 1),
	})

	run(t, "Invalid RequestHeaderModifier after forward due to invalid headers", testcase{
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1beta1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
					},
					Hostnames: []gatewayapi_v1beta1.Hostname{
						"test.projectcontour.io",
					},
					Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
						Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
						BackendRefs: []gatewayapi_v1beta1.HTTPBackendRef{
							{
								BackendRef: gatewayapi_v1beta1.BackendRef{
									BackendObjectReference: gatewayapi.ServiceBackendObjectRef("kuard", 8080),
								},
								Filters: []gatewayapi_v1beta1.HTTPRouteFilter{{
									Type: gatewayapi_v1beta1.HTTPRouteFilterRequestHeaderModifier,
									RequestHeaderModifier: &gatewayapi_v1beta1.HTTPRequestHeaderFilter{
										Set: []gatewayapi_v1beta1.HTTPHeader{
											{Name: "!invalid-header", Value: "foo"},
										},
									},
								}},
							},
						},
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonDegraded),
							Message: "invalid set header \"!invalid-Header\": [a valid HTTP header must consist of alphanumeric characters or '-' (e.g. 'X-Header-Name', regex used for validation is '[-A-Za-z0-9]+')] on request headers",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.RouteReasonAccepted),
							Message: "Accepted HTTPRoute",
						},
					},
				},
			},
		}},
		// Invalid filters still result in an attached route.
		wantGatewayStatusUpdate: validGatewayStatusUpdate("http", "HTTPRoute", 1),
	})

	run(t, "gateway.spec.addresses results in invalid gateway", testcase{
		objs: []interface{}{},
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				Addresses: []gatewayapi_v1beta1.GatewayAddress{{
					Value: "1.2.3.4",
				}},
				Listeners: []gatewayapi_v1beta1.Listener{{
					Name:     "http",
					Port:     80,
					Protocol: gatewayapi_v1beta1.HTTPProtocolType,
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
				}},
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonAddressNotAssigned),
					Message: "None of the addresses in Spec.Addresses have been assigned to the Gateway",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"http": {
					Name: "http",
					SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
						{
							Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName),
							Kind:  "HTTPRoute",
						},
					},
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionTrue,
							Reason:  "Ready",
							Message: "Valid listener",
						},
					},
				},
			},
		}},
	})

	run(t, "invalid allowedroutes API group results in a listener condition", testcase{
		objs: []interface{}{},
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				Listeners: []gatewayapi_v1beta1.Listener{{
					Name:     "http",
					Port:     80,
					Protocol: gatewayapi_v1beta1.HTTPProtocolType,
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Kinds: []gatewayapi_v1beta1.RouteGroupKind{
							{Group: gatewayapi.GroupPtr("invalid-group"), Kind: "HTTPRoute"},
						},
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
				}},
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonListenersNotValid),
					Message: "Listeners are not valid",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"http": {
					Name:           "http",
					SupportedKinds: nil,
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionFalse,
							Reason:  "Invalid",
							Message: "Invalid listener, see other listener conditions for details",
						},
						{
							Type:    string(gatewayapi_v1beta1.ListenerConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.ListenerReasonInvalidRouteKinds),
							Message: "Group \"invalid-group\" is not supported, group must be \"gateway.networking.k8s.io\"",
						},
					},
				},
			},
		}},
	})

	run(t, "invalid allowedroutes API kind results in a listener condition", testcase{
		objs: []interface{}{},
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				Listeners: []gatewayapi_v1beta1.Listener{{
					Name:     "http",
					Port:     80,
					Protocol: gatewayapi_v1beta1.HTTPProtocolType,
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Kinds: []gatewayapi_v1beta1.RouteGroupKind{
							{Kind: "FooRoute"},
						},
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
				}},
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonListenersNotValid),
					Message: "Listeners are not valid",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"http": {
					Name:           "http",
					SupportedKinds: nil,
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionFalse,
							Reason:  "Invalid",
							Message: "Invalid listener, see other listener conditions for details",
						},
						{
							Type:    string(gatewayapi_v1beta1.ListenerConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.ListenerReasonInvalidRouteKinds),
							Message: "Kind \"FooRoute\" is not supported, kind must be \"HTTPRoute\" or \"TLSRoute\"",
						},
					},
				},
			},
		}},
	})

	run(t, "allowedroute of TLSRoute on a non-TLS listener results in a listener condition", testcase{
		objs: []interface{}{},
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				Listeners: []gatewayapi_v1beta1.Listener{{
					Name:     "http",
					Port:     80,
					Protocol: gatewayapi_v1beta1.HTTPProtocolType,
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Kinds: []gatewayapi_v1beta1.RouteGroupKind{
							{Kind: "TLSRoute"},
						},
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
				}},
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonListenersNotValid),
					Message: "Listeners are not valid",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"http": {
					Name:           "http",
					SupportedKinds: nil,
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionFalse,
							Reason:  "Invalid",
							Message: "Invalid listener, see other listener conditions for details",
						},
						{
							Type:    string(gatewayapi_v1beta1.ListenerConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.ListenerReasonInvalidRouteKinds),
							Message: "TLSRoutes are incompatible with listener protocol \"HTTP\"",
						},
					},
				},
			},
		}},
	})

	run(t, "TLS certificate ref to a non-secret on an HTTPS listener results in a listener condition", testcase{
		objs: []interface{}{},
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				Listeners: []gatewayapi_v1beta1.Listener{{
					Name:     "https",
					Port:     443,
					Protocol: gatewayapi_v1beta1.HTTPSProtocolType,
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
					TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
						CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
							{
								Group: gatewayapi.GroupPtr("invalid-group"),
								Kind:  gatewayapi.KindPtr("NotASecret"),
								Name:  "foo",
							},
						},
					},
				}},
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonListenersNotValid),
					Message: "Listeners are not valid",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"https": {
					Name: "https",
					SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
						{Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName), Kind: "HTTPRoute"},
					},
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionFalse,
							Reason:  "Invalid",
							Message: "Invalid listener, see other listener conditions for details",
						},
						{
							Type:    string(gatewayapi_v1beta1.ListenerConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.ListenerReasonInvalidCertificateRef),
							Message: "Spec.VirtualHost.TLS.CertificateRefs \"foo\" must contain a reference to a core.Secret",
						},
					},
				},
			},
		}},
	})

	run(t, "nonexistent TLS certificate ref on an HTTPS listener results in a listener condition", testcase{
		objs: []interface{}{},
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				Listeners: []gatewayapi_v1beta1.Listener{{
					Name:     "https",
					Port:     443,
					Protocol: gatewayapi_v1beta1.HTTPSProtocolType,
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
					TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
						CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
							gatewayapi.CertificateRef("nonexistent-secret", "projectcontour"),
						},
					},
				}},
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonListenersNotValid),
					Message: "Listeners are not valid",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"https": {
					Name: "https",
					SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
						{Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName), Kind: "HTTPRoute"},
					},
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionFalse,
							Reason:  "Invalid",
							Message: "Invalid listener, see other listener conditions for details",
						},
						{
							Type:    string(gatewayapi_v1beta1.ListenerConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.ListenerReasonInvalidCertificateRef),
							Message: "Spec.VirtualHost.TLS.CertificateRefs \"nonexistent-secret\" referent is invalid: Secret not found",
						},
					},
				},
			},
		}},
	})

	run(t, "invalid listener protocol results in a listener condition", testcase{
		objs: []interface{}{},
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				Listeners: []gatewayapi_v1beta1.Listener{{
					Name:     "http",
					Port:     80,
					Protocol: "invalid",
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
				}},
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonListenersNotValid),
					Message: "Listeners are not valid",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"http": {
					Name:           "http",
					SupportedKinds: nil,
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionFalse,
							Reason:  "Invalid",
							Message: "Invalid listener, see other listener conditions for details",
						},
						{
							Type:    string(gatewayapi_v1beta1.ListenerConditionDetached),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.ListenerReasonUnsupportedProtocol),
							Message: "Listener protocol \"invalid\" is unsupported, must be one of HTTP, HTTPS or TLS",
						},
					},
				},
			},
		}},
	})

	run(t, "HTTPS listener without TLS defined results in a listener condition", testcase{
		objs: []interface{}{},
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				Listeners: []gatewayapi_v1beta1.Listener{{
					Name:     "https",
					Port:     443,
					Protocol: gatewayapi_v1beta1.HTTPSProtocolType,
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
				}},
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonListenersNotValid),
					Message: "Listeners are not valid",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"https": {
					Name: "https",
					SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
						{Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName), Kind: "HTTPRoute"},
					},
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionFalse,
							Reason:  "Invalid",
							Message: "Listener.TLS is required when protocol is \"HTTPS\".",
						},
					},
				},
			},
		}},
	})

	run(t, "TLS listener without TLS defined results in a listener condition", testcase{
		objs: []interface{}{},
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				Listeners: []gatewayapi_v1beta1.Listener{{
					Name:     "tls",
					Port:     443,
					Protocol: gatewayapi_v1beta1.TLSProtocolType,
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
				}},
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonListenersNotValid),
					Message: "Listeners are not valid",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"tls": {
					Name: "tls",
					SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
						{Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName), Kind: "TLSRoute"},
					},
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionFalse,
							Reason:  "Invalid",
							Message: "Listener.TLS is required when protocol is \"TLS\".",
						},
					},
				},
			},
		}},
	})

	run(t, "TLS Passthrough listener with a TLS certificate ref defined results in a listener condition", testcase{
		objs: []interface{}{},
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				Listeners: []gatewayapi_v1beta1.Listener{{
					Name:     "tls",
					Port:     443,
					Protocol: gatewayapi_v1beta1.TLSProtocolType,
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
					TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
						Mode: gatewayapi.TLSModeTypePtr(gatewayapi_v1beta1.TLSModePassthrough),
						CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
							gatewayapi.CertificateRef("tlscert", "projectcontour"),
						},
					},
				}},
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonListenersNotValid),
					Message: "Listeners are not valid",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"tls": {
					Name: "tls",
					SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
						{Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName), Kind: "TLSRoute"},
					},
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionFalse,
							Reason:  "Invalid",
							Message: "Listener.TLS.CertificateRefs cannot be defined when Listener.TLS.Mode is \"Passthrough\".",
						},
					},
				},
			},
		}},
	})

	run(t, "TLS listener with TLS.Mode=Terminate results in a listener condition", testcase{
		objs: []interface{}{},
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				Listeners: []gatewayapi_v1beta1.Listener{{
					Name:     "tls",
					Port:     443,
					Protocol: gatewayapi_v1beta1.TLSProtocolType,
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
					TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
						Mode: gatewayapi.TLSModeTypePtr(gatewayapi_v1beta1.TLSModeTerminate),
						CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
							gatewayapi.CertificateRef("tlscert", "projectcontour"),
						},
					},
				}},
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonListenersNotValid),
					Message: "Listeners are not valid",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"tls": {
					Name: "tls",
					SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
						{Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName), Kind: "TLSRoute"},
					},
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionFalse,
							Reason:  "Invalid",
							Message: "Listener.TLS.Mode must be \"Passthrough\" when protocol is \"TLS\".",
						},
					},
				},
			},
		}},
	})

	run(t, "HTTPS listener with TLS.Mode=Passthrough results in a listener condition", testcase{
		objs: []interface{}{},
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				Listeners: []gatewayapi_v1beta1.Listener{{
					Name:     "https",
					Port:     443,
					Protocol: gatewayapi_v1beta1.HTTPSProtocolType,
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
					TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
						Mode: gatewayapi.TLSModeTypePtr(gatewayapi_v1beta1.TLSModePassthrough),
					},
				}},
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonListenersNotValid),
					Message: "Listeners are not valid",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"https": {
					Name: "https",
					SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
						{Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName), Kind: "HTTPRoute"},
					},
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionFalse,
							Reason:  "Invalid",
							Message: "Listener.TLS.Mode must be \"Terminate\" when protocol is \"HTTPS\".",
						},
					},
				},
			},
		}},
	})

	run(t, "Listener with FromNamespaces=Selector, no selector specified", testcase{
		objs: []interface{}{},
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				Listeners: []gatewayapi_v1beta1.Listener{
					{
						Name:     "http",
						Port:     80,
						Protocol: gatewayapi_v1beta1.HTTPProtocolType,
						AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
							Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
								From:     gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromSelector),
								Selector: nil,
							},
						},
					}},
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonListenersNotValid),
					Message: "Listeners are not valid",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"http": {
					Name: "http",
					SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
						{Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName), Kind: "HTTPRoute"},
					},
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionFalse,
							Reason:  "Invalid",
							Message: "Listener.AllowedRoutes.Namespaces.Selector is required when Listener.AllowedRoutes.Namespaces.From is set to \"Selector\".",
						},
					},
				},
			},
		}},
	})

	run(t, "Listener with FromNamespaces=Selector, invalid selector (can't specify values with Exists operator)", testcase{
		objs: []interface{}{},
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				Listeners: []gatewayapi_v1beta1.Listener{
					{
						Name:     "http",
						Port:     80,
						Protocol: gatewayapi_v1beta1.HTTPProtocolType,
						AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
							Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
								From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromSelector),
								Selector: &metav1.LabelSelector{
									MatchExpressions: []metav1.LabelSelectorRequirement{{
										Key:      "something",
										Operator: metav1.LabelSelectorOpExists,
										Values:   []string{"error"},
									}},
								},
							},
						},
					}},
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonListenersNotValid),
					Message: "Listeners are not valid",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"http": {
					Name: "http",
					SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
						{Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName), Kind: "HTTPRoute"},
					},
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionFalse,
							Reason:  "Invalid",
							Message: "Error parsing Listener.AllowedRoutes.Namespaces.Selector: values: Invalid value: []string{\"error\"}: values set must be empty for exists and does not exist.",
						},
					},
				},
			},
		}},
	})

	run(t, "Listener with FromNamespaces=Selector, invalid selector (must specify MatchLabels and/or MatchExpressions)", testcase{
		objs: []interface{}{},
		gateway: &gatewayapi_v1beta1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "contour",
				Namespace: "projectcontour",
			},
			Spec: gatewayapi_v1beta1.GatewaySpec{
				Listeners: []gatewayapi_v1beta1.Listener{
					{
						Name:     "http",
						Port:     80,
						Protocol: gatewayapi_v1beta1.HTTPProtocolType,
						AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
							Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
								From:     gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromSelector),
								Selector: &metav1.LabelSelector{},
							},
						},
					}},
			},
		},
		wantGatewayStatusUpdate: []*status.GatewayStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "projectcontour", Name: "contour"},
			Conditions: map[gatewayapi_v1beta1.GatewayConditionType]metav1.Condition{
				gatewayapi_v1beta1.GatewayConditionScheduled: gatewayScheduledCondition(),
				gatewayapi_v1beta1.GatewayConditionReady: {
					Type:    string(gatewayapi_v1beta1.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(gatewayapi_v1beta1.GatewayReasonListenersNotValid),
					Message: "Listeners are not valid",
				},
			},
			ListenerStatus: map[string]*gatewayapi_v1beta1.ListenerStatus{
				"http": {
					Name: "http",
					SupportedKinds: []gatewayapi_v1beta1.RouteGroupKind{
						{Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName), Kind: "HTTPRoute"},
					},
					Conditions: []metav1.Condition{
						{
							Type:    "Ready",
							Status:  metav1.ConditionFalse,
							Reason:  "Invalid",
							Message: "Listener.AllowedRoutes.Namespaces.Selector must specify at least one MatchLabel or MatchExpression.",
						},
					},
				},
			},
		}},
	})

}

func TestGatewayAPITLSRouteDAGStatus(t *testing.T) {

	type testcase struct {
		objs                    []interface{}
		gateway                 *gatewayapi_v1beta1.Gateway
		wantRouteConditions     []*status.RouteStatusUpdate
		wantGatewayStatusUpdate []*status.GatewayStatusUpdate
	}

	run := func(t *testing.T, desc string, tc testcase) {
		t.Helper()
		t.Run(desc, func(t *testing.T) {
			t.Helper()
			builder := Builder{
				Source: KubernetesCache{
					RootNamespaces: []string{"roots", "marketing"},
					FieldLogger:    fixture.NewTestLogger(t),
					gateway:        tc.gateway,
					gatewayclass: &gatewayapi_v1beta1.GatewayClass{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-gc",
						},
						Spec: gatewayapi_v1beta1.GatewayClassSpec{
							ControllerName: "projectcontour.io/contour",
						},
						Status: gatewayapi_v1beta1.GatewayClassStatus{
							Conditions: []metav1.Condition{
								{
									Type:   string(gatewayapi_v1beta1.GatewayClassConditionStatusAccepted),
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
				Processors: []Processor{
					&GatewayAPIProcessor{
						FieldLogger: fixture.NewTestLogger(t),
					},
					&ListenerProcessor{},
				},
			}

			// Add a default cert to be used in tests with TLS.
			builder.Source.Insert(fixture.SecretProjectContourCert)

			for _, o := range tc.objs {
				builder.Source.Insert(o)
			}
			dag := builder.Build()
			gotRouteUpdates := dag.StatusCache.GetRouteUpdates()
			gotGatewayUpdates := dag.StatusCache.GetGatewayUpdates()

			ops := []cmp.Option{
				cmpopts.IgnoreFields(metav1.Condition{}, "LastTransitionTime"),
				cmpopts.IgnoreFields(status.RouteStatusUpdate{}, "GatewayRef"),
				cmpopts.IgnoreFields(status.RouteStatusUpdate{}, "Generation"),
				cmpopts.IgnoreFields(status.RouteStatusUpdate{}, "TransitionTime"),
				cmpopts.IgnoreFields(status.RouteStatusUpdate{}, "Resource"),
				cmpopts.IgnoreFields(status.GatewayStatusUpdate{}, "ExistingConditions"),
				cmpopts.IgnoreFields(status.GatewayStatusUpdate{}, "Generation"),
				cmpopts.IgnoreFields(status.GatewayStatusUpdate{}, "TransitionTime"),
				cmpopts.SortSlices(func(i, j metav1.Condition) bool {
					return i.Message < j.Message
				}),
			}

			// Since we're using a single static GatewayClass,
			// set the expected controller string here for all
			// test cases.
			for _, u := range tc.wantRouteConditions {
				u.GatewayController = builder.Source.gatewayclass.Spec.ControllerName

				for _, rps := range u.RouteParentStatuses {
					rps.ControllerName = builder.Source.gatewayclass.Spec.ControllerName
				}
			}

			if diff := cmp.Diff(tc.wantRouteConditions, gotRouteUpdates, ops...); diff != "" {
				t.Fatalf("expected route status: %v, got %v", tc.wantRouteConditions, diff)
			}

			if diff := cmp.Diff(tc.wantGatewayStatusUpdate, gotGatewayUpdates, ops...); diff != "" {
				t.Fatalf("expected gateway status: %v, got %v", tc.wantGatewayStatusUpdate, diff)
			}

		})
	}

	gw := &gatewayapi_v1beta1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "contour",
			Namespace: "projectcontour",
		},
		Spec: gatewayapi_v1beta1.GatewaySpec{
			Listeners: []gatewayapi_v1beta1.Listener{{
				Name:     "tls-passthrough",
				Port:     443,
				Protocol: gatewayapi_v1beta1.TLSProtocolType,
				TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
					Mode: gatewayapi.TLSModeTypePtr(gatewayapi_v1beta1.TLSModePassthrough),
				},
				AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
					Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
						From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
					},
				},
			}},
		},
	}

	kuardService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}

	run(t, "TLSRoute: spec.rules.backendRef.name not specified", testcase{
		gateway: gw,
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1alpha2.TLSRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1alpha2.TLSRouteSpec{
					CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1alpha2.ParentReference{
							gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour"),
						},
					},
					Hostnames: []gatewayapi_v1alpha2.Hostname{"test.projectcontour.io"},
					Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
						BackendRefs: []gatewayapi_v1alpha2.BackendRef{
							{
								BackendObjectReference: gatewayapi_v1alpha2.BackendObjectReference{
									Kind: gatewayapi.KindPtrV1Alpha2("Service"),
									Port: gatewayapi.PortNumPtrV1Alpha2(8080),
								},
							},
						},
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonDegraded),
							Message: "Spec.Rules.BackendRef.Name must be specified",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.RouteReasonAccepted),
							Message: "Accepted TLSRoute",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate(string(gw.Spec.Listeners[0].Name), "TLSRoute", 0),
	})

	run(t, "TLSRoute: spec.rules.backendRef.name invalid on two matches", testcase{
		gateway: gw,
		objs: []interface{}{
			&gatewayapi_v1alpha2.TLSRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1alpha2.TLSRouteSpec{
					CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1alpha2.ParentReference{
							gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour"),
						},
					},
					Hostnames: []gatewayapi_v1alpha2.Hostname{"test.projectcontour.io"},
					Rules: []gatewayapi_v1alpha2.TLSRouteRule{
						{BackendRefs: gatewayapi.TLSRouteBackendRef("invalid-one", 8080, nil)},
						{BackendRefs: gatewayapi.TLSRouteBackendRef("invalid-two", 8080, nil)},
					},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.RouteReasonBackendNotFound),
							Message: "service \"invalid-one\" is invalid: service \"default/invalid-one\" not found, service \"invalid-two\" is invalid: service \"default/invalid-two\" not found",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.RouteReasonAccepted),
							Message: "Accepted TLSRoute",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate(string(gw.Spec.Listeners[0].Name), "TLSRoute", 0),
	})

	run(t, "TLSRoute: spec.rules.backendRef.port not specified", testcase{
		gateway: gw,
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1alpha2.TLSRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1alpha2.TLSRouteSpec{
					CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1alpha2.ParentReference{
							gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour"),
						},
					},
					Hostnames: []gatewayapi_v1alpha2.Hostname{"test.projectcontour.io"},
					Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
						BackendRefs: []gatewayapi_v1alpha2.BackendRef{
							{
								BackendObjectReference: gatewayapi_v1alpha2.BackendObjectReference{
									Kind: gatewayapi.KindPtrV1Alpha2("Service"),
									Name: "kuard",
								},
							},
						},
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonDegraded),
							Message: "Spec.Rules.BackendRef.Port must be specified",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.RouteReasonAccepted),
							Message: "Accepted TLSRoute",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate(string(gw.Spec.Listeners[0].Name), "TLSRoute", 0),
	})

	run(t, "TLSRoute: spec.rules.backendRefs not specified", testcase{
		gateway: gw,
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1alpha2.TLSRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1alpha2.TLSRouteSpec{
					CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1alpha2.ParentReference{
							gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour"),
						},
					},
					Rules: []gatewayapi_v1alpha2.TLSRouteRule{
						{}, // rule with no backend refs
					},
					Hostnames: []gatewayapi_v1alpha2.Hostname{"test.projectcontour.io"},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonDegraded),
							Message: "At least one Spec.Rules.BackendRef must be specified.",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.RouteReasonAccepted),
							Message: "Accepted TLSRoute",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate(string(gw.Spec.Listeners[0].Name), "TLSRoute", 0),
	})

	run(t, "TLSRoute: spec.rules.hostname: invalid wildcard", testcase{
		gateway: gw,
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1alpha2.TLSRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1alpha2.TLSRouteSpec{
					CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1alpha2.ParentReference{
							gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour"),
						},
					},
					Hostnames: []gatewayapi_v1alpha2.Hostname{"*.*.projectcontour.io"},
					Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
						BackendRefs: gatewayapi.TLSRouteBackendRef("kuard", 8080, nil),
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonDegraded),
							Message: "invalid hostname \"*.*.projectcontour.io\": [a wildcard DNS-1123 subdomain must start with '*.', followed by a valid DNS subdomain, which must consist of lower case alphanumeric characters, '-' or '.' and end with an alphanumeric character (e.g. '*.example.com', regex used for validation is '\\*\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')]",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.RouteReasonNoMatchingListenerHostname),
							Message: "No intersecting hostnames were found between the listener and the route.",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate(string(gw.Spec.Listeners[0].Name), "TLSRoute", 0),
	})

	run(t, "TLSRoute: spec.rules.hostname: invalid hostname", testcase{
		gateway: gw,
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1alpha2.TLSRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1alpha2.TLSRouteSpec{
					CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1alpha2.ParentReference{
							gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour"),
						},
					},
					Hostnames: []gatewayapi_v1alpha2.Hostname{"#projectcontour.io"},
					Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
						BackendRefs: gatewayapi.TLSRouteBackendRef("kuard", 8080, nil),
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonDegraded),
							Message: "invalid hostname \"#projectcontour.io\": [a lowercase RFC 1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character (e.g. 'example.com', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')]",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.RouteReasonNoMatchingListenerHostname),
							Message: "No intersecting hostnames were found between the listener and the route.",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate(string(gw.Spec.Listeners[0].Name), "TLSRoute", 0),
	})

	run(t, "TLSRoute: spec.rules.hostname: invalid hostname, ip address", testcase{
		gateway: gw,
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1alpha2.TLSRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
					Labels: map[string]string{
						"app": "contour",
					},
				},
				Spec: gatewayapi_v1alpha2.TLSRouteSpec{
					CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1alpha2.ParentReference{
							gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour"),
						},
					},
					Hostnames: []gatewayapi_v1alpha2.Hostname{"1.2.3.4"},
					Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
						BackendRefs: gatewayapi.TLSRouteBackendRef("kuard", 8080, nil),
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionResolvedRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonDegraded),
							Message: "invalid hostname \"1.2.3.4\": must be a DNS name, not an IP address",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionFalse,
							Reason:  string(gatewayapi_v1beta1.RouteReasonNoMatchingListenerHostname),
							Message: "No intersecting hostnames were found between the listener and the route.",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate(string(gw.Spec.Listeners[0].Name), "TLSRoute", 0),
	})

	run(t, "TLSRoute: spec.rules.backendRefs has 0 weight", testcase{
		gateway: gw,
		objs: []interface{}{
			kuardService,
			&gatewayapi_v1alpha2.TLSRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "default",
				},
				Spec: gatewayapi_v1alpha2.TLSRouteSpec{
					CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
						ParentRefs: []gatewayapi_v1alpha2.ParentReference{
							gatewayapi.GatewayParentRefV1Alpha2(gw.Namespace, gw.Name),
						},
					},
					Hostnames: []gatewayapi_v1alpha2.Hostname{"test.projectcontour.io"},
					Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
						BackendRefs: gatewayapi.TLSRouteBackendRef(kuardService.Name, 8080, pointer.Int32(0)),
					}},
				},
			}},
		wantRouteConditions: []*status.RouteStatusUpdate{{
			FullName: types.NamespacedName{Namespace: "default", Name: "basic"},
			RouteParentStatuses: []*gatewayapi_v1beta1.RouteParentStatus{
				{
					ParentRef: gatewayapi.GatewayParentRef("projectcontour", "contour"),
					Conditions: []metav1.Condition{
						{
							Type:    string(status.ConditionValidBackendRefs),
							Status:  metav1.ConditionFalse,
							Reason:  string(status.ReasonAllBackendRefsHaveZeroWeights),
							Message: "At least one Spec.Rules.BackendRef must have a non-zero weight.",
						},
						{
							Type:    string(gatewayapi_v1beta1.RouteConditionAccepted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gatewayapi_v1beta1.RouteReasonAccepted),
							Message: "Accepted TLSRoute",
						},
					},
				},
			},
		}},
		wantGatewayStatusUpdate: validGatewayStatusUpdate(string(gw.Spec.Listeners[0].Name), "TLSRoute", 0),
	})
}

func gatewayScheduledCondition() metav1.Condition {
	return metav1.Condition{
		Type:    string(gatewayapi_v1beta1.GatewayConditionScheduled),
		Status:  metav1.ConditionTrue,
		Reason:  string(gatewayapi_v1beta1.GatewayReasonScheduled),
		Message: "Gateway is scheduled",
	}
}
