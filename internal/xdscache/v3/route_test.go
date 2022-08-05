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

	envoy_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/protobuf/proto"
	"github.com/projectcontour/contour/internal/dag"
	"github.com/projectcontour/contour/internal/protobuf"
	"github.com/stretchr/testify/assert"
)

func TestRouteCacheContents(t *testing.T) {
	tests := map[string]struct {
		contents map[string]*envoy_route_v3.RouteConfiguration
		want     []proto.Message
	}{
		"empty": {
			contents: nil,
			want:     nil,
		},
		"simple": {
			contents: map[string]*envoy_route_v3.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
			want: []proto.Message{
				&envoy_route_v3.RouteConfiguration{
					Name: "ingress_http",
				},
				&envoy_route_v3.RouteConfiguration{
					Name: "ingress_https",
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var rc RouteCache
			rc.Update(tc.contents)
			got := rc.Contents()
			protobuf.ExpectEqual(t, tc.want, got)
		})
	}
}

func TestRouteCacheQuery(t *testing.T) {
	tests := map[string]struct {
		contents map[string]*envoy_route_v3.RouteConfiguration
		query    []string
		want     []proto.Message
	}{
		"exact match": {
			contents: map[string]*envoy_route_v3.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
				},
			},
			query: []string{"ingress_http"},
			want: []proto.Message{
				&envoy_route_v3.RouteConfiguration{
					Name: "ingress_http",
				},
			},
		},
		"partial match": {
			contents: map[string]*envoy_route_v3.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
				},
			},
			query: []string{"stats-handler", "ingress_http"},
			want: []proto.Message{
				&envoy_route_v3.RouteConfiguration{
					Name: "ingress_http",
				},
				&envoy_route_v3.RouteConfiguration{
					Name: "stats-handler",
				},
			},
		},
		"no match": {
			contents: map[string]*envoy_route_v3.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
				},
			},
			query: []string{"stats-handler"},
			want: []proto.Message{
				&envoy_route_v3.RouteConfiguration{
					Name: "stats-handler",
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var rc RouteCache
			rc.Update(tc.contents)
			got := rc.Query(tc.query)
			protobuf.ExpectEqual(t, tc.want, got)
		})
	}
}

