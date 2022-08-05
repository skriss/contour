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
	"net/http"
	"testing"

	"github.com/projectcontour/contour/internal/fixture"
	"github.com/projectcontour/contour/internal/gatewayapi"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	gatewayapi_v1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayapi_v1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func TestDAGInsertGatewayAPI(t *testing.T) {
	kuardService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "projectcontour",
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
			Namespace: "projectcontour",
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
			Namespace: "projectcontour",
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

	kuardServiceCustomNs := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "custom",
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

	blogService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "blogsvc",
			Namespace: "projectcontour",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}

	validClass := &gatewayapi_v1beta1.GatewayClass{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-validClass",
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
	}

	gatewayHTTPAllNamespaces := &gatewayapi_v1beta1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "contour",
			Namespace: "projectcontour",
		},
		Spec: gatewayapi_v1beta1.GatewaySpec{
			GatewayClassName: gatewayapi_v1beta1.ObjectName(validClass.Name),
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

	gatewayHTTPSameNamespace := &gatewayapi_v1beta1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "contour",
			Namespace: "projectcontour",
		},
		Spec: gatewayapi_v1beta1.GatewaySpec{
			GatewayClassName: gatewayapi_v1beta1.ObjectName(validClass.Name),
			Listeners: []gatewayapi_v1beta1.Listener{{
				Name:     "http",
				Port:     80,
				Protocol: gatewayapi_v1beta1.HTTPProtocolType,
				AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
					Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
						From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromSame),
					},
				},
			}},
		},
	}

	gatewayHTTPNamespaceSelector := &gatewayapi_v1beta1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "contour",
			Namespace: "projectcontour",
		},
		Spec: gatewayapi_v1beta1.GatewaySpec{
			GatewayClassName: gatewayapi_v1beta1.ObjectName(validClass.Name),
			Listeners: []gatewayapi_v1beta1.Listener{{
				Port:     80,
				Protocol: gatewayapi_v1beta1.HTTPProtocolType,
				AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
					Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
						From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromSelector),
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "contour",
							},
							MatchExpressions: []metav1.LabelSelectorRequirement{{
								Key:      "type",
								Operator: "In",
								Values:   []string{"controller"},
							}},
						},
					},
				},
			}},
		},
	}

	hostname := gatewayapi_v1beta1.Hostname("gateway.projectcontour.io")
	wildcardHostname := gatewayapi_v1beta1.Hostname("*.projectcontour.io")

	gatewayHTTPWithHostname := &gatewayapi_v1beta1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "contour",
			Namespace: "projectcontour",
		},
		Spec: gatewayapi_v1beta1.GatewaySpec{
			GatewayClassName: gatewayapi_v1beta1.ObjectName(validClass.Name),
			Listeners: []gatewayapi_v1beta1.Listener{{
				Port:     80,
				Hostname: &hostname,
				Protocol: gatewayapi_v1beta1.HTTPProtocolType,
				AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
					Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
						From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
					},
				},
			}},
		},
	}

	gatewayHTTPWithWildcardHostname := &gatewayapi_v1beta1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "contour",
			Namespace: "projectcontour",
		},
		Spec: gatewayapi_v1beta1.GatewaySpec{
			GatewayClassName: gatewayapi_v1beta1.ObjectName(validClass.Name),
			Listeners: []gatewayapi_v1beta1.Listener{{
				Port:     80,
				Hostname: &wildcardHostname,
				Protocol: gatewayapi_v1beta1.HTTPProtocolType,
				AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
					Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
						From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
					},
				},
			}},
		},
	}

	gatewayHTTPWithAddresses := &gatewayapi_v1beta1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "contour",
			Namespace: "projectcontour",
		},
		Spec: gatewayapi_v1beta1.GatewaySpec{
			GatewayClassName: gatewayapi_v1beta1.ObjectName(validClass.Name),
			Addresses: []gatewayapi_v1beta1.GatewayAddress{
				{
					Type:  gatewayapi.GatewayAddressTypePtr(gatewayapi_v1beta1.IPAddressType),
					Value: "1.2.3.4",
				},
			},
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

	gatewayTLSPassthroughAllNamespaces := &gatewayapi_v1beta1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "contour",
			Namespace: "projectcontour",
		},
		Spec: gatewayapi_v1beta1.GatewaySpec{
			GatewayClassName: gatewayapi_v1beta1.ObjectName(validClass.Name),
			Listeners: []gatewayapi_v1beta1.Listener{{
				Port:     80,
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

	gatewayTLSPassthroughSameNamespace := &gatewayapi_v1beta1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "contour",
			Namespace: "projectcontour",
		},
		Spec: gatewayapi_v1beta1.GatewaySpec{
			GatewayClassName: gatewayapi_v1beta1.ObjectName(validClass.Name),
			Listeners: []gatewayapi_v1beta1.Listener{{
				Port:     80,
				Protocol: gatewayapi_v1beta1.TLSProtocolType,
				TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
					Mode: gatewayapi.TLSModeTypePtr(gatewayapi_v1beta1.TLSModePassthrough),
				},
				AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
					Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
						From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromSame),
					},
				},
			}},
		},
	}

	gatewayTLSPassthroughNamespaceSelector := &gatewayapi_v1beta1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "contour",
			Namespace: "projectcontour",
		},
		Spec: gatewayapi_v1beta1.GatewaySpec{
			GatewayClassName: gatewayapi_v1beta1.ObjectName(validClass.Name),
			Listeners: []gatewayapi_v1beta1.Listener{{
				Port:     80,
				Protocol: gatewayapi_v1beta1.TLSProtocolType,
				TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
					Mode: gatewayapi.TLSModeTypePtr(gatewayapi_v1beta1.TLSModePassthrough),
				},
				AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
					Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
						From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromSelector),
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"matching-label-key": "matching-label-value"},
						},
					},
				},
			}},
		},
	}

	sec1 := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "projectcontour",
		},
		Type: v1.SecretTypeTLS,
		Data: secretdata(fixture.CERTIFICATE, fixture.RSA_PRIVATE_KEY),
	}

	sec2 := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "tls-cert-namespace",
		},
		Type: v1.SecretTypeTLS,
		Data: secretdata(fixture.CERTIFICATE, fixture.RSA_PRIVATE_KEY),
	}

	gatewayTLSTerminateCertInDifferentNamespace := &gatewayapi_v1beta1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "contour",
			Namespace: "projectcontour",
		},
		Spec: gatewayapi_v1beta1.GatewaySpec{
			GatewayClassName: gatewayapi_v1beta1.ObjectName(validClass.Name),
			Listeners: []gatewayapi_v1beta1.Listener{{
				Name:     "https",
				Port:     443,
				Protocol: gatewayapi_v1beta1.HTTPSProtocolType,
				TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
					Mode: gatewayapi.TLSModeTypePtr(gatewayapi_v1beta1.TLSModeTerminate),
					CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
						gatewayapi.CertificateRef(sec2.Name, sec2.Namespace),
					},
				},
				AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
					Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
						From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
					},
				},
			}},
		},
	}

	gatewayHTTPSAllNamespaces := &gatewayapi_v1beta1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "contour",
			Namespace: "projectcontour",
		},
		Spec: gatewayapi_v1beta1.GatewaySpec{
			GatewayClassName: gatewayapi_v1beta1.ObjectName(validClass.Name),
			Listeners: []gatewayapi_v1beta1.Listener{{
				Port:     443,
				Protocol: gatewayapi_v1beta1.HTTPSProtocolType,
				TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
					CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
						gatewayapi.CertificateRef(sec1.Name, sec1.Namespace),
					},
				},
				AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
					Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
						From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
					},
				},
			}},
		},
	}

	gatewayHTTPAndHTTPS := &gatewayapi_v1beta1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "contour",
			Namespace: "projectcontour",
		},
		Spec: gatewayapi_v1beta1.GatewaySpec{
			GatewayClassName: gatewayapi_v1beta1.ObjectName(validClass.Name),
			Listeners: []gatewayapi_v1beta1.Listener{
				{
					Name:     "http-listener",
					Port:     80,
					Protocol: gatewayapi_v1beta1.HTTPProtocolType,
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
				},
				{
					Name:     "https-listener",
					Port:     443,
					Protocol: gatewayapi_v1beta1.HTTPSProtocolType,
					TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
						CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
							gatewayapi.CertificateRef(sec1.Name, sec1.Namespace),
						},
					},
					AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
						Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
							From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
						},
					},
				},
			},
		},
	}

	basicHTTPRoute := &gatewayapi_v1beta1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "basic",
			Namespace: "projectcontour",
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
	}

	basicTLSRoute := &gatewayapi_v1alpha2.TLSRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "basic",
			Namespace: "projectcontour",
		},
		Spec: gatewayapi_v1alpha2.TLSRouteSpec{
			CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
				ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
			},
			Hostnames: []gatewayapi_v1alpha2.Hostname{"test.projectcontour.io"},
			Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
				BackendRefs: gatewayapi.TLSRouteBackendRef("kuard", 8080, nil),
			}},
		},
	}

	tests := map[string]struct {
		objs         []interface{}
		gatewayclass *gatewayapi_v1beta1.GatewayClass
		gateway      *gatewayapi_v1beta1.Gateway
		want         []*Listener
	}{
		"insert basic single route, single hostname": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				basicHTTPRoute,
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("test.projectcontour.io", prefixrouteHTTPRoute("/", service(kuardService))),
					),
				},
			),
		},
		"gateway with addresses is unsupported": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPWithAddresses,
			objs: []interface{}{
				kuardService,
				basicHTTPRoute,
			},
			want: listeners(),
		},
		"gateway without a gatewayclass": {
			gateway: gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				basicHTTPRoute,
			},
			want: listeners(),
		},
		"insert basic single route, single hostname, gateway same namespace selector, route in gateway's namespace": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPSameNamespace,
			objs: []interface{}{
				kuardService,
				basicHTTPRoute,
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("test.projectcontour.io", prefixrouteHTTPRoute("/", service(kuardService))),
					),
				},
			),
		},
		"insert basic single route, single hostname, gateway same namespace selector, route in different namespace": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPSameNamespace,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "different-ns-than-gateway",
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
			},
			want: listeners(),
		},
		"insert basic single route, single hostname, gateway From namespace selector": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPNamespaceSelector,
			objs: []interface{}{
				&v1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "custom",
						Labels: map[string]string{
							"app":  "contour",
							"type": "controller",
						},
					},
				},
				kuardServiceCustomNs,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "custom",
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
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("test.projectcontour.io", prefixrouteHTTPRoute("/", service(kuardServiceCustomNs))),
					),
				},
			),
		},
		"insert basic single route, single hostname, gateway From namespace selector, not matching": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPNamespaceSelector,
			objs: []interface{}{
				&v1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "custom",
						Labels: map[string]string{
							"app":  "notmatch",
							"type": "someother",
						},
					},
				},
				kuardServiceCustomNs,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "custom",
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
			},
			want: listeners(),
		},

		"HTTPRoute does not include the gateway in its list of parent refs": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{
						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{
								gatewayapi.GatewayParentRef("projectcontour", "some-other-gateway"),
								gatewayapi.GatewayParentRef("projectcontour", "some-other-gateway-2"),
							},
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
			},
			want: listeners(),
		},

		// BEGIN TLSRoute<->Gateway selection test cases
		"TLSRoute: Gateway selects TLSRoutes in all namespaces": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughAllNamespaces,
			objs: []interface{}{
				kuardServiceCustomNs,
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "custom",
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{"test.projectcontour.io"},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: gatewayapi.TLSRouteBackendRef("kuard", 8080, nil),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTPS_LISTENER_NAME,
					Port: 443,
					SecureVirtualHosts: securevirtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "test.projectcontour.io",
							},
							TCPProxy: &TCPProxy{
								Clusters: clustersWeight(service(kuardServiceCustomNs)),
							},
						},
					),
				},
			),
		},
		"TLSRoute: Gateway selects TLSRoutes in same namespace, and route is in the same namespace": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughSameNamespace,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: gatewayTLSPassthroughSameNamespace.Namespace,
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{"test.projectcontour.io"},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: gatewayapi.TLSRouteBackendRef("kuard", 8080, nil),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTPS_LISTENER_NAME,
					Port: 443,
					SecureVirtualHosts: securevirtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "test.projectcontour.io",
							},
							TCPProxy: &TCPProxy{
								Clusters: clustersWeight(service(kuardService)),
							},
						},
					),
				},
			),
		},
		"TLSRoute: Gateway selects TLSRoutes in same namespace, and route is not in the same namespace": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughSameNamespace,
			objs: []interface{}{
				kuardServiceCustomNs,
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: kuardServiceCustomNs.Namespace,
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{"test.projectcontour.io"},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: gatewayapi.TLSRouteBackendRef("kuard", 8080, nil),
						}},
					},
				},
			},
			want: listeners(),
		},
		"TLSRoute: Gateway selects TLSRoutes in namespaces matching selector, and route is in a matching namespace": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughNamespaceSelector,
			objs: []interface{}{
				kuardServiceCustomNs,
				&v1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   kuardServiceCustomNs.Namespace,
						Labels: map[string]string{"matching-label-key": "matching-label-value"},
					},
				},
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: kuardServiceCustomNs.Namespace,
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{"test.projectcontour.io"},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: gatewayapi.TLSRouteBackendRef("kuard", 8080, nil),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTPS_LISTENER_NAME,
					Port: 443,
					SecureVirtualHosts: securevirtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "test.projectcontour.io",
							},
							TCPProxy: &TCPProxy{
								Clusters: clustersWeight(service(kuardServiceCustomNs)),
							},
						},
					),
				},
			),
		},
		"TLSRoute: Gateway selects TLSRoutes in namespaces matching selector, and route is in a non-matching namespace": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughNamespaceSelector,
			objs: []interface{}{
				kuardServiceCustomNs,
				&v1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   kuardServiceCustomNs.Namespace,
						Labels: map[string]string{"matching-label-key": "this-label-value-does-not-match"},
					},
				},
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: kuardServiceCustomNs.Namespace,
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{"test.projectcontour.io"},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: gatewayapi.TLSRouteBackendRef("kuard", 8080, nil),
						}},
					},
				},
			},
			want: listeners(),
		},

		"TLSRoute: Gateway selects non-TLSRoutes": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces, // selects HTTPRoutes, not TLSRoutes
			objs: []interface{}{
				kuardService,
				basicTLSRoute,
			},
			want: listeners(),
		},

		"TLSRoute: TLSRoute allows Gateways from list, and gateway is not in the list": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: kuardService.Namespace,
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("some-other-namespace", "some-other-gateway-name")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{"test.projectcontour.io"},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: gatewayapi.TLSRouteBackendRef("kuard", 8080, nil),
						}},
					},
				},
			},
			want: listeners(),
		},
		// END TLSRoute<->Gateway selection test cases

		"TLS Listener with TLS.Mode=Passthrough is invalid if certificateRef is specified": {
			gatewayclass: validClass,
			gateway: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "contour",
					Namespace: "projectcontour",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					Listeners: []gatewayapi_v1beta1.Listener{{
						Port:     80,
						Protocol: gatewayapi_v1beta1.TLSProtocolType,
						TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
							Mode: gatewayapi.TLSModeTypePtr(gatewayapi_v1beta1.TLSModePassthrough),
							CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
								gatewayapi.CertificateRef(sec1.Name, sec1.Namespace),
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
				kuardService,
				basicTLSRoute,
			},
			want: listeners(),
		},
		"TLS Listener with TLS.Mode=Terminate is invalid": {
			gatewayclass: validClass,
			gateway: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "contour",
					Namespace: "projectcontour",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					GatewayClassName: gatewayapi_v1beta1.ObjectName(validClass.Name),
					Listeners: []gatewayapi_v1beta1.Listener{{
						Port:     80,
						Protocol: gatewayapi_v1beta1.TLSProtocolType,
						TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
							Mode: gatewayapi.TLSModeTypePtr(gatewayapi_v1beta1.TLSModeTerminate),
							CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
								gatewayapi.CertificateRef(sec1.Name, sec1.Namespace),
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
				sec1,
				kuardService,
				basicTLSRoute,
			},
			want: listeners(),
		},
		"TLS Listener with TLS not defined is invalid": {
			gatewayclass: validClass,
			gateway: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "contour",
					Namespace: "projectcontour",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					Listeners: []gatewayapi_v1beta1.Listener{{
						Port:     80,
						Protocol: gatewayapi_v1beta1.TLSProtocolType,
						AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
							Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
								From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
							},
						},
					}},
				},
			},
			objs: []interface{}{
				kuardService,
				basicTLSRoute,
			},
			want: listeners(),
		},
		"TLSRoute with invalid listener protocol of HTTP": {
			gatewayclass: validClass,
			gateway: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "contour",
					Namespace: "projectcontour",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					Listeners: []gatewayapi_v1beta1.Listener{{
						Port:     80,
						Protocol: gatewayapi_v1beta1.HTTPProtocolType,
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
			},
			objs: []interface{}{
				kuardService,
				basicTLSRoute,
			},
			want: listeners(),
		},
		"TLSRoute with invalid listener kind": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				basicTLSRoute,
			},
			want: listeners(),
		},
		// Issue: https://github.com/projectcontour/contour/issues/3591
		"one gateway with two httproutes, different hostnames": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				basicHTTPRoute,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic-two",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{
						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1beta1.Hostname{
							"another.projectcontour.io",
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("another.projectcontour.io", prefixrouteHTTPRoute("/", service(kuardService))),
						virtualhost("test.projectcontour.io", prefixrouteHTTPRoute("/", service(kuardService))),
					),
				},
			),
		},
		"insert gateway with selector kind that doesn't match": {
			gatewayclass: validClass,
			gateway: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "contour",
					Namespace: "projectcontour",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					Listeners: []gatewayapi_v1beta1.Listener{{
						Port:     80,
						Protocol: gatewayapi_v1beta1.HTTPProtocolType,
						AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
							Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
								From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
							},
							Kinds: []gatewayapi_v1beta1.RouteGroupKind{
								{
									Group: gatewayapi.GroupPtr(gatewayapi_v1alpha2.GroupName),
									Kind:  gatewayapi_v1beta1.Kind("INVALID-KIND"),
								},
							},
						},
					}},
				},
			},
			objs: []interface{}{
				kuardService,
				basicHTTPRoute,
			},
			want: listeners(),
		},
		"insert gateway with selector group that doesn't match": {
			gatewayclass: validClass,
			gateway: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "contour",
					Namespace: "projectcontour",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					Listeners: []gatewayapi_v1beta1.Listener{{
						Port:     80,
						Protocol: gatewayapi_v1beta1.HTTPProtocolType,
						AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
							Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
								From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
							},
							Kinds: []gatewayapi_v1beta1.RouteGroupKind{
								{
									Group: gatewayapi.GroupPtr("invalid-group-name"),
									Kind:  gatewayapi_v1beta1.Kind("HTTPRoute"),
								},
							},
						},
					}},
				},
			},
			objs: []interface{}{
				kuardService,
				basicHTTPRoute,
			},
			want: listeners(),
		},
		"insert basic multiple routes, single hostname": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				blogService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
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
						}, {
							Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/blog"),
							BackendRefs: gatewayapi.HTTPBackendRef("blogsvc", 80, 1),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("test.projectcontour.io",
							prefixrouteHTTPRoute("/", service(kuardService)), segmentPrefixHTTPRoute("/blog", service(blogService))),
					),
				},
			),
		},
		"multiple hosts": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{
						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1beta1.Hostname{
							"test.projectcontour.io",
							"test2.projectcontour.io",
							"test3.projectcontour.io",
							"test4.projectcontour.io",
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("test.projectcontour.io", prefixrouteHTTPRoute("/", service(kuardService))),
						virtualhost("test2.projectcontour.io", prefixrouteHTTPRoute("/", service(kuardService))),
						virtualhost("test3.projectcontour.io", prefixrouteHTTPRoute("/", service(kuardService))),
						virtualhost("test4.projectcontour.io", prefixrouteHTTPRoute("/", service(kuardService))),
					),
				},
			),
		},
		"no host defined": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{
						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*", prefixrouteHTTPRoute("/", service(kuardService))),
					),
				},
			),
		},
		"wildcard hostname": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{
						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1beta1.Hostname{
							"*.projectcontour.io",
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(virtualhost("*.projectcontour.io",
						&Route{
							PathMatchCondition: prefixString("/"),
							Clusters:           clustersWeight(service(kuardService)),
						}),
					),
				},
			),
		},
		"invalid hostnames - IP": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
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
							"192.168.122.1",
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						}},
					},
				},
			},
			want: listeners(),
		},
		"invalid hostnames - with port": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
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
							"test.projectcontour.io:80",
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						}},
					},
				},
			},
			want: listeners(),
		},
		"invalid hostnames - wildcard label by itself": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
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
							"*",
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						}},
					},
				},
			},
			want: listeners(),
		},
		// If the ServiceName referenced from an HTTPRoute is missing,
		// the route should return an HTTP 500.
		"missing service": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{
						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*", directResponseRoute("/", http.StatusInternalServerError)),
					),
				},
			),
		},
		// If port is not defined the route will return an HTTP 500.
		"missing port": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "default",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{
						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: []gatewayapi_v1beta1.HTTPBackendRef{{
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
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*", directResponseRoute("/", http.StatusInternalServerError)),
					),
				},
			),
		},
		"HTTPRoute references a backend in a different namespace, no ReferenceGrant": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
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
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: []gatewayapi_v1beta1.HTTPBackendRef{{
								BackendRef: gatewayapi_v1beta1.BackendRef{
									BackendObjectReference: gatewayapi_v1beta1.BackendObjectReference{
										Kind:      gatewayapi.KindPtr("Service"),
										Namespace: gatewayapi.NamespacePtr(kuardService.Namespace),
										Name:      gatewayapi_v1beta1.ObjectName(kuardService.Name),
										Port:      gatewayapi.PortNumPtr(8080),
									},
									Weight: pointer.Int32(1),
								},
							}},
						}},
					},
				},
			},
			want: listeners(&Listener{
				Name: HTTP_LISTENER_NAME,
				Port: 80,
				VirtualHosts: virtualhosts(
					virtualhost("*", directResponseRoute("/", http.StatusInternalServerError)),
				),
			}),
		},
		"HTTPRoute references a backend in a different namespace, with valid ReferenceGrant": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
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
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: []gatewayapi_v1beta1.HTTPBackendRef{{
								BackendRef: gatewayapi_v1beta1.BackendRef{
									BackendObjectReference: gatewayapi_v1beta1.BackendObjectReference{
										Kind:      gatewayapi.KindPtr("Service"),
										Namespace: gatewayapi.NamespacePtr(kuardService.Namespace),
										Name:      gatewayapi_v1beta1.ObjectName(kuardService.Name),
										Port:      gatewayapi.PortNumPtr(8080),
									},
									Weight: pointer.Int32(1),
								},
							}},
						}},
					},
				},
				&gatewayapi_v1alpha2.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: kuardService.Namespace,
					},
					Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
						From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
							Group:     gatewayapi_v1alpha2.GroupName,
							Kind:      "HTTPRoute",
							Namespace: "default",
						}},
						To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
							Kind: "Service",
						}},
					},
				},
			},
			want: listeners(&Listener{
				Name:         HTTP_LISTENER_NAME,
				Port:         80,
				VirtualHosts: virtualhosts(virtualhost("*", prefixrouteHTTPRoute("/", service(kuardService)))),
			}),
		},
		"HTTPRoute references a backend in a different namespace, with valid ReferenceGrant (service-specific)": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
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
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: []gatewayapi_v1beta1.HTTPBackendRef{{
								BackendRef: gatewayapi_v1beta1.BackendRef{
									BackendObjectReference: gatewayapi_v1beta1.BackendObjectReference{
										Kind:      gatewayapi.KindPtr("Service"),
										Namespace: gatewayapi.NamespacePtr(kuardService.Namespace),
										Name:      gatewayapi_v1beta1.ObjectName(kuardService.Name),
										Port:      gatewayapi.PortNumPtr(8080),
									},
									Weight: pointer.Int32(1),
								},
							}},
						}},
					},
				},
				&gatewayapi_v1alpha2.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: kuardService.Namespace,
					},
					Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
						From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
							Group:     gatewayapi_v1alpha2.GroupName,
							Kind:      "HTTPRoute",
							Namespace: "default",
						}},
						To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
							Kind: "Service",
							Name: gatewayapi.ObjectNamePtr(kuardService.Name),
						}},
					},
				},
			},
			want: listeners(&Listener{
				Name:         HTTP_LISTENER_NAME,
				Port:         80,
				VirtualHosts: virtualhosts(virtualhost("*", prefixrouteHTTPRoute("/", service(kuardService)))),
			}),
		},
		"HTTPRoute references a backend in a different namespace, with invalid ReferenceGrant (wrong Kind)": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
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
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: []gatewayapi_v1beta1.HTTPBackendRef{{
								BackendRef: gatewayapi_v1beta1.BackendRef{
									BackendObjectReference: gatewayapi_v1beta1.BackendObjectReference{
										Kind:      gatewayapi.KindPtr("Service"),
										Namespace: gatewayapi.NamespacePtr(kuardService.Namespace),
										Name:      gatewayapi_v1beta1.ObjectName(kuardService.Name),
										Port:      gatewayapi.PortNumPtr(8080),
									},
									Weight: pointer.Int32(1),
								},
							}},
						}},
					},
				},
				&gatewayapi_v1alpha2.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: kuardService.Namespace,
					},
					Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
						From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
							Group:     gatewayapi_v1alpha2.GroupName,
							Kind:      "TLSRoute",
							Namespace: "default",
						}},
						To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
							Kind: "Service",
						}},
					},
				},
			},
			want: listeners(&Listener{
				Name: HTTP_LISTENER_NAME,
				Port: 80,
				VirtualHosts: virtualhosts(
					virtualhost("*", directResponseRoute("/", http.StatusInternalServerError)),
				),
			}),
		},
		"HTTPRoute references a backend in a different namespace, with invalid ReferenceGrant (grant in wrong namespace)": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
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
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: []gatewayapi_v1beta1.HTTPBackendRef{{
								BackendRef: gatewayapi_v1beta1.BackendRef{
									BackendObjectReference: gatewayapi_v1beta1.BackendObjectReference{
										Kind:      gatewayapi.KindPtr("Service"),
										Namespace: gatewayapi.NamespacePtr(kuardService.Namespace),
										Name:      gatewayapi_v1beta1.ObjectName(kuardService.Name),
										Port:      gatewayapi.PortNumPtr(8080),
									},
									Weight: pointer.Int32(1),
								},
							}},
						}},
					},
				},
				&gatewayapi_v1alpha2.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "some-other-namespace", // would need to be "projectcontour" to be valid
					},
					Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
						From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
							Group:     gatewayapi_v1alpha2.GroupName,
							Kind:      "HTTPRoute",
							Namespace: "default",
						}},
						To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
							Kind: "Service",
						}},
					},
				},
			},
			want: listeners(&Listener{
				Name: HTTP_LISTENER_NAME,
				Port: 80,
				VirtualHosts: virtualhosts(
					virtualhost("*", directResponseRoute("/", http.StatusInternalServerError)),
				),
			}),
		},
		"HTTPRoute references a backend in a different namespace, with invalid ReferenceGrant (wrong from namespace)": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
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
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: []gatewayapi_v1beta1.HTTPBackendRef{{
								BackendRef: gatewayapi_v1beta1.BackendRef{
									BackendObjectReference: gatewayapi_v1beta1.BackendObjectReference{
										Kind:      gatewayapi.KindPtr("Service"),
										Namespace: gatewayapi.NamespacePtr(kuardService.Namespace),
										Name:      gatewayapi_v1beta1.ObjectName(kuardService.Name),
										Port:      gatewayapi.PortNumPtr(8080),
									},
									Weight: pointer.Int32(1),
								},
							}},
						}},
					},
				},
				&gatewayapi_v1alpha2.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: kuardService.Namespace,
					},
					Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
						From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
							Group:     gatewayapi_v1alpha2.GroupName,
							Kind:      "HTTPRoute",
							Namespace: "some-other-namespace", // would need to be "default" to be valid
						}},
						To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
							Kind: "Service",
						}},
					},
				},
			},
			want: listeners(&Listener{
				Name: HTTP_LISTENER_NAME,
				Port: 80,
				VirtualHosts: virtualhosts(
					virtualhost("*", directResponseRoute("/", http.StatusInternalServerError)),
				),
			}),
		},
		"HTTPRoute references a backend in a different namespace, with invalid ReferenceGrant (wrong service name)": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
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
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: []gatewayapi_v1beta1.HTTPBackendRef{{
								BackendRef: gatewayapi_v1beta1.BackendRef{
									BackendObjectReference: gatewayapi_v1beta1.BackendObjectReference{
										Kind:      gatewayapi.KindPtr("Service"),
										Namespace: gatewayapi.NamespacePtr(kuardService.Namespace),
										Name:      gatewayapi_v1beta1.ObjectName(kuardService.Name),
										Port:      gatewayapi.PortNumPtr(8080),
									},
									Weight: pointer.Int32(1),
								},
							}},
						}},
					},
				},
				&gatewayapi_v1alpha2.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: kuardService.Namespace,
					},
					Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
						From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
							Group:     gatewayapi_v1alpha2.GroupName,
							Kind:      "HTTPRoute",
							Namespace: "default",
						}},
						To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
							Kind: "Service",
							Name: gatewayapi.ObjectNamePtr("some-other-service"), // would need to be "kuard" to be valid.
						}},
					},
				},
			},
			want: listeners(&Listener{
				Name: HTTP_LISTENER_NAME,
				Port: 80,
				VirtualHosts: virtualhosts(
					virtualhost("*", directResponseRoute("/", http.StatusInternalServerError)),
				),
			}),
		},
		"insert basic single route with exact path match": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{
						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1beta1.Hostname{
							"test.projectcontour.io",
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchExact, "/blog"),
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("test.projectcontour.io",
							exactrouteHTTPRoute("/blog", service(kuardService))),
					),
				},
			),
		},
		// Single host with single route containing multiple prefixes to the same service.
		"insert basic single route with multiple prefixes, single hostname": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
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
							}, {
								Path: &gatewayapi_v1beta1.HTTPPathMatch{
									Type:  gatewayapi.PathMatchTypePtr(gatewayapi_v1beta1.PathMatchPathPrefix),
									Value: pointer.StringPtr("/blog"),
								},
							}, {
								Path: &gatewayapi_v1beta1.HTTPPathMatch{
									Type:  gatewayapi.PathMatchTypePtr(gatewayapi_v1beta1.PathMatchPathPrefix),
									Value: pointer.StringPtr("/tech"),
								},
							}},
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("test.projectcontour.io",
							prefixrouteHTTPRoute("/", service(kuardService)),
							segmentPrefixHTTPRoute("/blog", service(kuardService)),
							segmentPrefixHTTPRoute("/tech", service(kuardService))),
					),
				},
			),
		},
		"insert basic single route, single hostname, gateway with TLS, HTTP protocol is ignored": {
			gatewayclass: validClass,
			gateway: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "contour",
					Namespace: "projectcontour",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					Listeners: []gatewayapi_v1beta1.Listener{{
						Port:     443,
						Protocol: gatewayapi_v1beta1.HTTPProtocolType,
						TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
							CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
								gatewayapi.CertificateRef(sec1.Name, sec1.Namespace),
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
				sec1,
				kuardService,
				basicHTTPRoute,
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("test.projectcontour.io",
							prefixrouteHTTPRoute("/", service(kuardService)),
						)),
				},
			),
		},
		"insert basic single route, single hostname, gateway with TLS, HTTPS protocol missing certificateRef": {
			gatewayclass: validClass,
			gateway: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "contour",
					Namespace: "projectcontour",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					Listeners: []gatewayapi_v1beta1.Listener{{
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
			objs: []interface{}{
				sec1,
				kuardService,
				basicHTTPRoute,
			},
			want: listeners(),
		},
		"insert basic single route, single hostname, gateway with TLS": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPSAllNamespaces,
			objs: []interface{}{
				sec1,
				kuardService,
				basicHTTPRoute,
			},
			want: listeners(
				&Listener{
					Name: HTTPS_LISTENER_NAME,
					Port: 443,
					SecureVirtualHosts: securevirtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name:   "test.projectcontour.io",
								Routes: routes(prefixrouteHTTPRoute("/", service(kuardService))),
							},
							Secret: secret(sec1),
						},
					),
				},
			),
		},
		"insert basic single route, single hostname, gateway with missing TLS certificate": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPSAllNamespaces,
			objs: []interface{}{
				kuardService,
				basicHTTPRoute,
			},
			want: listeners(),
		},
		"insert basic single route, single hostname, gateway with invalid TLS certificate": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPSAllNamespaces,
			objs: []interface{}{
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tlscert",
						Namespace: "projectcontour",
					},
					Type: v1.SecretTypeTLS,
					Data: secretdata("wrong", "wronger"),
				},
				kuardService,
				basicHTTPRoute,
			},
			want: listeners(),
		},
		"insert basic single route, single hostname, gateway with TLS & Insecure Listeners": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAndHTTPS,
			objs: []interface{}{
				sec1,
				kuardService,
				basicHTTPRoute,
			},
			want: listeners(
				&Listener{
					Name: HTTPS_LISTENER_NAME,
					Port: 443,
					SecureVirtualHosts: securevirtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name:   "test.projectcontour.io",
								Routes: routes(prefixrouteHTTPRoute("/", service(kuardService))),
							},
							Secret: secret(sec1),
						},
					),
				},
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("test.projectcontour.io", prefixrouteHTTPRoute("/", service(kuardService))),
					),
				},
			),
		},
		"TLS Listener Gateway CertificateRef must be type core.Secret": {
			gatewayclass: validClass,
			gateway: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "contour",
					Namespace: "projectcontour",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					Listeners: []gatewayapi_v1beta1.Listener{{
						Port:     443,
						Protocol: gatewayapi_v1beta1.HTTPSProtocolType,
						TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
							CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
								{
									Group: gatewayapi.GroupPtr("custom"),
									Kind:  gatewayapi.KindPtr("shhhh"),
									Name:  gatewayapi_v1beta1.ObjectName(sec1.Name),
								},
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
				sec1,
				blogService,
				basicHTTPRoute,
			},
			want: listeners(),
		},
		"TLS Listener Gateway CertificateRef must be specified": {
			gatewayclass: validClass,
			gateway: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "contour",
					Namespace: "projectcontour",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					Listeners: []gatewayapi_v1beta1.Listener{{
						Port:     443,
						Protocol: gatewayapi_v1beta1.HTTPSProtocolType,
						TLS:      &gatewayapi_v1beta1.GatewayTLSConfig{},
						AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
							Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
								From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
							},
						},
					}},
				},
			},
			objs: []interface{}{
				sec1,
				blogService,
				basicHTTPRoute,
			},
			want: listeners(),
		},

		// BEGIN TLS CertificateRef + ReferenceGrant tests
		"Gateway references TLS cert in different namespace, with valid ReferenceGrant": {
			gatewayclass: validClass,
			gateway:      gatewayTLSTerminateCertInDifferentNamespace,
			objs: []interface{}{
				sec2,
				&gatewayapi_v1alpha2.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tls-cert-reference-grant",
						Namespace: sec2.Namespace,
					},
					Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
						From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
							Group:     gatewayapi_v1alpha2.GroupName,
							Kind:      "Gateway",
							Namespace: gatewayapi_v1alpha2.Namespace(gatewayTLSTerminateCertInDifferentNamespace.Namespace),
						}},
						To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
							Kind: "Secret",
						}},
					},
				},
				basicHTTPRoute,
				kuardService,
			},
			want: listeners(
				&Listener{
					Name: HTTPS_LISTENER_NAME,
					Port: 443,
					SecureVirtualHosts: securevirtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name:   "test.projectcontour.io",
								Routes: routes(prefixrouteHTTPRoute("/", service(kuardService))),
							},
							Secret: secret(sec2),
						},
					),
				},
			),
		},
		"Gateway references TLS cert in different namespace, with no ReferenceGrant": {
			gatewayclass: validClass,
			gateway:      gatewayTLSTerminateCertInDifferentNamespace,
			objs: []interface{}{
				sec2,
				basicHTTPRoute,
				kuardService,
			},
			want: listeners(),
		},
		"Gateway references TLS cert in different namespace, with valid ReferenceGrant (secret-specific)": {
			gatewayclass: validClass,
			gateway:      gatewayTLSTerminateCertInDifferentNamespace,
			objs: []interface{}{
				sec2,
				&gatewayapi_v1alpha2.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tls-cert-reference-grant",
						Namespace: sec2.Namespace,
					},
					Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
						From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
							Group:     gatewayapi_v1alpha2.GroupName,
							Kind:      "Gateway",
							Namespace: gatewayapi_v1alpha2.Namespace(gatewayTLSTerminateCertInDifferentNamespace.Namespace),
						}},
						To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
							Kind: "Secret",
							Name: gatewayapi.ObjectNamePtr(sec2.Name),
						}},
					},
				},
				basicHTTPRoute,
				kuardService,
			},
			want: listeners(
				&Listener{
					Name: HTTPS_LISTENER_NAME,
					Port: 443,
					SecureVirtualHosts: securevirtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name:   "test.projectcontour.io",
								Routes: routes(prefixrouteHTTPRoute("/", service(kuardService))),
							},
							Secret: secret(sec2),
						},
					),
				},
			),
		},
		"Gateway references TLS cert in different namespace, with invalid ReferenceGrant (grant in wrong namespace)": {
			gatewayclass: validClass,
			gateway:      gatewayTLSTerminateCertInDifferentNamespace,
			objs: []interface{}{
				sec2,
				&gatewayapi_v1alpha2.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tls-cert-reference-grant",
						Namespace: "wrong-namespace",
					},
					Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
						From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
							Group:     gatewayapi_v1alpha2.GroupName,
							Kind:      "Gateway",
							Namespace: gatewayapi_v1alpha2.Namespace(gatewayTLSTerminateCertInDifferentNamespace.Namespace),
						}},
						To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
							Kind: "Secret",
						}},
					},
				},
				basicHTTPRoute,
				kuardService,
			},
			want: listeners(),
		},
		"Gateway references TLS cert in different namespace, with invalid ReferenceGrant (wrong From namespace)": {
			gatewayclass: validClass,
			gateway:      gatewayTLSTerminateCertInDifferentNamespace,
			objs: []interface{}{
				sec2,
				&gatewayapi_v1alpha2.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tls-cert-reference-grant",
						Namespace: sec2.Namespace,
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
				basicHTTPRoute,
				kuardService,
			},
			want: listeners(),
		},
		"Gateway references TLS cert in different namespace, with invalid ReferenceGrant (wrong From kind)": {
			gatewayclass: validClass,
			gateway:      gatewayTLSTerminateCertInDifferentNamespace,
			objs: []interface{}{
				sec2,
				&gatewayapi_v1alpha2.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tls-cert-reference-grant",
						Namespace: sec2.Namespace,
					},
					Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
						From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
							Group:     gatewayapi_v1alpha2.GroupName,
							Kind:      "WrongKind",
							Namespace: gatewayapi_v1alpha2.Namespace(gatewayTLSTerminateCertInDifferentNamespace.Namespace),
						}},
						To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
							Kind: "Secret",
						}},
					},
				},
				basicHTTPRoute,
				kuardService,
			},
			want: listeners(),
		},
		"Gateway references TLS cert in different namespace, with invalid ReferenceGrant (wrong To kind)": {
			gatewayclass: validClass,
			gateway:      gatewayTLSTerminateCertInDifferentNamespace,
			objs: []interface{}{
				sec2,
				&gatewayapi_v1alpha2.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tls-cert-reference-grant",
						Namespace: sec2.Namespace,
					},
					Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
						From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
							Group:     gatewayapi_v1alpha2.GroupName,
							Kind:      "Gateway",
							Namespace: gatewayapi_v1alpha2.Namespace(gatewayTLSTerminateCertInDifferentNamespace.Namespace),
						}},
						To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
							Kind: "WrongKind",
						}},
					},
				},
				basicHTTPRoute,
				kuardService,
			},
			want: listeners(),
		},
		"Gateway references TLS cert in different namespace, with invalid ReferenceGrant (wrong secret name)": {
			gatewayclass: validClass,
			gateway:      gatewayTLSTerminateCertInDifferentNamespace,
			objs: []interface{}{
				sec2,
				&gatewayapi_v1alpha2.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tls-cert-reference-grant",
						Namespace: sec2.Namespace,
					},
					Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
						From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
							Group:     gatewayapi_v1alpha2.GroupName,
							Kind:      "Gateway",
							Namespace: gatewayapi_v1alpha2.Namespace(gatewayTLSTerminateCertInDifferentNamespace.Namespace),
						}},
						To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
							Kind: "Secret",
							Name: gatewayapi.ObjectNamePtr("wrong-name"),
						}},
					},
				},
				basicHTTPRoute,
				kuardService,
			},
			want: listeners(),
		},

		// END CertificateRef ReferenceGrant tests

		"No valid hostnames defined": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{
						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1beta1.Hostname{
							"*.*.projectcontour.io",
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: gatewayapi.HTTPBackendRef("blogsvc", 80, 1),
						}},
					},
				},
			},
			want: listeners(),
		},
		"Invalid listener protocol type (TCP)": {
			gatewayclass: validClass,
			gateway: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "contour",
					Namespace: "projectcontour",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					Listeners: []gatewayapi_v1beta1.Listener{{
						Port:     80,
						Protocol: gatewayapi_v1beta1.TCPProtocolType,
						AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
							Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
								From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
							},
						},
					}},
				},
			},
			objs: []interface{}{basicHTTPRoute},
			want: listeners(),
		},
		"Invalid listener protocol type (UDP)": {
			gatewayclass: validClass,
			gateway: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "contour",
					Namespace: "projectcontour",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					Listeners: []gatewayapi_v1beta1.Listener{{
						Port:     80,
						Protocol: gatewayapi_v1beta1.UDPProtocolType,
						AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
							Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
								From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
							},
						},
					}},
				},
			},
			objs: []interface{}{basicHTTPRoute},
			want: listeners(),
		},
		"Invalid listener protocol type (custom)": {
			gatewayclass: validClass,
			gateway: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "contour",
					Namespace: "projectcontour",
				},
				Spec: gatewayapi_v1beta1.GatewaySpec{
					Listeners: []gatewayapi_v1beta1.Listener{{
						Port:     80,
						Protocol: "projectcontour.io/HTTPUDP",
						AllowedRoutes: &gatewayapi_v1beta1.AllowedRoutes{
							Namespaces: &gatewayapi_v1beta1.RouteNamespaces{
								From: gatewayapi.FromNamespacesPtr(gatewayapi_v1beta1.NamespacesFromAll),
							},
						},
					}},
				},
			},
			objs: []interface{}{basicHTTPRoute},
			want: listeners(),
		},
		"gateway with HTTP and HTTPS listeners, each route selects a different listener": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAndHTTPS,
			objs: []interface{}{
				sec1,
				kuardService,
				blogService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{
						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{
								gatewayapi.GatewayListenerParentRef("projectcontour", "contour", "http-listener"),
							},
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
						Name:      "basictls",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{

						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{
								gatewayapi.GatewayListenerParentRef("projectcontour", "contour", "https-listener"),
							},
						},
						Hostnames: []gatewayapi_v1beta1.Hostname{
							"test.projectcontour.io",
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: gatewayapi.HTTPBackendRef("blogsvc", 80, 1),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTPS_LISTENER_NAME,
					Port: 443,
					SecureVirtualHosts: securevirtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name:   "test.projectcontour.io",
								Routes: routes(prefixrouteHTTPRoute("/", service(blogService))),
							},
							Secret: secret(sec1),
						},
					),
				},
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("test.projectcontour.io", prefixrouteHTTPRoute("/", service(kuardService))),
					),
				},
			),
		},
		"insert basic single route with single header match and path match": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
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
								Headers: gatewayapi.HTTPHeaderMatch(gatewayapi_v1beta1.HeaderMatchExact, "foo", "bar"),
							}},
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(virtualhost("test.projectcontour.io",
						&Route{
							PathMatchCondition: prefixString("/"),
							HeaderMatchConditions: []HeaderMatchCondition{
								{Name: "foo", Value: "bar", MatchType: "exact", Invert: false},
							},
							Clusters: clustersWeight(service(kuardService)),
						}),
					),
				},
			),
		},
		"insert two routes with single header match, path match and header match": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{
						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1beta1.Hostname{
							"test.projectcontour.io",
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{
							{
								Matches: []gatewayapi_v1beta1.HTTPRouteMatch{
									{
										Path: &gatewayapi_v1beta1.HTTPPathMatch{
											Type:  gatewayapi.PathMatchTypePtr(gatewayapi_v1beta1.PathMatchPathPrefix),
											Value: pointer.StringPtr("/blog"),
										},
									}, {
										Path: &gatewayapi_v1beta1.HTTPPathMatch{
											Type:  gatewayapi.PathMatchTypePtr(gatewayapi_v1beta1.PathMatchPathPrefix),
											Value: pointer.StringPtr("/tech"),
										},
										Headers: gatewayapi.HTTPHeaderMatch(gatewayapi_v1beta1.HeaderMatchExact, "foo", "bar"),
									},
								},
								BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
							}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(virtualhost("test.projectcontour.io",
						&Route{
							PathMatchCondition: prefixSegment("/blog"),
							Clusters:           clustersWeight(service(kuardService)),
						},
						&Route{
							PathMatchCondition: prefixSegment("/tech"),
							HeaderMatchConditions: []HeaderMatchCondition{
								{Name: "foo", Value: "bar", MatchType: "exact", Invert: false},
							},
							Clusters: clustersWeight(service(kuardService)),
						},
					)),
				},
			),
		},
		"insert two routes with single header match without explicit path match": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
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
								Headers: gatewayapi.HTTPHeaderMatch(gatewayapi_v1beta1.HeaderMatchExact, "foo", "bar"),
							}},
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(virtualhost("test.projectcontour.io",
						&Route{
							PathMatchCondition: prefixString("/"),
							HeaderMatchConditions: []HeaderMatchCondition{
								{Name: "foo", Value: "bar", MatchType: "exact", Invert: false},
							},
							Clusters: clustersWeight(service(kuardService)),
						},
					)),
				},
			),
		},
		"insert route with multiple header matches including multiple for the same key": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
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
								Headers: []gatewayapi_v1beta1.HTTPHeaderMatch{
									{Name: "header-1", Value: "value-1"},
									{Name: "header-2", Value: "value-2"},
									{Name: "header-1", Value: "value-3"},
									{Name: "HEADER-1", Value: "value-4"},
								},
							}},
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(virtualhost("test.projectcontour.io",
						&Route{
							PathMatchCondition: prefixString("/"),
							HeaderMatchConditions: []HeaderMatchCondition{
								{Name: "header-1", Value: "value-1", MatchType: "exact"},
								{Name: "header-2", Value: "value-2", MatchType: "exact"},
							},
							Clusters: clustersWeight(service(kuardService)),
						},
					)),
				},
			),
		},
		"route with HTTP method match": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
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
								Method: gatewayapi.HTTPMethodPtr(gatewayapi_v1beta1.HTTPMethodGet),
							}},
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(virtualhost("test.projectcontour.io",
						&Route{
							PathMatchCondition: prefixString("/"),
							HeaderMatchConditions: []HeaderMatchCondition{
								{Name: ":method", Value: "GET", MatchType: "exact"},
							},
							Clusters: clustersWeight(service(kuardService)),
						}),
					),
				},
			),
		},
		"insert single route with single query param match without type specified and path match": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
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
										Name:  "param-1",
										Value: "value-1",
									},
								},
							}},
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(virtualhost("test.projectcontour.io",
						&Route{
							PathMatchCondition: prefixString("/"),
							QueryParamMatchConditions: []QueryParamMatchCondition{
								{Name: "param-1", Value: "value-1", MatchType: QueryParamMatchTypeExact},
							},
							Clusters: clustersWeight(service(kuardService)),
						}),
					),
				},
			),
		},
		"insert single route with single query param match with type specified and path match": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
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
										Type:  gatewayapi.QueryParamMatchTypePtr(gatewayapi_v1beta1.QueryParamMatchExact),
										Name:  "param-1",
										Value: "value-1",
									},
								},
							}},
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(virtualhost("test.projectcontour.io",
						&Route{
							PathMatchCondition: prefixString("/"),
							QueryParamMatchConditions: []QueryParamMatchCondition{
								{Name: "param-1", Value: "value-1", MatchType: QueryParamMatchTypeExact},
							},
							Clusters: clustersWeight(service(kuardService)),
						}),
					),
				},
			),
		},
		"insert single route with multiple query param matches including multiple for the same key": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
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
										Type:  gatewayapi.QueryParamMatchTypePtr(gatewayapi_v1beta1.QueryParamMatchExact),
										Name:  "param-1",
										Value: "value-1",
									},
									{
										Type:  gatewayapi.QueryParamMatchTypePtr(gatewayapi_v1beta1.QueryParamMatchExact),
										Name:  "param-2",
										Value: "value-2",
									},
									{
										Type:  gatewayapi.QueryParamMatchTypePtr(gatewayapi_v1beta1.QueryParamMatchExact),
										Name:  "param-1",
										Value: "value-3",
									},
									{
										Type:  gatewayapi.QueryParamMatchTypePtr(gatewayapi_v1beta1.QueryParamMatchExact),
										Name:  "Param-1",
										Value: "value-4",
									},
								},
							}},
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(virtualhost("test.projectcontour.io",
						&Route{
							PathMatchCondition: prefixString("/"),
							QueryParamMatchConditions: []QueryParamMatchCondition{
								{Name: "param-1", Value: "value-1", MatchType: QueryParamMatchTypeExact},
								{Name: "param-2", Value: "value-2", MatchType: QueryParamMatchTypeExact},
								{Name: "Param-1", Value: "value-4", MatchType: QueryParamMatchTypeExact},
							},
							Clusters: clustersWeight(service(kuardService)),
						}),
					),
				},
			),
		},
		"Route rule with request header modifier": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
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
										{Name: gatewayapi_v1beta1.HTTPHeaderName("custom-header-set"), Value: "foo-bar"},
										{Name: gatewayapi_v1beta1.HTTPHeaderName("Host"), Value: "bar.com"},
									},
									Add: []gatewayapi_v1beta1.HTTPHeader{
										{Name: "custom-header-add", Value: "foo-bar"},
									},
								},
							}},
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(virtualhost("test.projectcontour.io",
						&Route{
							PathMatchCondition: prefixString("/"),
							Clusters:           clustersWeight(service(kuardService)),
							RequestHeadersPolicy: &HeadersPolicy{
								Set: map[string]string{
									"Custom-Header-Set": "foo-bar", // Verify the header key is canonicalized.
								},
								Add: map[string]string{
									"Custom-Header-Add": "foo-bar", // Verify the header key is canonicalized.
								},
								HostRewrite: "bar.com",
							},
						},
					)),
				},
			),
		},
		"HTTP forward with request header modifier": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
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
										Weight:                 pointer.Int32(1),
									},
									Filters: []gatewayapi_v1beta1.HTTPRouteFilter{{
										Type: gatewayapi_v1beta1.HTTPRouteFilterRequestHeaderModifier,
										RequestHeaderModifier: &gatewayapi_v1beta1.HTTPRequestHeaderFilter{
											Set: []gatewayapi_v1beta1.HTTPHeader{
												{Name: gatewayapi_v1beta1.HTTPHeaderName("custom-header-set"), Value: "foo-bar"},
												{Name: gatewayapi_v1beta1.HTTPHeaderName("Host"), Value: "bar.com"},
											},
											Add: []gatewayapi_v1beta1.HTTPHeader{
												{Name: "custom-header-add", Value: "foo-bar"},
											},
										},
									}},
								},
							},
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(virtualhost("test.projectcontour.io",
						&Route{
							PathMatchCondition: prefixString("/"),
							Clusters:           clusterHeaders(map[string]string{"Custom-Header-Set": "foo-bar"}, map[string]string{"Custom-Header-Add": "foo-bar"}, nil, "bar.com", service(kuardService)),
						},
					)),
				},
			),
		},
		"Route rule with invalid request header modifier": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{
						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1beta1.Hostname{
							"test.projectcontour.io",
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{
							{
								Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
								BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
								Filters: []gatewayapi_v1beta1.HTTPRouteFilter{{
									Type: gatewayapi_v1beta1.HTTPRouteFilterRequestHeaderModifier,
									RequestHeaderModifier: &gatewayapi_v1beta1.HTTPRequestHeaderFilter{
										Set: []gatewayapi_v1beta1.HTTPHeader{
											{Name: gatewayapi_v1beta1.HTTPHeaderName("custom-header-set"), Value: "foo-bar"},
											{Name: gatewayapi_v1beta1.HTTPHeaderName("Host"), Value: "bar.com"},
										},
										Add: []gatewayapi_v1beta1.HTTPHeader{
											{Name: "!invalid-header-add", Value: "foo-bar"},
										},
									},
								}},
							}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(virtualhost("test.projectcontour.io",
						&Route{
							PathMatchCondition: prefixString("/"),
							Clusters:           clustersWeight(service(kuardService)),
							RequestHeadersPolicy: &HeadersPolicy{
								Set:         map[string]string{"Custom-Header-Set": "foo-bar"},
								Add:         map[string]string{}, // Invalid header should not be set.
								HostRewrite: "bar.com",
							},
						},
					)),
				},
			),
		},
		"HTTP forward with invalid request header modifier": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{
						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1beta1.Hostname{
							"test.projectcontour.io",
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{
							{
								Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
								BackendRefs: []gatewayapi_v1beta1.HTTPBackendRef{
									{
										BackendRef: gatewayapi_v1beta1.BackendRef{
											BackendObjectReference: gatewayapi.ServiceBackendObjectRef("kuard", 8080),
											Weight:                 pointer.Int32(1),
										},
										Filters: []gatewayapi_v1beta1.HTTPRouteFilter{{
											Type: gatewayapi_v1beta1.HTTPRouteFilterRequestHeaderModifier,
											RequestHeaderModifier: &gatewayapi_v1beta1.HTTPRequestHeaderFilter{
												Set: []gatewayapi_v1beta1.HTTPHeader{
													{Name: gatewayapi_v1beta1.HTTPHeaderName("custom-header-set"), Value: "foo-bar"},
													{Name: gatewayapi_v1beta1.HTTPHeaderName("Host"), Value: "bar.com"},
												},
												Add: []gatewayapi_v1beta1.HTTPHeader{
													{Name: "!invalid-header-add", Value: "foo-bar"},
												},
											},
										}},
									},
								},
							}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(virtualhost("test.projectcontour.io",
						&Route{
							PathMatchCondition: prefixString("/"),
							Clusters:           clusterHeaders(map[string]string{"Custom-Header-Set": "foo-bar"}, map[string]string{}, nil, "bar.com", service(kuardService)),
						},
					)),
				},
			),
		},
		"HTTPRoute rule with request redirect filter": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
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
							Filters: []gatewayapi_v1beta1.HTTPRouteFilter{{
								Type: gatewayapi_v1beta1.HTTPRouteFilterRequestRedirect,
								RequestRedirect: &gatewayapi_v1beta1.HTTPRequestRedirectFilter{
									Scheme:     pointer.String("https"),
									Hostname:   gatewayapi.PreciseHostname("envoyproxy.io"),
									Port:       gatewayapi.PortNumPtr(443),
									StatusCode: pointer.Int(301),
								},
							}},
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(virtualhost("test.projectcontour.io",
						&Route{
							PathMatchCondition: prefixString("/"),
							Redirect: &Redirect{
								Scheme:     "https",
								Hostname:   "envoyproxy.io",
								PortNumber: 443,
								StatusCode: 301,
							},
						},
					)),
				},
			),
		},
		"HTTPRoute rule with request redirect filter with multiple matches": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{
						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1beta1.Hostname{
							"test.projectcontour.io",
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches: append(
								gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
								gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/another-match")...,
							),
							Filters: []gatewayapi_v1beta1.HTTPRouteFilter{{
								Type: gatewayapi_v1beta1.HTTPRouteFilterRequestRedirect,
								RequestRedirect: &gatewayapi_v1beta1.HTTPRequestRedirectFilter{
									Scheme:     pointer.String("https"),
									Hostname:   gatewayapi.PreciseHostname("envoyproxy.io"),
									Port:       gatewayapi.PortNumPtr(443),
									StatusCode: pointer.Int(301),
								},
							}},
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(virtualhost("test.projectcontour.io",
						&Route{
							PathMatchCondition: prefixString("/"),
							Redirect: &Redirect{
								Scheme:     "https",
								Hostname:   "envoyproxy.io",
								PortNumber: 443,
								StatusCode: 301,
							},
						},
						&Route{
							PathMatchCondition: prefixSegment("/another-match"),
							Redirect: &Redirect{
								Scheme:     "https",
								Hostname:   "envoyproxy.io",
								PortNumber: 443,
								StatusCode: 301,
							},
						},
					)),
				},
			),
		},
		"HTTPRoute rule with request mirror filter": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				kuardService2,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
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
							}},
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(virtualhost("test.projectcontour.io",
						withMirror(prefixrouteHTTPRoute("/", service(kuardService)), service(kuardService2)))),
				},
			),
		},
		"HTTPRoute rule with request mirror filter with multiple matches": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				kuardService2,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{
						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1beta1.Hostname{
							"test.projectcontour.io",
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches: append(
								gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
								gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/another-match")...,
							),
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
							Filters: []gatewayapi_v1beta1.HTTPRouteFilter{{
								Type: gatewayapi_v1beta1.HTTPRouteFilterRequestMirror,
								RequestMirror: &gatewayapi_v1beta1.HTTPRequestMirrorFilter{
									BackendRef: gatewayapi.ServiceBackendObjectRef("kuard2", 8080),
								},
							}},
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(virtualhost("test.projectcontour.io",
						withMirror(prefixrouteHTTPRoute("/", service(kuardService)), service(kuardService2)),
						withMirror(segmentPrefixHTTPRoute("/another-match", service(kuardService)), service(kuardService2)),
					)),
				},
			),
		},
		"different weights for multiple forwardTos": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				kuardService2,
				kuardService3,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{
						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: gatewayapi.HTTPBackendRefs(
								gatewayapi.HTTPBackendRef("kuard", 8080, 5),
								gatewayapi.HTTPBackendRef("kuard2", 8080, 10),
								gatewayapi.HTTPBackendRef("kuard3", 8080, 15),
							),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*", prefixrouteHTTPRoute("/",
							&Service{
								Weighted: WeightedService{
									Weight:           5,
									ServiceName:      kuardService.Name,
									ServiceNamespace: kuardService.Namespace,
									ServicePort:      kuardService.Spec.Ports[0],
								},
							},
							&Service{
								Weighted: WeightedService{
									Weight:           10,
									ServiceName:      kuardService2.Name,
									ServiceNamespace: kuardService2.Namespace,
									ServicePort:      kuardService2.Spec.Ports[0],
								},
							},
							&Service{
								Weighted: WeightedService{
									Weight:           15,
									ServiceName:      kuardService3.Name,
									ServiceNamespace: kuardService3.Namespace,
									ServicePort:      kuardService3.Spec.Ports[0],
								},
							},
						)),
					),
				},
			),
		},
		"one service weight zero w/weights for other forwardTos": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				kuardService2,
				kuardService3,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{
						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches: gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: gatewayapi.HTTPBackendRefs(
								gatewayapi.HTTPBackendRef("kuard", 8080, 5),
								gatewayapi.HTTPBackendRef("kuard2", 8080, 0),
								gatewayapi.HTTPBackendRef("kuard3", 8080, 15),
							),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*", prefixrouteHTTPRoute("/",
							&Service{
								Weighted: WeightedService{
									Weight:           5,
									ServiceName:      kuardService.Name,
									ServiceNamespace: kuardService.Namespace,
									ServicePort:      kuardService.Spec.Ports[0],
								},
							},
							&Service{
								Weighted: WeightedService{
									Weight:           0,
									ServiceName:      kuardService2.Name,
									ServiceNamespace: kuardService2.Namespace,
									ServicePort:      kuardService2.Spec.Ports[0],
								},
							},
							&Service{
								Weighted: WeightedService{
									Weight:           15,
									ServiceName:      kuardService3.Name,
									ServiceNamespace: kuardService3.Namespace,
									ServicePort:      kuardService3.Spec.Ports[0],
								},
							},
						)),
					),
				},
			),
		},
		"weight of zero for a single forwardTo results in 500": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				kuardService2,
				kuardService3,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{
						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchPathPrefix, "/"),
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 0),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*", directResponseRouteService("/", http.StatusInternalServerError, &Service{
							Weighted: WeightedService{
								Weight:           0,
								ServiceName:      kuardService.Name,
								ServiceNamespace: kuardService.Namespace,
								ServicePort:      kuardService.Spec.Ports[0],
							},
						})),
					),
				},
			),
		},
		"basic TLSRoute": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{"tcp.projectcontour.io"},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: gatewayapi.TLSRouteBackendRef("kuard", 8080, nil),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTPS_LISTENER_NAME,
					Port: 443,
					SecureVirtualHosts: securevirtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "tcp.projectcontour.io",
							},
							TCPProxy: &TCPProxy{
								Clusters: clustersWeight(service(kuardService)),
							},
						},
					),
				},
			),
		},
		"TLSRoute references a backend in a different namespace, no ReferenceGrant": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "default",
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{"tcp.projectcontour.io"},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: gatewayapi.TLSRouteBackendRef("kuard", 8080, nil),
						}},
					},
				},
			},
			want: listeners(),
		},
		"TLSRoute references a backend in a different namespace, with valid ReferenceGrant": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "default",
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{"tcp.projectcontour.io"},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: []gatewayapi_v1alpha2.BackendRef{
								{
									BackendObjectReference: gatewayapi_v1alpha2.BackendObjectReference{
										Kind:      gatewayapi.KindPtrV1Alpha2("Service"),
										Namespace: gatewayapi.NamespacePtrV1Alpha2(kuardService.Namespace),
										Name:      gatewayapi_v1alpha2.ObjectName(kuardService.Name),
										Port:      gatewayapi.PortNumPtrV1Alpha2(8080),
									},
									Weight: pointer.Int32(1),
								},
							},
						}},
					},
				},
				&gatewayapi_v1alpha2.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: kuardService.Namespace,
					},
					Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
						From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
							Group:     gatewayapi_v1alpha2.GroupName,
							Kind:      "TLSRoute",
							Namespace: "default",
						}},
						To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
							Kind: "Service",
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTPS_LISTENER_NAME,
					Port: 443,
					SecureVirtualHosts: securevirtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "tcp.projectcontour.io",
							},
							TCPProxy: &TCPProxy{
								Clusters: clustersWeight(service(kuardService)),
							},
						},
					),
				},
			),
		},
		"TLSRoute references a backend in a different namespace, with valid ReferenceGrant (service-specific)": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "default",
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{"tcp.projectcontour.io"},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: []gatewayapi_v1alpha2.BackendRef{
								{
									BackendObjectReference: gatewayapi_v1alpha2.BackendObjectReference{
										Kind:      gatewayapi.KindPtrV1Alpha2("Service"),
										Namespace: gatewayapi.NamespacePtrV1Alpha2(kuardService.Namespace),
										Name:      gatewayapi_v1alpha2.ObjectName(kuardService.Name),
										Port:      gatewayapi.PortNumPtrV1Alpha2(8080),
									},
									Weight: pointer.Int32(1),
								},
							},
						}},
					},
				},
				&gatewayapi_v1alpha2.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: kuardService.Namespace,
					},
					Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
						From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
							Group:     gatewayapi_v1alpha2.GroupName,
							Kind:      "TLSRoute",
							Namespace: "default",
						}},
						To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
							Kind: "Service",
							Name: gatewayapi.ObjectNamePtr(kuardService.Name),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTPS_LISTENER_NAME,
					Port: 443,
					SecureVirtualHosts: securevirtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "tcp.projectcontour.io",
							},
							TCPProxy: &TCPProxy{
								Clusters: clustersWeight(service(kuardService)),
							},
						},
					),
				},
			),
		},
		"TLSRoute references a backend in a different namespace, with invalid ReferenceGrant (wrong Kind)": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "default",
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{"tcp.projectcontour.io"},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: []gatewayapi_v1alpha2.BackendRef{
								{
									BackendObjectReference: gatewayapi_v1alpha2.BackendObjectReference{
										Kind:      gatewayapi.KindPtrV1Alpha2("Service"),
										Namespace: gatewayapi.NamespacePtrV1Alpha2(kuardService.Namespace),
										Name:      gatewayapi_v1alpha2.ObjectName(kuardService.Name),
										Port:      gatewayapi.PortNumPtrV1Alpha2(8080),
									},
									Weight: pointer.Int32(1),
								},
							},
						}},
					},
				},
				&gatewayapi_v1alpha2.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: kuardService.Namespace,
					},
					Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
						From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
							Group:     gatewayapi_v1alpha2.GroupName,
							Kind:      "HTTPRoute", // would need to be TLSRoute to be valid
							Namespace: "default",
						}},
						To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
							Kind: "Service",
						}},
					},
				},
			},
			want: listeners(),
		},
		"TLSRoute references a backend in a different namespace, with invalid ReferenceGrant (grant in wrong namespace)": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "default",
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{"tcp.projectcontour.io"},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: []gatewayapi_v1alpha2.BackendRef{
								{
									BackendObjectReference: gatewayapi_v1alpha2.BackendObjectReference{
										Kind:      gatewayapi.KindPtrV1Alpha2("Service"),
										Namespace: gatewayapi.NamespacePtrV1Alpha2(kuardService.Namespace),
										Name:      gatewayapi_v1alpha2.ObjectName(kuardService.Name),
										Port:      gatewayapi.PortNumPtrV1Alpha2(8080),
									},
									Weight: pointer.Int32(1),
								},
							},
						}},
					},
				},
				&gatewayapi_v1alpha2.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "some-other-namespace", // would have to be "projectcontour" to be valid
					},
					Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
						From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
							Group:     gatewayapi_v1alpha2.GroupName,
							Kind:      "TLSRoute",
							Namespace: "default",
						}},
						To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
							Kind: "Service",
						}},
					},
				},
			},
			want: listeners(),
		},
		"TLSRoute references a backend in a different namespace, with invalid ReferenceGrant (wrong from namespace)": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "default",
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{"tcp.projectcontour.io"},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: []gatewayapi_v1alpha2.BackendRef{
								{
									BackendObjectReference: gatewayapi_v1alpha2.BackendObjectReference{
										Kind:      gatewayapi.KindPtrV1Alpha2("Service"),
										Namespace: gatewayapi.NamespacePtrV1Alpha2(kuardService.Namespace),
										Name:      gatewayapi_v1alpha2.ObjectName(kuardService.Name),
										Port:      gatewayapi.PortNumPtrV1Alpha2(8080),
									},
									Weight: pointer.Int32(1),
								},
							},
						}},
					},
				},
				&gatewayapi_v1alpha2.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: kuardService.Namespace,
					},
					Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
						From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
							Group:     gatewayapi_v1alpha2.GroupName,
							Kind:      "TLSRoute",
							Namespace: "some-other-namespace", // would have to be "default" to be valid
						}},
						To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
							Kind: "Service",
						}},
					},
				},
			},
			want: listeners(),
		},
		"TLSRoute references a backend in a different namespace, with invalid ReferenceGrant (wrong service name)": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "default",
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{"tcp.projectcontour.io"},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: []gatewayapi_v1alpha2.BackendRef{
								{
									BackendObjectReference: gatewayapi_v1alpha2.BackendObjectReference{
										Kind:      gatewayapi.KindPtrV1Alpha2("Service"),
										Namespace: gatewayapi.NamespacePtrV1Alpha2(kuardService.Namespace),
										Name:      gatewayapi_v1alpha2.ObjectName(kuardService.Name),
										Port:      gatewayapi.PortNumPtrV1Alpha2(8080),
									},
									Weight: pointer.Int32(1),
								},
							},
						}},
					},
				},
				&gatewayapi_v1alpha2.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: kuardService.Namespace,
					},
					Spec: gatewayapi_v1alpha2.ReferenceGrantSpec{
						From: []gatewayapi_v1alpha2.ReferenceGrantFrom{{
							Group:     gatewayapi_v1alpha2.GroupName,
							Kind:      "TLSRoute",
							Namespace: "default",
						}},
						To: []gatewayapi_v1alpha2.ReferenceGrantTo{{
							Kind: "Service",
							Name: gatewayapi.ObjectNamePtr("some-other-service"), // would have to be "kuard" to be valid
						}},
					},
				},
			},
			want: listeners(),
		},
		"TLSRoute with multiple SNIs": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{
							"tcp.projectcontour.io",
							"another.projectcontour.io",
							"thing.projectcontour.io",
						},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: gatewayapi.TLSRouteBackendRef("kuard", 8080, nil),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTPS_LISTENER_NAME,
					Port: 443,
					SecureVirtualHosts: securevirtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "another.projectcontour.io",
							},
							TCPProxy: &TCPProxy{
								Clusters: clustersWeight(service(kuardService)),
							},
						},
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "tcp.projectcontour.io",
							},
							TCPProxy: &TCPProxy{
								Clusters: clustersWeight(service(kuardService)),
							},
						},
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "thing.projectcontour.io",
							},
							TCPProxy: &TCPProxy{
								Clusters: clustersWeight(service(kuardService)),
							},
						},
					),
				},
			),
		},
		"TLSRoute with multiple SNIs, one is invalid": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{
							"tcp.projectcontour.io",
							"*.*.another.projectcontour.io",
							"thing.projectcontour.io",
						},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: gatewayapi.TLSRouteBackendRef("kuard", 8080, nil),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTPS_LISTENER_NAME,
					Port: 443,
					SecureVirtualHosts: securevirtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "tcp.projectcontour.io",
							},
							TCPProxy: &TCPProxy{
								Clusters: clustersWeight(service(kuardService)),
							},
						},
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "thing.projectcontour.io",
							},
							TCPProxy: &TCPProxy{
								Clusters: clustersWeight(service(kuardService)),
							},
						},
					),
				},
			),
		},
		"TLSRoute with multiple SNIs, all are invalid": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{
							"tcp.*.projectcontour.io",
							"*.*.another.projectcontour.io",
							"!!thing.projectcontour.io",
						},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: gatewayapi.TLSRouteBackendRef("kuard", 8080, nil),
						}},
					},
				},
			},
			want: listeners(),
		},
		"TLSRoute without any hostnames specified results in '*' match all": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughAllNamespaces,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: gatewayapi.TLSRouteBackendRef("kuard", 8080, nil),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTPS_LISTENER_NAME,
					Port: 443,
					SecureVirtualHosts: securevirtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "*",
							},
							TCPProxy: &TCPProxy{
								Clusters: clustersWeight(service(kuardService)),
							},
						},
					),
				},
			),
		},
		"TLSRoute with missing forwardTo service": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughAllNamespaces,
			objs: []interface{}{
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{"tcp.projectcontour.io"},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: gatewayapi.TLSRouteBackendRef("kuard", 8080, nil),
						}},
					},
				},
			},
			want: listeners(),
		},
		"TLSRoute with multiple weighted ForwardTos": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughAllNamespaces,
			objs: []interface{}{
				kuardService,
				kuardService2,
				kuardService3,
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{"tcp.projectcontour.io"},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: gatewayapi.TLSRouteBackendRefs(
								gatewayapi.TLSRouteBackendRef("kuard", 8080, pointer.Int32Ptr(1)),
								gatewayapi.TLSRouteBackendRef("kuard2", 8080, pointer.Int32Ptr(2)),
								gatewayapi.TLSRouteBackendRef("kuard3", 8080, pointer.Int32Ptr(3)),
							),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTPS_LISTENER_NAME,
					Port: 443,
					SecureVirtualHosts: securevirtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "tcp.projectcontour.io",
							},
							TCPProxy: &TCPProxy{

								Clusters: clustersWeight(
									weightedService(kuardService, 1),
									weightedService(kuardService2, 2),
									weightedService(kuardService3, 3),
								),
							},
						},
					),
				},
			),
		},
		"TLSRoute with multiple weighted ForwardTos and one zero weight": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughAllNamespaces,
			objs: []interface{}{
				kuardService,
				kuardService2,
				kuardService3,
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{"tcp.projectcontour.io"},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: gatewayapi.TLSRouteBackendRefs(
								gatewayapi.TLSRouteBackendRef("kuard", 8080, pointer.Int32Ptr(1)),
								gatewayapi.TLSRouteBackendRef("kuard2", 8080, pointer.Int32Ptr(0)),
								gatewayapi.TLSRouteBackendRef("kuard3", 8080, pointer.Int32Ptr(3)),
							),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTPS_LISTENER_NAME,
					Port: 443,
					SecureVirtualHosts: securevirtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "tcp.projectcontour.io",
							},
							TCPProxy: &TCPProxy{

								Clusters: clustersWeight(
									weightedService(kuardService, 1),
									weightedService(kuardService2, 0),
									weightedService(kuardService3, 3),
								),
							},
						},
					),
				},
			),
		},
		"TLSRoute with multiple unweighted ForwardTos all default to 1": {
			gatewayclass: validClass,
			gateway:      gatewayTLSPassthroughAllNamespaces,
			objs: []interface{}{
				kuardService,
				kuardService2,
				kuardService3,
				&gatewayapi_v1alpha2.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1alpha2.TLSRouteSpec{
						CommonRouteSpec: gatewayapi_v1alpha2.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1alpha2.ParentReference{gatewayapi.GatewayParentRefV1Alpha2("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1alpha2.Hostname{"tcp.projectcontour.io"},
						Rules: []gatewayapi_v1alpha2.TLSRouteRule{{
							BackendRefs: gatewayapi.TLSRouteBackendRefs(
								gatewayapi.TLSRouteBackendRef("kuard", 8080, nil),
								gatewayapi.TLSRouteBackendRef("kuard2", 8080, nil),
								gatewayapi.TLSRouteBackendRef("kuard3", 8080, nil),
							),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTPS_LISTENER_NAME,
					Port: 443,
					SecureVirtualHosts: securevirtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "tcp.projectcontour.io",
							},
							TCPProxy: &TCPProxy{

								Clusters: clustersWeight(
									weightedService(kuardService, 1),
									weightedService(kuardService2, 1),
									weightedService(kuardService3, 1),
								),
							},
						},
					),
				},
			),
		},
		"insert gateway listener with host": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPWithHostname,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{
						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchExact, "/blog"),
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("gateway.projectcontour.io",
							exactrouteHTTPRoute("/blog", service(kuardService))),
					),
				},
			),
		},
		"insert gateway listener with host, httproute with host": {
			gatewayclass: validClass,
			gateway:      gatewayHTTPWithWildcardHostname,
			objs: []interface{}{
				kuardService,
				&gatewayapi_v1beta1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "basic",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.HTTPRouteSpec{
						CommonRouteSpec: gatewayapi_v1beta1.CommonRouteSpec{
							ParentRefs: []gatewayapi_v1beta1.ParentReference{gatewayapi.GatewayParentRef("projectcontour", "contour")},
						},
						Hostnames: []gatewayapi_v1beta1.Hostname{
							"http.projectcontour.io",
						},
						Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
							Matches:     gatewayapi.HTTPRouteMatch(gatewayapi_v1beta1.PathMatchExact, "/blog"),
							BackendRefs: gatewayapi.HTTPBackendRef("kuard", 8080, 1),
						}},
					},
				},
			},
			want: listeners(
				&Listener{
					Name: HTTP_LISTENER_NAME,
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("http.projectcontour.io",
							exactrouteHTTPRoute("/blog", service(kuardService))),
					),
				},
			),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			builder := Builder{
				Source: KubernetesCache{
					gatewayclass: tc.gatewayclass,
					gateway:      tc.gateway,
					FieldLogger:  fixture.NewTestLogger(t),
				},
				Processors: []Processor{
					&GatewayAPIProcessor{
						FieldLogger: fixture.NewTestLogger(t),
					},
					&ListenerProcessor{},
				},
			}

			for _, o := range tc.objs {
				builder.Source.Insert(o)
			}
			dag := builder.Build()

			got := make(map[int]*Listener)
			for _, l := range dag.Listeners {
				got[l.Port] = l
			}

			want := make(map[int]*Listener)
			for _, v := range tc.want {
				want[v.Port] = v
			}
			assert.Equal(t, want, got)
		})
	}
}

