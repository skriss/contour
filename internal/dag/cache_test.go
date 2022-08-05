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
	"context"
	"errors"
	"testing"

	"github.com/projectcontour/contour/internal/fixture"
	"github.com/projectcontour/contour/internal/gatewayapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapi_v1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayapi_v1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func TestKubernetesCacheInsert(t *testing.T) {
	tests := map[string]struct {
		cacheGateway *types.NamespacedName
		pre          []interface{}
		obj          interface{}
		want         bool
	}{
		"insert secret": {
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(fixture.CERTIFICATE, fixture.RSA_PRIVATE_KEY),
			},
			want: false,
		},
		"insert secret w/ blank ca.crt": {
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeTLS,
				Data: map[string][]byte{
					CACertificateKey:    []byte(""),
					v1.TLSCertKey:       []byte(fixture.CERTIFICATE),
					v1.TLSPrivateKeyKey: []byte(fixture.RSA_PRIVATE_KEY),
				},
			},
			want: true,
		},
		"insert CA secret w/ explanatory text": {
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeOpaque,
				Data: map[string][]byte{
					CACertificateKey: []byte(fixture.CERTIFICATE_WITH_TEXT),
				},
			},
			want: true,
		},
		"insert CA bundle secret w/ non-PEM data": {
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeOpaque,
				Data: caBundleData(fixture.CERTIFICATE, fixture.CERTIFICATE, fixture.CERTIFICATE, fixture.CERTIFICATE),
			},
			want: true,
		},
		"insert CA bundle secret w/ no CN for CA": {
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "caNoCN",
					Namespace: "default",
				},
				Type: v1.SecretTypeOpaque,
				Data: caBundleData(fixture.CA_CERT_NO_CN),
			},
			want: true,
		},

		"insert CA bundle secret w/ non-PEM data and no certificates": {
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeOpaque,
				Data: caBundleData(),
			},
			want: false,
		},

		"insert certificate secret": {
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ca",
					Namespace: "default",
				},
				Type: v1.SecretTypeOpaque,
				Data: map[string][]byte{
					CACertificateKey: []byte(fixture.CERTIFICATE),
				},
			},
			// TODO(dfc) this should be false because the CA secret is
			// not referenced, but computing its reference duplicates the
			// work done rebuilding the dag so for the moment assume that
			// any CA secret causes a rebuild.
			want: true,
		},
		"insert unknown": {
			obj:  "not an object",
			want: false,
		},
		"insert service": {
			obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service",
					Namespace: "default",
				},
			},
			want: false,
		},
		"insert namespace": {
			obj: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "namespace",
					Namespace: "default",
				},
			},
			want: true,
		},
		// invalid gatewayclass test case is unneeded since the controller
		// uses a predicate to filter events before they're given to the EventHandler.
		"insert valid gatewayclass": {
			obj: &gatewayapi_v1beta1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "contour",
				},
			},
			want: true,
		},
		"insert gateway-api Gateway": {
			obj: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "contour",
					Namespace: "projectcontour",
				},
			},
			want: true,
		},
		"insert gateway-api HTTPRoute": {
			obj: &gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "httproute",
					Namespace: "default",
				},
			},
			want: true,
		},
		"insert gateway-api TLSRoute": {
			obj: &gatewayapi_v1alpha2.TLSRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tlsroute",
					Namespace: "default",
				},
			},
			want: true,
		},
		"insert gateway-api ReferenceGrant": {
			obj: &gatewayapi_v1alpha2.ReferenceGrant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "referencegrant-1",
					Namespace: "default",
				},
			},
			want: true,
		},
		"insert secret that is referred by configuration file": {
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secretReferredByConfigFile",
					Namespace: "default",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(fixture.CERTIFICATE, fixture.RSA_PRIVATE_KEY),
			},
			want: true,
		},

		// SPECIFIC GATEWAY TESTS
		"specific gateway configured, insert gatewayclass, no gateway cached": {
			cacheGateway: &types.NamespacedName{
				Namespace: "gateway-namespace",
				Name:      "gateway-name",
			},
			obj: &gatewayapi_v1beta1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "gatewayclass-1",
				},
			},
			want: false,
		},
		"specific gateway configured, insert gatewayclass, gateway cached referencing different gatewayclass": {
			cacheGateway: &types.NamespacedName{
				Namespace: "gateway-namespace",
				Name:      "gateway-name",
			},
			pre: []interface{}{
				&gatewayapi_v1beta1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "gateway-namespace",
						Name:      "gateway-name",
					},
					Spec: gatewayapi_v1beta1.GatewaySpec{
						GatewayClassName: gatewayapi_v1beta1.ObjectName("some-other-gatewayclass"),
					},
				},
			},
			obj: &gatewayapi_v1beta1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "gatewayclass-1",
				},
			},
			want: false,
		},
		"specific gateway configured, insert gatewayclass, gateway cached referencing matching gatewayclass": {
			cacheGateway: &types.NamespacedName{
				Namespace: "gateway-namespace",
				Name:      "gateway-name",
			},
			pre: []interface{}{
				&gatewayapi_v1beta1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "gateway-namespace",
						Name:      "gateway-name",
					},
					Spec: gatewayapi_v1beta1.GatewaySpec{
						GatewayClassName: gatewayapi_v1beta1.ObjectName("gatewayclass-1"),
					},
				},
			},
			obj: &gatewayapi_v1beta1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "gatewayclass-1",
				},
			},
			want: true,
		},
		"specific gateway configured, insert gateway, namespace/name don't match": {
			cacheGateway: &types.NamespacedName{
				Namespace: "gateway-namespace",
				Name:      "gateway-name",
			},
			obj: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "gateway-namespace",
					Name:      "some-other-gateway-name",
				},
			},
			want: false,
		},
		"specific gateway configured, insert gateway, namespace/name match": {
			cacheGateway: &types.NamespacedName{
				Namespace: "gateway-namespace",
				Name:      "gateway-name",
			},
			obj: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "gateway-namespace",
					Name:      "gateway-name",
				},
			},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cache := KubernetesCache{
				ConfiguredGatewayToCache: tc.cacheGateway,
				ConfiguredSecretRefs: []*types.NamespacedName{
					{Name: "secretReferredByConfigFile", Namespace: "default"}},
				FieldLogger: fixture.NewTestLogger(t),
				Client:      new(fakeReader),
			}
			for _, p := range tc.pre {
				cache.Insert(p)
			}
			got := cache.Insert(tc.obj)
			assert.Equalf(t, tc.want, got, "Insert failed for object %v ", tc.obj)
		})
	}
}