func TestSortLongestRouteFirst(t *testing.T) {
	tests := map[string]struct {
		routes []*dag.Route
		want   []*dag.Route
	}{
		"two prefixes": {
			routes: []*dag.Route{{
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/"},
			}, {
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/longer"},
			}},
			want: []*dag.Route{{
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/longer"},
			}, {
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/"},
			}},
		},
		"two regexes": {
			routes: []*dag.Route{{
				PathMatchCondition: &dag.RegexMatchCondition{Regex: "/v2"},
			}, {
				PathMatchCondition: &dag.RegexMatchCondition{Regex: "/v1/.+"},
			}},
			want: []*dag.Route{{
				PathMatchCondition: &dag.RegexMatchCondition{Regex: "/v2"},
			}, {
				PathMatchCondition: &dag.RegexMatchCondition{Regex: "/v1/.+"},
			}},
		},
		"two exact matches": {
			routes: []*dag.Route{{
				PathMatchCondition: &dag.ExactMatchCondition{Path: "/foo"},
			}, {
				PathMatchCondition: &dag.ExactMatchCondition{Path: "/foo/"},
			}},
			want: []*dag.Route{{
				PathMatchCondition: &dag.ExactMatchCondition{Path: "/foo/"},
			}, {
				PathMatchCondition: &dag.ExactMatchCondition{Path: "/foo"},
			}},
		},
		"exact sorts before regex sorts before prefix": {
			routes: []*dag.Route{{
				PathMatchCondition: &dag.RegexMatchCondition{Regex: "/api/v?"},
			}, {
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/"},
			}, {
				PathMatchCondition: &dag.RegexMatchCondition{Regex: ".*"},
			}, {
				PathMatchCondition: &dag.ExactMatchCondition{Path: "/api/"},
			}},
			want: []*dag.Route{{
				PathMatchCondition: &dag.ExactMatchCondition{Path: "/api/"},
			}, {
				PathMatchCondition: &dag.RegexMatchCondition{Regex: "/api/v?"},
			}, {
				PathMatchCondition: &dag.RegexMatchCondition{Regex: ".*"},
			}, {
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/"},
			}},
		},
		"more headers sort before less": {
			routes: []*dag.Route{{
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/"},
			}, {
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/"},
				HeaderMatchConditions: []dag.HeaderMatchCondition{
					{Name: "x-request-id", MatchType: "present"},
				},
			}},
			want: []*dag.Route{{
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/"},
				HeaderMatchConditions: []dag.HeaderMatchCondition{
					{Name: "x-request-id", MatchType: "present"},
				},
			}, {
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/"},
			}},
		},

		// Verify that longest path sorts before longest
		// headers. We used to sort by longest header list
		// first, which does end up with the same net result,
		// so this isn't strictly necessary.  However, ordering
		// the path first is arguably more intuitive, and
		// allows us to avoid comparing the header matches
		// unless necessary.
		"longest path before longest headers": {
			routes: []*dag.Route{{
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/"},
				HeaderMatchConditions: []dag.HeaderMatchCondition{
					{Name: "x-request-id", MatchType: "present"},
				},
			}, {
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/longest/path/match"},
			}},
			want: []*dag.Route{{
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/longest/path/match"},
			}, {
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/"},
				HeaderMatchConditions: []dag.HeaderMatchCondition{
					{Name: "x-request-id", MatchType: "present"},
				},
			}},
		},

		// The path and the length of header condition list are equal,
		// so we should order lexicographically by header name.
		"headers sort stably by name": {
			routes: []*dag.Route{{
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/"},
				HeaderMatchConditions: []dag.HeaderMatchCondition{
					{Name: "zzz-2", MatchType: "present"},
					{Name: "zzz-1", MatchType: "present"},
				},
			}, {
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/"},
				HeaderMatchConditions: []dag.HeaderMatchCondition{
					{Name: "aaa-2", MatchType: "present"},
					{Name: "aaa-1", MatchType: "present"},
				},
			}},
			want: []*dag.Route{{
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/"},
				HeaderMatchConditions: []dag.HeaderMatchCondition{
					{Name: "aaa-1", MatchType: "present"},
					{Name: "aaa-2", MatchType: "present"},
				},
			}, {
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/"},
				HeaderMatchConditions: []dag.HeaderMatchCondition{
					{Name: "zzz-1", MatchType: "present"},
					{Name: "zzz-2", MatchType: "present"},
				},
			}},
		},

		// If we have multiple conditions on the same header, ensure
		// that we order on the match type too.
		"headers order by match type": {
			routes: []*dag.Route{{
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/"},
			}, {
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/"},
				HeaderMatchConditions: []dag.HeaderMatchCondition{
					{Name: "x-request-1", MatchType: "present"},
					{Name: "x-request-2", MatchType: "present", Invert: true},
					{Name: "x-request-1", MatchType: "exact", Value: "foo"},
				},
			}},
			want: []*dag.Route{{
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/"},
				HeaderMatchConditions: []dag.HeaderMatchCondition{
					{Name: "x-request-1", MatchType: "exact", Value: "foo"},
					{Name: "x-request-1", MatchType: "present"},
					{Name: "x-request-2", MatchType: "present", Invert: true},
				},
			}, {
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/"},
			}},
		},

		// Verify that we always order the headers, even if
		// we don't need to compare the header conditions to
		// order multiple routes with the same prefix.
		"headers order in single route": {
			routes: []*dag.Route{{
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/"},
				HeaderMatchConditions: []dag.HeaderMatchCondition{
					{Name: "x-request-1", MatchType: "present"},
					{Name: "x-request-2", MatchType: "present", Invert: true},
					{Name: "x-request-1", MatchType: "exact", Value: "foo"},
				},
			}},
			want: []*dag.Route{{
				PathMatchCondition: &dag.PrefixMatchCondition{Prefix: "/"},
				HeaderMatchConditions: []dag.HeaderMatchCondition{
					{Name: "x-request-1", MatchType: "exact", Value: "foo"},
					{Name: "x-request-1", MatchType: "present"},
					{Name: "x-request-2", MatchType: "present", Invert: true},
				},
			}},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := append([]*dag.Route{}, tc.routes...) // shallow copy
			sortRoutes(got)
			assert.Equal(t, tc.want, got)
		})
	}
}
