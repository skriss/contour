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

package k8s

import (
	"testing"

	"github.com/projectcontour/contour/internal/fixture"
	"github.com/projectcontour/contour/internal/gatewayapi"
	"github.com/projectcontour/contour/internal/k8s/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapi_v1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func TestServiceStatusLoadBalancerWatcherOnAdd(t *testing.T) {
	lbstatus := make(chan v1.LoadBalancerStatus, 1)
	sw := ServiceStatusLoadBalancerWatcher{
		ServiceName: "envoy",
		LBStatus:    lbstatus,
		Log:         fixture.NewTestLogger(t),
	}

	recv := func() (v1.LoadBalancerStatus, bool) {
		select {
		case lbs := <-sw.LBStatus:
			return lbs, true
		default:
			return v1.LoadBalancerStatus{}, false
		}
	}

	// assert adding something other than a service generates no notification.
	sw.OnAdd(&v1.Pod{})
	_, ok := recv()
	if ok {
		t.Fatalf("expected no result when adding")
	}

	// assert adding a service with an different name generates no notification
	var svc v1.Service
	svc.Name = "potato"
	sw.OnAdd(&svc)
	_, ok = recv()
	if ok {
		t.Fatalf("expected no result when adding a service with a different name")
	}

	// assert adding a service with the correct name generates a notification
	svc.Name = sw.ServiceName
	svc.Status.LoadBalancer.Ingress = []v1.LoadBalancerIngress{{Hostname: "projectcontour.io"}}
	sw.OnAdd(&svc)
	got, ok := recv()
	if !ok {
		t.Fatalf("expected result when adding a service with the correct name")
	}
	want := v1.LoadBalancerStatus{
		Ingress: []v1.LoadBalancerIngress{{Hostname: "projectcontour.io"}},
	}
	assert.Equal(t, got, want)
}

func TestServiceStatusLoadBalancerWatcherOnUpdate(t *testing.T) {
	lbstatus := make(chan v1.LoadBalancerStatus, 1)

	sw := ServiceStatusLoadBalancerWatcher{
		ServiceName: "envoy",
		LBStatus:    lbstatus,
		Log:         fixture.NewTestLogger(t),
	}

	recv := func() (v1.LoadBalancerStatus, bool) {
		select {
		case lbs := <-sw.LBStatus:
			return lbs, true
		default:
			return v1.LoadBalancerStatus{}, false
		}
	}

	// assert updating something other than a service generates no notification.
	sw.OnUpdate(&v1.Pod{}, &v1.Pod{})
	_, ok := recv()
	if ok {
		t.Fatalf("expected no result when updating")
	}

	// assert updating a service with an different name generates no notification
	var oldSvc, newSvc v1.Service
	oldSvc.Name = "potato"
	newSvc.Name = "elephant"
	sw.OnUpdate(&oldSvc, &newSvc)
	_, ok = recv()
	if ok {
		t.Fatalf("expected no result when updating a service with a different name")
	}

	// assert updating a service with the correct name generates a notification
	var svc v1.Service
	svc.Name = sw.ServiceName
	svc.Status.LoadBalancer.Ingress = []v1.LoadBalancerIngress{{Hostname: "projectcontour.io"}}
	sw.OnUpdate(&oldSvc, &svc)
	got, ok := recv()
	if !ok {
		t.Fatalf("expected result when updating a service with the correct name")
	}
	want := v1.LoadBalancerStatus{
		Ingress: []v1.LoadBalancerIngress{{Hostname: "projectcontour.io"}},
	}
	assert.Equal(t, got, want)
}

func TestServiceStatusLoadBalancerWatcherOnDelete(t *testing.T) {
	lbstatus := make(chan v1.LoadBalancerStatus, 1)

	sw := ServiceStatusLoadBalancerWatcher{
		ServiceName: "envoy",
		LBStatus:    lbstatus,
		Log:         fixture.NewTestLogger(t),
	}

	recv := func() (v1.LoadBalancerStatus, bool) {
		select {
		case lbs := <-sw.LBStatus:
			return lbs, true
		default:
			return v1.LoadBalancerStatus{}, false
		}
	}

	// assert deleting something other than a service generates no notification.
	sw.OnDelete(&v1.Pod{})
	_, ok := recv()
	if ok {
		t.Fatalf("expected no result when deleting")
	}

	// assert adding a service with an different name generates no notification
	var svc v1.Service
	svc.Name = "potato"
	sw.OnDelete(&svc)
	_, ok = recv()
	if ok {
		t.Fatalf("expected no result when deleting a service with a different name")
	}

	// assert deleting a service with the correct name generates a blank notification
	svc.Name = sw.ServiceName
	svc.Status.LoadBalancer.Ingress = []v1.LoadBalancerIngress{{Hostname: "projectcontour.io"}}
	sw.OnDelete(&svc)
	got, ok := recv()
	if !ok {
		t.Fatalf("expected result when deleting a service with the correct name")
	}
	want := v1.LoadBalancerStatus{
		Ingress: nil,
	}
	assert.Equal(t, got, want)
}