// Simple fake for use with specific Gateway test cases,
// just returns an error on Get. This could be improved
// or replaced with a mock but would also require
// further changes to the test structure to be useful for
// validating that the gateway's gatewayclass is fetched
// correctly.
type fakeReader struct{}

func (r *fakeReader) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	return errors.New("not implemented")
}

func (r *fakeReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	panic("not implemented")
}

func TestKubernetesCacheRemove(t *testing.T) {
	cache := func(objs ...interface{}) *KubernetesCache {
		cache := KubernetesCache{
			FieldLogger: fixture.NewTestLogger(t),
		}
		for _, o := range objs {
			cache.Insert(o)
		}
		return &cache
	}

	tests := map[string]struct {
		cache *KubernetesCache
		obj   interface{}
		want  bool
	}{
		"remove secret": {
			cache: cache(&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeTLS,
				Data: map[string][]byte{
					v1.TLSCertKey:       []byte(fixture.CERTIFICATE),
					v1.TLSPrivateKeyKey: []byte(fixture.RSA_PRIVATE_KEY),
				},
			}),
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeTLS,
			},
			want: true,
		},
		"remove service": {
			cache: cache(&v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service",
					Namespace: "default",
				},
			}),
			obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service",
					Namespace: "default",
				},
			},
			want: true,
		},
		"remove namespace": {
			cache: cache(&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "namespace",
					Namespace: "default",
				},
			}),
			obj: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "namespace",
					Namespace: "default",
				},
			},
			want: true,
		},
		"remove gatewayclass": {
			cache: cache(&gatewayapi_v1beta1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "contour",
				},
			}),
			obj: &gatewayapi_v1beta1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "contour",
				},
			},
			want: true,
		},
		"remove gateway-api Gateway": {
			cache: cache(&gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "contour",
					Namespace: "projectcontour",
				},
			}),
			obj: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "contour",
					Namespace: "projectcontour",
				},
			},
			want: true,
		},
		"remove gateway-api HTTPRoute": {
			cache: cache(&gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "httproute",
					Namespace: "default",
				},
			}),
			obj: &gatewayapi_v1beta1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "httproute",
					Namespace: "default",
				},
			},
			want: true,
		},
		"remove gateway-api TLSRoute": {
			cache: cache(&gatewayapi_v1alpha2.TLSRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tlsroute",
					Namespace: "default",
				},
			}),
			obj: &gatewayapi_v1alpha2.TLSRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tlsroute",
					Namespace: "default",
				},
			},
			want: true,
		},
		"remove gateway-api ReferenceGrant": {
			cache: cache(&gatewayapi_v1alpha2.ReferenceGrant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "referencegrant",
					Namespace: "default",
				},
			}),
			obj: &gatewayapi_v1alpha2.ReferenceGrant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "referencegrant",
					Namespace: "default",
				},
			},
			want: true,
		},
		"remove unknown": {
			cache: cache("not an object"),
			obj:   "not an object",
			want:  false,
		},
		"specific gateway configured, remove gatewayclass, no gatewayclass cached": {
			cache: &KubernetesCache{
				ConfiguredGatewayToCache: &types.NamespacedName{Namespace: "gateway-namespace", Name: "gateway-name"},
			},
			obj: &gatewayapi_v1beta1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "gatewayclass-1",
				},
			},
			want: false,
		},
		"specific gateway configured, remove gatewayclass, non-matching name": {
			cache: &KubernetesCache{
				ConfiguredGatewayToCache: &types.NamespacedName{Namespace: "gateway-namespace", Name: "gateway-name"},
				gatewayclass: &gatewayapi_v1beta1.GatewayClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "gatewayclass-1",
					},
				},
			},
			obj: &gatewayapi_v1beta1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "some-other-gatewayclass",
				},
			},
			want: false,
		},
		"specific gateway configured, remove gatewayclass, matching name": {
			cache: &KubernetesCache{
				ConfiguredGatewayToCache: &types.NamespacedName{Namespace: "gateway-namespace", Name: "gateway-name"},
				gatewayclass: &gatewayapi_v1beta1.GatewayClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "gatewayclass-1",
					},
				},
			},
			obj: &gatewayapi_v1beta1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "gatewayclass-1",
				},
			},
			want: true,
		},
		"specific gateway configured, remove gateway, no gateway cached": {
			cache: &KubernetesCache{
				ConfiguredGatewayToCache: &types.NamespacedName{Namespace: "gateway-namespace", Name: "gateway-name"},
			},
			obj: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "gateway-namespace",
					Name:      "gateway-name",
				},
			},
			want: false,
		},
		"specific gateway configured, remove gateway, non-matching namespace/name": {
			cache: &KubernetesCache{
				ConfiguredGatewayToCache: &types.NamespacedName{Namespace: "gateway-namespace", Name: "gateway-name"},
				gateway: &gatewayapi_v1beta1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "gateway-namespace",
						Name:      "gateway-name",
					},
				},
			},
			obj: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "gateway-namespace",
					Name:      "some-other-gateway",
				},
			},
			want: false,
		},
		"specific gateway configured, remove gateway, matching namespace/name": {
			cache: &KubernetesCache{
				ConfiguredGatewayToCache: &types.NamespacedName{Namespace: "gateway-namespace", Name: "gateway-name"},
				gateway: &gatewayapi_v1beta1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "gateway-namespace",
						Name:      "gateway-name",
					},
				},
			},
			obj: &gatewayapi_v1beta1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "gateway-namespace",
					Name:      "gateway-name",
				},
			},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.cache.Remove(tc.obj)
			assert.Equalf(t, tc.want, got, "Remove failed for object %v ", tc.obj)
		})
	}
}