func TestBuilderRunsProcessorsInOrder(t *testing.T) {
	var got []string

	b := Builder{
		Processors: []Processor{
			ProcessorFunc(func(*DAG, *KubernetesCache) { got = append(got, "foo") }),
			ProcessorFunc(func(*DAG, *KubernetesCache) { got = append(got, "bar") }),
			ProcessorFunc(func(*DAG, *KubernetesCache) { got = append(got, "baz") }),
			ProcessorFunc(func(*DAG, *KubernetesCache) { got = append(got, "abc") }),
			ProcessorFunc(func(*DAG, *KubernetesCache) { got = append(got, "def") }),
		},
	}

	b.Build()

	assert.Equal(t, []string{"foo", "bar", "baz", "abc", "def"}, got)
}

func routes(routes ...*Route) map[string]*Route {
	if len(routes) == 0 {
		return nil
	}
	m := make(map[string]*Route)
	for _, r := range routes {
		m[conditionsToString(r)] = r
	}
	return m
}

func directResponseRoute(prefix string, statusCode uint32) *Route {
	return &Route{
		PathMatchCondition: prefixString(prefix),
		DirectResponse:     &DirectResponse{StatusCode: statusCode},
	}
}

func directResponseRouteService(prefix string, statusCode uint32, first *Service, rest ...*Service) *Route {
	services := append([]*Service{first}, rest...)
	return &Route{
		PathMatchCondition: prefixString(prefix),
		DirectResponse:     &DirectResponse{StatusCode: statusCode},
		Clusters:           clustersWeight(services...),
	}
}

