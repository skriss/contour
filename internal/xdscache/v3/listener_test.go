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

package v3

import (
	"testing"

	envoy_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/golang/protobuf/proto"
	"github.com/projectcontour/contour/apis/projectcontour/v1alpha1"
	"github.com/projectcontour/contour/internal/dag"
	envoy_v3 "github.com/projectcontour/contour/internal/envoy/v3"
	"github.com/projectcontour/contour/internal/protobuf"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestListenerCacheContents(t *testing.T) {
	tests := map[string]struct {
		contents map[string]*envoy_listener_v3.Listener
		want     []proto.Message
	}{
		"empty": {
			contents: nil,
			want:     nil,
		},
		"simple": {
			contents: listenermap(&envoy_listener_v3.Listener{
				Name:          ENVOY_HTTP_LISTENER,
				Address:       envoy_v3.SocketAddress("0.0.0.0", 8080),
				FilterChains:  envoy_v3.FilterChains(envoy_v3.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy_v3.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG, "", nil, v1alpha1.LogLevelInfo), 0)),
				SocketOptions: envoy_v3.TCPKeepaliveSocketOptions(),
			}),
			want: []proto.Message{
				&envoy_listener_v3.Listener{
					Name:          ENVOY_HTTP_LISTENER,
					Address:       envoy_v3.SocketAddress("0.0.0.0", 8080),
					FilterChains:  envoy_v3.FilterChains(envoy_v3.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy_v3.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG, "", nil, v1alpha1.LogLevelInfo), 0)),
					SocketOptions: envoy_v3.TCPKeepaliveSocketOptions(),
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var lc ListenerCache
			lc.Update(tc.contents)
			got := lc.Contents()
			protobuf.ExpectEqual(t, tc.want, got)
		})
	}
}

func TestListenerCacheQuery(t *testing.T) {
	tests := map[string]struct {
		contents map[string]*envoy_listener_v3.Listener
		query    []string
		want     []proto.Message
	}{
		"exact match": {
			contents: listenermap(&envoy_listener_v3.Listener{
				Name:          ENVOY_HTTP_LISTENER,
				Address:       envoy_v3.SocketAddress("0.0.0.0", 8080),
				FilterChains:  envoy_v3.FilterChains(envoy_v3.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy_v3.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG, "", nil, v1alpha1.LogLevelInfo), 0)),
				SocketOptions: envoy_v3.TCPKeepaliveSocketOptions(),
			}),
			query: []string{ENVOY_HTTP_LISTENER},
			want: []proto.Message{
				&envoy_listener_v3.Listener{
					Name:          ENVOY_HTTP_LISTENER,
					Address:       envoy_v3.SocketAddress("0.0.0.0", 8080),
					FilterChains:  envoy_v3.FilterChains(envoy_v3.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy_v3.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG, "", nil, v1alpha1.LogLevelInfo), 0)),
					SocketOptions: envoy_v3.TCPKeepaliveSocketOptions(),
				},
			},
		},
		"partial match": {
			contents: listenermap(&envoy_listener_v3.Listener{
				Name:          ENVOY_HTTP_LISTENER,
				Address:       envoy_v3.SocketAddress("0.0.0.0", 8080),
				FilterChains:  envoy_v3.FilterChains(envoy_v3.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy_v3.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG, "", nil, v1alpha1.LogLevelInfo), 0)),
				SocketOptions: envoy_v3.TCPKeepaliveSocketOptions(),
			}),
			query: []string{ENVOY_HTTP_LISTENER, "stats-listener"},
			want: []proto.Message{
				&envoy_listener_v3.Listener{
					Name:          ENVOY_HTTP_LISTENER,
					Address:       envoy_v3.SocketAddress("0.0.0.0", 8080),
					FilterChains:  envoy_v3.FilterChains(envoy_v3.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy_v3.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG, "", nil, v1alpha1.LogLevelInfo), 0)),
					SocketOptions: envoy_v3.TCPKeepaliveSocketOptions(),
				},
			},
		},
		"no match": {
			contents: listenermap(&envoy_listener_v3.Listener{
				Name:          ENVOY_HTTP_LISTENER,
				Address:       envoy_v3.SocketAddress("0.0.0.0", 8080),
				FilterChains:  envoy_v3.FilterChains(envoy_v3.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy_v3.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG, "", nil, v1alpha1.LogLevelInfo), 0)),
				SocketOptions: envoy_v3.TCPKeepaliveSocketOptions(),
			}),
			query: []string{"stats-listener"},
			want:  nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var lc ListenerCache
			lc.Update(tc.contents)
			got := lc.Query(tc.query)
			protobuf.ExpectEqual(t, tc.want, got)
		})
	}
}

func transportSocket(secretname string, tlsMinProtoVersion envoy_tls_v3.TlsParameters_TlsProtocol, cipherSuites []string, alpnprotos ...string) *envoy_core_v3.TransportSocket {
	secret := &dag.Secret{
		Object: &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretname,
				Namespace: "default",
			},
			Type: v1.SecretTypeTLS,
			Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
		},
	}
	return envoy_v3.DownstreamTLSTransportSocket(
		envoy_v3.DownstreamTLSContext(secret, tlsMinProtoVersion, cipherSuites, nil, alpnprotos...),
	)
}

func listenermap(listeners ...*envoy_listener_v3.Listener) map[string]*envoy_listener_v3.Listener {
	m := make(map[string]*envoy_listener_v3.Listener)
	for _, l := range listeners {
		m[l.Name] = l
	}
	return m
}