func TestLookupService(t *testing.T) {
	cache := func(objs ...interface{}) *KubernetesCache {
		cache := KubernetesCache{
			FieldLogger: fixture.NewTestLogger(t),
		}
		for _, o := range objs {
			cache.Insert(o)
		}
		return &cache
	}

	service := func(ns, name string, ports ...v1.ServicePort) *v1.Service {
		return &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Spec: v1.ServiceSpec{
				Ports: ports,
			},
		}
	}

	port := func(name string, port int32, protocol v1.Protocol) v1.ServicePort {
		return v1.ServicePort{
			Name:     name,
			Port:     port,
			Protocol: protocol,
		}
	}

	tests := map[string]struct {
		cache    *KubernetesCache
		meta     types.NamespacedName
		port     intstr.IntOrString
		wantSvc  *v1.Service
		wantPort v1.ServicePort
		wantErr  error
	}{
		"service and port exist with valid service protocol, lookup by port num": {
			cache:    cache(service("default", "service-1", port("http", 80, v1.ProtocolTCP))),
			meta:     types.NamespacedName{Namespace: "default", Name: "service-1"},
			port:     intstr.FromInt(80),
			wantSvc:  service("default", "service-1", port("http", 80, v1.ProtocolTCP)),
			wantPort: port("http", 80, v1.ProtocolTCP),
		},
		"service and port exist with valid service protocol, lookup by port name": {
			cache:    cache(service("default", "service-1", port("http", 80, v1.ProtocolTCP))),
			meta:     types.NamespacedName{Namespace: "default", Name: "service-1"},
			port:     intstr.FromString("http"),
			wantSvc:  service("default", "service-1", port("http", 80, v1.ProtocolTCP)),
			wantPort: port("http", 80, v1.ProtocolTCP),
		},
		"service and port exist with valid service protocol, lookup by wrong port num": {
			cache:   cache(service("default", "service-1", port("http", 80, v1.ProtocolTCP))),
			meta:    types.NamespacedName{Namespace: "default", Name: "service-1"},
			port:    intstr.FromInt(9999),
			wantErr: errors.New(`port "9999" on service "default/service-1" not matched`),
		},
		"service and port exist with valid service protocol, lookup by wrong port name": {
			cache:   cache(service("default", "service-1", port("http", 80, v1.ProtocolTCP))),
			meta:    types.NamespacedName{Namespace: "default", Name: "service-1"},
			port:    intstr.FromString("wrong-port-name"),
			wantErr: errors.New(`port "wrong-port-name" on service "default/service-1" not matched`),
		},
		"service and port exist, invalid service protocol": {
			cache:   cache(service("default", "service-1", port("http", 80, v1.ProtocolUDP))),
			meta:    types.NamespacedName{Namespace: "default", Name: "service-1"},
			port:    intstr.FromString("http"),
			wantSvc: service("default", "service-1", port("http", 80, v1.ProtocolTCP)),
			wantErr: errors.New(`unsupported service protocol "UDP"`),
		},
		"service does not exist": {
			cache:   cache(service("default", "service-1", port("http", 80, v1.ProtocolTCP))),
			meta:    types.NamespacedName{Namespace: "default", Name: "nonexistent-service"},
			port:    intstr.FromInt(80),
			wantErr: errors.New(`service "default/nonexistent-service" not found`),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotSvc, gotPort, gotErr := tc.cache.LookupService(tc.meta, tc.port)

			switch {
			case tc.wantErr != nil:
				require.Error(t, gotErr)
				assert.EqualError(t, tc.wantErr, gotErr.Error())
			default:
				assert.Nil(t, gotErr)
				assert.Equal(t, tc.wantSvc, gotSvc)
				assert.Equal(t, tc.wantPort, gotPort)
			}
		})
	}
}