func prefixrouteHTTPRoute(prefix string, first *Service, rest ...*Service) *Route {
	services := append([]*Service{first}, rest...)
	return &Route{
		PathMatchCondition: prefixString(prefix),
		Clusters:           clustersWeight(services...),
	}
}

func segmentPrefixHTTPRoute(prefix string, first *Service, rest ...*Service) *Route {
	services := append([]*Service{first}, rest...)
	return &Route{
		PathMatchCondition: prefixSegment(prefix),
		Clusters:           clustersWeight(services...),
	}
}

func exactrouteHTTPRoute(path string, first *Service, rest ...*Service) *Route {
	services := append([]*Service{first}, rest...)
	return &Route{
		PathMatchCondition: &ExactMatchCondition{Path: path},
		Clusters:           clustersWeight(services...),
	}
}

func clusterHeaders(requestSet map[string]string, requestAdd map[string]string, requestRemove []string, hostRewrite string, services ...*Service) (c []*Cluster) {
	for _, s := range services {
		c = append(c, &Cluster{
			Upstream: s,
			Protocol: s.Protocol,
			RequestHeadersPolicy: &HeadersPolicy{
				Set:         requestSet,
				Add:         requestAdd,
				Remove:      requestRemove,
				HostRewrite: hostRewrite,
			},
			Weight: s.Weighted.Weight,
		})
	}
	return c
}

