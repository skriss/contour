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

package main

import (
	"testing"
	"time"

	"github.com/projectcontour/contour/internal/dag"
	"github.com/projectcontour/contour/internal/timeout"
	xdscache_v3 "github.com/projectcontour/contour/internal/xdscache/v3"
	"github.com/projectcontour/contour/pkg/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
)

func TestGetDAGBuilder(t *testing.T) {
	commonAssertions := func(t *testing.T, builder *dag.Builder) {
		t.Helper()

		// note that these first two assertions will not hold when a gateway
		// is configured, but we don't currently have test cases that cover
		// that so it's OK to keep them in the "common" assertions for now.
		assert.Len(t, builder.Processors, 4)
		assert.Equal(t, types.NamespacedName{}, builder.Source.ConfiguredGateway)

		assert.IsType(t, &dag.ListenerProcessor{}, builder.Processors[len(builder.Processors)-1])
	}

	t.Run("all default options", func(t *testing.T) {
		got := getDAGBuilder(newServeContext(), nil, nil, nil, logrus.StandardLogger())
		commonAssertions(t, &got)
		assert.Empty(t, got.Source.ConfiguredSecretRefs)
	})

	t.Run("client cert specified", func(t *testing.T) {
		clientCert := &types.NamespacedName{Namespace: "client-ns", Name: "client-name"}

		got := getDAGBuilder(newServeContext(), nil, clientCert, nil, logrus.StandardLogger())
		commonAssertions(t, &got)
		assert.ElementsMatch(t, got.Source.ConfiguredSecretRefs, []*types.NamespacedName{clientCert})
	})

	t.Run("fallback cert specified", func(t *testing.T) {
		fallbackCert := &types.NamespacedName{Namespace: "fallback-ns", Name: "fallback-name"}

		got := getDAGBuilder(newServeContext(), nil, nil, fallbackCert, logrus.StandardLogger())
		commonAssertions(t, &got)
		assert.ElementsMatch(t, got.Source.ConfiguredSecretRefs, []*types.NamespacedName{fallbackCert})
	})

	t.Run("client and fallback certs specified", func(t *testing.T) {
		clientCert := &types.NamespacedName{Namespace: "client-ns", Name: "client-name"}
		fallbackCert := &types.NamespacedName{Namespace: "fallback-ns", Name: "fallback-name"}

		got := getDAGBuilder(newServeContext(), nil, clientCert, fallbackCert, logrus.StandardLogger())

		commonAssertions(t, &got)
		assert.ElementsMatch(t, got.Source.ConfiguredSecretRefs, []*types.NamespacedName{clientCert, fallbackCert})
	})

	t.Run("request and response headers policy specified", func(t *testing.T) {
		ctx := newServeContext()
		ctx.Config.Policy.RequestHeadersPolicy.Set = map[string]string{
			"req-set-key-1": "req-set-val-1",
			"req-set-key-2": "req-set-val-2",
		}
		ctx.Config.Policy.RequestHeadersPolicy.Remove = []string{"req-remove-key-1", "req-remove-key-2"}
		ctx.Config.Policy.ResponseHeadersPolicy.Set = map[string]string{
			"res-set-key-1": "res-set-val-1",
			"res-set-key-2": "res-set-val-2",
		}
		ctx.Config.Policy.ResponseHeadersPolicy.Remove = []string{"res-remove-key-1", "res-remove-key-2"}

		got := getDAGBuilder(ctx, nil, nil, nil, logrus.StandardLogger())
		commonAssertions(t, &got)

		httpProxyProcessor := mustGetHTTPProxyProcessor(t, &got)
		assert.EqualValues(t, ctx.Config.Policy.RequestHeadersPolicy.Set, httpProxyProcessor.RequestHeadersPolicy.Set)
		assert.ElementsMatch(t, ctx.Config.Policy.RequestHeadersPolicy.Remove, httpProxyProcessor.RequestHeadersPolicy.Remove)
		assert.EqualValues(t, ctx.Config.Policy.ResponseHeadersPolicy.Set, httpProxyProcessor.ResponseHeadersPolicy.Set)
		assert.ElementsMatch(t, ctx.Config.Policy.ResponseHeadersPolicy.Remove, httpProxyProcessor.ResponseHeadersPolicy.Remove)
	})

	// TODO(3453): test additional properties of the DAG builder (processor fields, cache fields, Gateway tests (requires a client fake))
}

func mustGetHTTPProxyProcessor(t *testing.T, builder *dag.Builder) *dag.HTTPProxyProcessor {
	t.Helper()
	for i := range builder.Processors {
		found, ok := builder.Processors[i].(*dag.HTTPProxyProcessor)
		if ok {
			return found
		}
	}

	require.FailNow(t, "HTTPProxyProcessor not found in list of DAG builder's processors")
	return nil
}

func TestGetListenerConfig(t *testing.T) {
	// defaultListenerConfig returns the listener config that's
	// expected given a serveContext with no fields specified.
	defaultListenerConfig := func() xdscache_v3.ListenerConfig {
		return xdscache_v3.ListenerConfig{
			HTTPListeners: map[string]xdscache_v3.Listener{
				"ingress_http": {Name: "ingress_http"},
			},
			HTTPSListeners: map[string]xdscache_v3.Listener{
				"ingress_https": {Name: "ingress_https"},
			},
			MinimumTLSVersion: "1.2",
			CipherSuites: []string{
				"[ECDHE-ECDSA-AES128-GCM-SHA256|ECDHE-ECDSA-CHACHA20-POLY1305]",
				"[ECDHE-RSA-AES128-GCM-SHA256|ECDHE-RSA-CHACHA20-POLY1305]",
				"ECDHE-ECDSA-AES256-GCM-SHA384",
				"ECDHE-RSA-AES256-GCM-SHA384",
			},
			AllowChunkedLength: true,
		}
	}

	tests := map[string]struct {
		ctx       *serveContext
		setupWant func(*xdscache_v3.ListenerConfig)
	}{
		"default serve context": {
			ctx:       &serveContext{},
			setupWant: func(*xdscache_v3.ListenerConfig) {},
		},
		"timeout values specified": {
			ctx: &serveContext{
				Config: config.Parameters{
					Timeouts: config.TimeoutParameters{
						ConnectionIdleTimeout:         "1s",
						StreamIdleTimeout:             "2s",
						DelayedCloseTimeout:           "3s",
						MaxConnectionDuration:         "4s",
						ConnectionShutdownGracePeriod: "5s",
						RequestTimeout:                "6s",
					},
				},
			},
			setupWant: func(cfg *xdscache_v3.ListenerConfig) {
				cfg.ConnectionIdleTimeout = timeout.DurationSetting(1 * time.Second)
				cfg.StreamIdleTimeout = timeout.DurationSetting(2 * time.Second)
				cfg.DelayedCloseTimeout = timeout.DurationSetting(3 * time.Second)
				cfg.MaxConnectionDuration = timeout.DurationSetting(4 * time.Second)
				cfg.ConnectionShutdownGracePeriod = timeout.DurationSetting(5 * time.Second)
				cfg.RequestTimeout = timeout.DurationSetting(6 * time.Second)
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			want := defaultListenerConfig()
			tc.setupWant(&want)

			got, err := getListenerConfig(logrus.StandardLogger(), tc.ctx, nil)
			require.NoError(t, err)

			assert.Equal(t, want, got)
		})
	}
}