func TestServiceTriggersRebuild(t *testing.T) {

	cache := func(objs ...interface{}) *KubernetesCache {
		cache := KubernetesCache{
			FieldLogger: fixture.NewTestLogger(t),
		}
		for _, o := range objs {
			cache.Insert(o)
		}
		return &cache
	}

	service := func(namespace, name string) *v1.Service {
		return &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
	}

	httpRoute := func(namespace, name string) *gatewayapi_v1beta1.HTTPRoute {
		return &gatewayapi_v1beta1.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: gatewayapi_v1beta1.HTTPRouteSpec{
				Rules: []gatewayapi_v1beta1.HTTPRouteRule{{
					BackendRefs: gatewayapi.HTTPBackendRef(name, 80, 1),
				}},
			},
		}
	}

	tests := map[string]struct {
		cache *KubernetesCache
		svc   *v1.Service
		want  bool
	}{
		"empty cache does not trigger rebuild": {
			cache: cache(),
			svc:   service("default", "service-1"),
			want:  false,
		},
		"httproute exists in same namespace as service": {
			cache: cache(
				service("default", "service-1"),
				httpRoute("default", "service-1"),
			),
			svc:  service("default", "service-1"),
			want: true,
		},
		"httproute does not exist in same namespace as service": {
			cache: cache(
				service("default", "service-1"),
				httpRoute("user", "service-1"),
			),
			svc:  service("default", "service-1"),
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.cache.serviceTriggersRebuild(tc.svc))
		})
	}
}