func clustersWeight(services ...*Service) (c []*Cluster) {
	for _, s := range services {
		c = append(c, &Cluster{
			Upstream: s,
			Protocol: s.Protocol,
			Weight:   s.Weighted.Weight,
		})
	}
	return c
}

func service(s *v1.Service) *Service {
	return weightedService(s, 1)
}

func weightedService(s *v1.Service, weight uint32) *Service {
	return &Service{
		Weighted: WeightedService{
			Weight:           weight,
			ServiceName:      s.Name,
			ServiceNamespace: s.Namespace,
			ServicePort:      s.Spec.Ports[0],
		},
	}
}

func secret(s *v1.Secret) *Secret {
	return &Secret{
		Object: s,
	}
}

func virtualhosts(vx ...*VirtualHost) []*VirtualHost {
	return vx
}

func securevirtualhosts(vx ...*SecureVirtualHost) []*SecureVirtualHost {
	return vx
}

func virtualhost(name string, first *Route, rest ...*Route) *VirtualHost {
	return &VirtualHost{
		Name:   name,
		Routes: routes(append([]*Route{first}, rest...)...),
	}
}

func listeners(ls ...*Listener) []*Listener {
	var v []*Listener
	v = append(v, ls...)
	return v
}

func prefixString(prefix string) MatchCondition {
	return &PrefixMatchCondition{Prefix: prefix, PrefixMatchType: PrefixMatchString}
}
func prefixSegment(prefix string) MatchCondition {
	return &PrefixMatchCondition{Prefix: prefix, PrefixMatchType: PrefixMatchSegment}
}

func withMirror(r *Route, mirror *Service) *Route {
	r.MirrorPolicy = &MirrorPolicy{
		Cluster: &Cluster{
			Upstream: mirror,
		},
	}
	return r
}