//go:generate go run github.com/vektra/mockery/v2 --case=snake --name=Cache --srcpkg=sigs.k8s.io/controller-runtime/pkg/cache
func TestStatusAddressUpdater_Gateway(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	ipLBStatus := v1.LoadBalancerStatus{
		Ingress: []v1.LoadBalancerIngress{
			{
				IP: "127.0.0.1",
			},
		},
	}

	hostnameLBStatus := v1.LoadBalancerStatus{
		Ingress: []v1.LoadBalancerIngress{
			{
				Hostname: "ingress.projectcontour.io",
			},
		},
	}

	testCases := map[string]struct {
		status                     v1.LoadBalancerStatus
		gatewayClassControllerName string
		gatewayRef                 *types.NamespacedName
		preop                      *gatewayapi_v1beta1.Gateway
		postop                     *gatewayapi_v1beta1.Gateway
	}{
		"happy path (IP)": {
			status:                     ipLBStatus,
			gatewayClassControllerName: "projectcontour.io/contour",
			preop: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "projectcontour",
					Name:      "contour-gateway",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					GatewayClassName: gatewayapi_v1beta1.ObjectName("contour-gatewayclass"),
				},
				Status: gatewayapi_v1beta1.GatewayStatus{
					Conditions: []metav1.Condition{
						{
							Type:   string(gatewayapi_v1beta1.GatewayConditionReady),
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
			postop: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "projectcontour",
					Name:      "contour-gateway",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					GatewayClassName: gatewayapi_v1beta1.ObjectName("contour-gatewayclass"),
				},
				Status: gatewayapi_v1beta1.GatewayStatus{
					Conditions: []metav1.Condition{
						{
							Type:   string(gatewayapi_v1beta1.GatewayConditionReady),
							Status: metav1.ConditionTrue,
						},
					},
					Addresses: []gatewayapi_v1beta1.GatewayAddress{
						{
							Type:  gatewayapi.AddressTypePtr(gatewayapi_v1beta1.IPAddressType),
							Value: ipLBStatus.Ingress[0].IP,
						},
					},
				},
			},
		},
		"happy path (hostname)": {
			status:                     hostnameLBStatus,
			gatewayClassControllerName: "projectcontour.io/contour",
			preop: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "projectcontour",
					Name:      "contour-gateway",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					GatewayClassName: gatewayapi_v1beta1.ObjectName("contour-gatewayclass"),
				},
				Status: gatewayapi_v1beta1.GatewayStatus{
					Conditions: []metav1.Condition{
						{
							Type:   string(gatewayapi_v1beta1.GatewayConditionReady),
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
			postop: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "projectcontour",
					Name:      "contour-gateway",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					GatewayClassName: gatewayapi_v1beta1.ObjectName("contour-gatewayclass"),
				},
				Status: gatewayapi_v1beta1.GatewayStatus{
					Conditions: []metav1.Condition{
						{
							Type:   string(gatewayapi_v1beta1.GatewayConditionReady),
							Status: metav1.ConditionTrue,
						},
					},
					Addresses: []gatewayapi_v1beta1.GatewayAddress{
						{
							Type:  gatewayapi.AddressTypePtr(gatewayapi_v1beta1.HostnameAddressType),
							Value: hostnameLBStatus.Ingress[0].Hostname,
						},
					},
				},
			},
		},
		"Gateway not controlled by this Contour": {
			status:                     ipLBStatus,
			gatewayClassControllerName: "projectcontour.io/some-other-controller",
			preop: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "projectcontour",
					Name:      "contour-gateway",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					GatewayClassName: gatewayapi_v1beta1.ObjectName("contour-gatewayclass"),
				},
				Status: gatewayapi_v1beta1.GatewayStatus{
					Conditions: []metav1.Condition{
						{
							Type:   string(gatewayapi_v1beta1.GatewayConditionReady),
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
			postop: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "projectcontour",
					Name:      "contour-gateway",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					GatewayClassName: gatewayapi_v1beta1.ObjectName("contour-gatewayclass"),
				},
				Status: gatewayapi_v1beta1.GatewayStatus{
					Conditions: []metav1.Condition{
						{
							Type:   string(gatewayapi_v1beta1.GatewayConditionReady),
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
		},
		"Specific gateway configured, gateway does not match": {
			status:     ipLBStatus,
			gatewayRef: &types.NamespacedName{Namespace: "projectcontour", Name: "contour-gateway"},
			preop: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "projectcontour",
					Name:      "some-other-gateway",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					GatewayClassName: gatewayapi_v1beta1.ObjectName("contour-gatewayclass"),
				},
				Status: gatewayapi_v1beta1.GatewayStatus{
					Conditions: []metav1.Condition{
						{
							Type:   string(gatewayapi_v1beta1.GatewayConditionReady),
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
			postop: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "projectcontour",
					Name:      "some-other-gateway",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					GatewayClassName: gatewayapi_v1beta1.ObjectName("contour-gatewayclass"),
				},
				Status: gatewayapi_v1beta1.GatewayStatus{
					Conditions: []metav1.Condition{
						{
							Type:   string(gatewayapi_v1beta1.GatewayConditionReady),
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
		},
		"Specific gateway configured, gateway matches": {
			status:     ipLBStatus,
			gatewayRef: &types.NamespacedName{Namespace: "projectcontour", Name: "contour-gateway"},
			preop: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "projectcontour",
					Name:      "contour-gateway",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					GatewayClassName: gatewayapi_v1beta1.ObjectName("contour-gatewayclass"),
				},
				Status: gatewayapi_v1beta1.GatewayStatus{
					Conditions: []metav1.Condition{
						{
							Type:   string(gatewayapi_v1beta1.GatewayConditionReady),
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
			postop: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "projectcontour",
					Name:      "contour-gateway",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					GatewayClassName: gatewayapi_v1beta1.ObjectName("contour-gatewayclass"),
				},
				Status: gatewayapi_v1beta1.GatewayStatus{
					Conditions: []metav1.Condition{
						{
							Type:   string(gatewayapi_v1beta1.GatewayConditionReady),
							Status: metav1.ConditionTrue,
						},
					},
					Addresses: []gatewayapi_v1beta1.GatewayAddress{
						{
							Type:  gatewayapi.AddressTypePtr(gatewayapi_v1beta1.IPAddressType),
							Value: ipLBStatus.Ingress[0].IP,
						},
					},
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name+" OnAdd", func(t *testing.T) {
			suc := StatusUpdateCacher{}
			assert.True(t, suc.Add(tc.preop.Name, tc.preop.Namespace, tc.preop), "unable to add object to cache")

			mockCache := &mocks.Cache{}
			mockCache.
				On("Get", mock.Anything, client.ObjectKey{Name: string(tc.preop.Spec.GatewayClassName)}, mock.Anything).
				Run(func(args mock.Arguments) {
					// The cache's Get function takes a pointer to a struct and updates it
					// with the data from the API server; this simulates that behavior by
					// updating the struct pointed to by the third argument with the fields
					// we care about. See Run's godoc for more info.
					args[2].(*gatewayapi_v1beta1.GatewayClass).Spec.ControllerName = gatewayapi_v1beta1.GatewayController(tc.gatewayClassControllerName)
				}).
				Return(nil)

			isu := StatusAddressUpdater{
				Logger:                log,
				GatewayControllerName: "projectcontour.io/contour",
				GatewayRef:            tc.gatewayRef,
				Cache:                 mockCache,
				LBStatus:              tc.status,
				StatusUpdater:         &suc,
			}

			isu.OnAdd(tc.preop)

			newObj := suc.Get(tc.preop.Name, tc.preop.Namespace)
			assert.Equal(t, tc.postop, newObj)
		})

		t.Run(name+" OnUpdate", func(t *testing.T) {
			suc := StatusUpdateCacher{}
			assert.True(t, suc.Add(tc.preop.Name, tc.preop.Namespace, tc.preop), "unable to add object to cache")

			mockCache := &mocks.Cache{}
			mockCache.
				On("Get", mock.Anything, client.ObjectKey{Name: string(tc.preop.Spec.GatewayClassName)}, mock.Anything).
				Run(func(args mock.Arguments) {
					// The cache's Get function takes a pointer to a struct and updates it
					// with the data from the API server; this simulates that behavior by
					// updating the struct pointed to by the third argument with the fields
					// we care about. See Run's godoc for more info.
					args[2].(*gatewayapi_v1beta1.GatewayClass).Spec.ControllerName = gatewayapi_v1beta1.GatewayController(tc.gatewayClassControllerName)
				}).
				Return(nil)

			isu := StatusAddressUpdater{
				Logger:                log,
				GatewayControllerName: "projectcontour.io/contour",
				GatewayRef:            tc.gatewayRef,
				Cache:                 mockCache,
				LBStatus:              tc.status,
				StatusUpdater:         &suc,
			}

			isu.OnUpdate(tc.preop, tc.preop)

			newObj := suc.Get(tc.preop.Name, tc.preop.Namespace)
			assert.Equal(t, tc.postop, newObj)
		})
	}
}