func TestSecretTriggersRebuild(t *testing.T) {

	secret := func(namespace, name string) *v1.Secret {
		return &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Type: v1.SecretTypeTLS,
			Data: secretdata(fixture.CERTIFICATE, fixture.RSA_PRIVATE_KEY),
		}
	}

	caSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ca",
			Namespace: "default",
		},
		Data: map[string][]byte{
			CACertificateKey: []byte(fixture.CERTIFICATE),
		},
	}

	cache := func(objs ...interface{}) *KubernetesCache {
		cache := KubernetesCache{
			FieldLogger: fixture.NewTestLogger(t),
		}
		for _, o := range objs {
			cache.Insert(o)
		}
		return &cache
	}

	tests := map[string]struct {
		cache  *KubernetesCache
		secret *v1.Secret
		want   bool
	}{
		"empty cache does not trigger rebuild": {
			cache:  cache(),
			secret: secret("default", "secret"),
			want:   false,
		},
		"CA secret triggers rebuild": {
			cache:  cache(),
			secret: caSecret,
			want:   true,
		},
		"configuration file secret triggers rebuild": {
			cache: &KubernetesCache{
				FieldLogger: fixture.NewTestLogger(t),
				ConfiguredSecretRefs: []*types.NamespacedName{{
					Namespace: "user",
					Name:      "tlscert",
				}},
			},
			secret: secret("user", "tlscert"),
			want:   true,
		},
		"no defined gateway does not trigger rebuild": {
			cache: &KubernetesCache{
				FieldLogger: fixture.NewTestLogger(t),
				gateway:     nil,
			},
			secret: secret("default", "tlscert"),
			want:   false,
		},
		"gateway does not define TLS on listener, does not trigger rebuild": {
			cache: cache(
				&gatewayapi_v1beta1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "contour",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.GatewaySpec{
						Listeners: []gatewayapi_v1beta1.Listener{{
							TLS: nil,
						}},
					},
				},
			),
			secret: secret("default", "tlscert"),
			want:   false,
		},
		"gateway does not define TLS.CertificateRef on listener, does not trigger rebuild": {
			cache: cache(
				&gatewayapi_v1beta1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "contour",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.GatewaySpec{
						Listeners: []gatewayapi_v1beta1.Listener{{
							TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
								CertificateRefs: nil,
							},
						}},
					},
				},
			),
			secret: secret("default", "tlscert"),
			want:   false,
		},
		"gateway listener references secret, triggers rebuild (core Group)": {
			cache: cache(
				&gatewayapi_v1beta1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "contour",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.GatewaySpec{
						Listeners: []gatewayapi_v1beta1.Listener{{
							TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
								CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
									gatewayapi.CertificateRef("tlscert", ""),
								},
							},
						}},
					},
				},
			),
			secret: secret("projectcontour", "tlscert"),
			want:   true,
		},
		"gateway listener references secret, triggers rebuild (v1 Group)": {
			cache: cache(
				&gatewayapi_v1beta1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "contour",
						Namespace: "projectcontour",
					},
					Spec: gatewayapi_v1beta1.GatewaySpec{
						Listeners: []gatewayapi_v1beta1.Listener{{
							TLS: &gatewayapi_v1beta1.GatewayTLSConfig{
								CertificateRefs: []gatewayapi_v1beta1.SecretObjectReference{
									gatewayapi.CertificateRef("tlscert", ""),
								},
							},
						}},
					},
				},
			),
			secret: secret("projectcontour", "tlscert"),
			want:   true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.cache.secretTriggersRebuild(tc.secret))
		})
	}
}
