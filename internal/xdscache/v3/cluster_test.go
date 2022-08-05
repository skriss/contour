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
	"time"

	envoy_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"github.com/golang/protobuf/proto"
	envoy_v3 "github.com/projectcontour/contour/internal/envoy/v3"
	"github.com/projectcontour/contour/internal/protobuf"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClusterCacheContents(t *testing.T) {
	tests := map[string]struct {
		contents map[string]*envoy_cluster_v3.Cluster
		want     []proto.Message
	}{
		"empty": {
			contents: nil,
			want:     nil,
		},
		"simple": {
			contents: clustermap(
				&envoy_cluster_v3.Cluster{
					Name:                 "default/kuard/443/da39a3ee5e",
					AltStatName:          "default_kuard_443",
					ClusterDiscoveryType: envoy_v3.ClusterDiscoveryType(envoy_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy_v3.ConfigSource("contour"),
						ServiceName: "default/kuard",
					},
				}),
			want: []proto.Message{
				cluster(&envoy_cluster_v3.Cluster{
					Name:                 "default/kuard/443/da39a3ee5e",
					AltStatName:          "default_kuard_443",
					ClusterDiscoveryType: envoy_v3.ClusterDiscoveryType(envoy_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy_v3.ConfigSource("contour"),
						ServiceName: "default/kuard",
					},
				}),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var cc ClusterCache
			cc.Update(tc.contents)
			got := cc.Contents()
			protobuf.ExpectEqual(t, tc.want, got)
		})
	}
}

func TestClusterCacheQuery(t *testing.T) {
	tests := map[string]struct {
		contents map[string]*envoy_cluster_v3.Cluster
		query    []string
		want     []proto.Message
	}{
		"exact match": {
			contents: clustermap(
				&envoy_cluster_v3.Cluster{
					Name:                 "default/kuard/443/da39a3ee5e",
					AltStatName:          "default_kuard_443",
					ClusterDiscoveryType: envoy_v3.ClusterDiscoveryType(envoy_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy_v3.ConfigSource("contour"),
						ServiceName: "default/kuard",
					},
				}),
			query: []string{"default/kuard/443/da39a3ee5e"},
			want: []proto.Message{
				cluster(&envoy_cluster_v3.Cluster{
					Name:                 "default/kuard/443/da39a3ee5e",
					AltStatName:          "default_kuard_443",
					ClusterDiscoveryType: envoy_v3.ClusterDiscoveryType(envoy_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy_v3.ConfigSource("contour"),
						ServiceName: "default/kuard",
					},
				}),
			},
		},
		"partial match": {
			contents: clustermap(
				&envoy_cluster_v3.Cluster{
					Name:                 "default/kuard/443/da39a3ee5e",
					AltStatName:          "default_kuard_443",
					ClusterDiscoveryType: envoy_v3.ClusterDiscoveryType(envoy_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy_v3.ConfigSource("contour"),
						ServiceName: "default/kuard",
					},
				}),
			query: []string{"default/kuard/443/da39a3ee5e", "foo/bar/baz"},
			want: []proto.Message{
				cluster(&envoy_cluster_v3.Cluster{
					Name:                 "default/kuard/443/da39a3ee5e",
					AltStatName:          "default_kuard_443",
					ClusterDiscoveryType: envoy_v3.ClusterDiscoveryType(envoy_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy_v3.ConfigSource("contour"),
						ServiceName: "default/kuard",
					},
				}),
			},
		},
		"no match": {
			contents: clustermap(
				&envoy_cluster_v3.Cluster{
					Name:                 "default/kuard/443/da39a3ee5e",
					AltStatName:          "default_kuard_443",
					ClusterDiscoveryType: envoy_v3.ClusterDiscoveryType(envoy_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy_v3.ConfigSource("contour"),
						ServiceName: "default/kuard",
					},
				}),
			query: []string{"foo/bar/baz"},
			want:  nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var cc ClusterCache
			cc.Update(tc.contents)
			got := cc.Query(tc.query)
			protobuf.ExpectEqual(t, tc.want, got)
		})
	}
}

func service(ns, name string, ports ...v1.ServicePort) *v1.Service {
	return serviceWithAnnotations(ns, name, nil, ports...)
}

func serviceWithAnnotations(ns, name string, annotations map[string]string, ports ...v1.ServicePort) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   ns,
			Annotations: annotations,
		},
		Spec: v1.ServiceSpec{
			Ports: ports,
		},
	}
}

func cluster(c *envoy_cluster_v3.Cluster) *envoy_cluster_v3.Cluster {
	// NOTE: Keep this in sync with envoy.defaultCluster().
	defaults := &envoy_cluster_v3.Cluster{
		ConnectTimeout: protobuf.Duration(2 * time.Second),
		CommonLbConfig: envoy_v3.ClusterCommonLBConfig(),
		LbPolicy:       envoy_cluster_v3.Cluster_ROUND_ROBIN,
	}

	proto.Merge(defaults, c)
	return defaults
}

func clustermap(clusters ...*envoy_cluster_v3.Cluster) map[string]*envoy_cluster_v3.Cluster {
	m := make(map[string]*envoy_cluster_v3.Cluster)
	for _, c := range clusters {
		m[c.Name] = cluster(c)
	}
	return m
}
