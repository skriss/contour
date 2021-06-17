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

package v1alpha1

import (
	contour_api_v1 "github.com/projectcontour/contour/apis/projectcontour/v1"
)

// Target defines what backend to connect a given ExtensionService to.
// Exactly one field should be populated: either a set of weighted Kubernetes
// services, or a specific address.
type Target struct {
	// Services specifies the set of Kubernetes Service resources that
	// receive GRPC extension API requests.
	// If no weights are specified for any of the entries in
	// this array, traffic will be spread evenly across all the
	// services.
	// Otherwise, traffic is balanced proportionally to the
	// Weight field in each entry.
	//
	// +optional
	// +kubebuilder:validation:MinItems=1
	Services []ExtensionServiceTarget `json:"services"`

	// Address defines a specific address to use.
	Address string `json:"address"`
}

type ExternalAuthServiceSpec struct {
	// Target defines the backend to send external auth requests to.
	//
	// +required
	Target Target `json:"target"`

	// AuthPolicy sets a default authorization policy for client requests.
	// This policy will be used unless overridden by individual routes.
	//
	// +optional
	AuthPolicy *contour_api_v1.AuthorizationPolicy `json:"authPolicy,omitempty"`

	// ResponseTimeout configures maximum time to wait for a check response from the authorization server.
	// Timeout durations are expressed in the Go [Duration format](https://godoc.org/time#ParseDuration).
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
	// The string "infinity" is also a valid input and specifies no timeout.
	//
	// +optional
	// +kubebuilder:validation:Pattern=`^(((\d*(\.\d*)?h)|(\d*(\.\d*)?m)|(\d*(\.\d*)?s)|(\d*(\.\d*)?ms)|(\d*(\.\d*)?us)|(\d*(\.\d*)?µs)|(\d*(\.\d*)?ns))+|infinity|infinite)$`
	ResponseTimeout string `json:"responseTimeout,omitempty"`

	// If FailOpen is true, the client request is forwarded to the upstream service
	// even if the authorization server fails to respond. This field should not be
	// set in most cases. It is intended for use only while migrating applications
	// from internal authorization to Contour external authorization.
	//
	// +optional
	FailOpen bool `json:"failOpen,omitempty"`

	// UpstreamValidation defines how to verify the backend service's certificate
	// +optional
	UpstreamValidation *contour_api_v1.UpstreamValidation `json:"validation,omitempty"`

	// Protocol may be used to specify (or override) the protocol used to reach this Service.
	// Values may be h2 or h2c. If omitted, protocol-selection falls back on Service annotations.
	//
	// +optional
	// +kubebuilder:validation:Enum=h2;h2c
	Protocol *string `json:"protocol,omitempty"`

	// The policy for load balancing GRPC service requests. Note that the
	// `Cookie` and `RequestHash` load balancing strategies cannot be used
	// here.
	//
	// +optional
	LoadBalancerPolicy *contour_api_v1.LoadBalancerPolicy `json:"loadBalancerPolicy,omitempty"`

	// The timeout policy for requests to the services.
	//
	// +optional
	TimeoutPolicy *contour_api_v1.TimeoutPolicy `json:"timeoutPolicy,omitempty"`

	// This field sets the version of the GRPC protocol that Envoy uses to
	// send requests to the extension service. Since Contour always uses the
	// v3 Envoy API, this is currently fixed at "v3". However, other
	// protocol options will be available in future.
	//
	// +optional
	// +kubebuilder:validation:Enum=v3
	ProtocolVersion ExtensionProtocolVersion `json:"protocolVersion,omitempty"`
}

type RateLimitServiceSpec struct {
	// Target defines the backend to send external auth requests to.
	//
	// +required
	Target Target `json:"target"`

	// Domain is passed to the Rate Limit Service.
	Domain string `yaml:"domain,omitempty"`

	// FailOpen defines whether to allow requests to proceed when the
	// Rate Limit Service fails to respond with a valid rate limit
	// decision within the timeout defined on the extension service.
	FailOpen bool `yaml:"failOpen,omitempty"`

	// EnableXRateLimitHeaders defines whether to include the X-RateLimit
	// headers X-RateLimit-Limit, X-RateLimit-Remaining, and X-RateLimit-Reset
	// (as defined by the IETF Internet-Draft linked below), on responses
	// to clients when the Rate Limit Service is consulted for a request.
	//
	// ref. https://tools.ietf.org/id/draft-polli-ratelimit-headers-03.html
	EnableXRateLimitHeaders bool `yaml:"enableXRateLimitHeaders,omitempty"`

	// UpstreamValidation defines how to verify the backend service's certificate
	// +optional
	UpstreamValidation *contour_api_v1.UpstreamValidation `json:"validation,omitempty"`

	// Protocol may be used to specify (or override) the protocol used to reach this Service.
	// Values may be h2 or h2c. If omitted, protocol-selection falls back on Service annotations.
	//
	// +optional
	// +kubebuilder:validation:Enum=h2;h2c
	Protocol *string `json:"protocol,omitempty"`

	// The policy for load balancing GRPC service requests. Note that the
	// `Cookie` and `RequestHash` load balancing strategies cannot be used
	// here.
	//
	// +optional
	LoadBalancerPolicy *contour_api_v1.LoadBalancerPolicy `json:"loadBalancerPolicy,omitempty"`

	// The timeout policy for requests to the services.
	//
	// +optional
	TimeoutPolicy *contour_api_v1.TimeoutPolicy `json:"timeoutPolicy,omitempty"`

	// This field sets the version of the GRPC protocol that Envoy uses to
	// send requests to the extension service. Since Contour always uses the
	// v3 Envoy API, this is currently fixed at "v3". However, other
	// protocol options will be available in future.
	//
	// +optional
	// +kubebuilder:validation:Enum=v3
	ProtocolVersion ExtensionProtocolVersion `json:"protocolVersion,omitempty"`
}

type AccessLoggingServiceSpec struct {
	// Target defines the backend to send external auth requests to.
	//
	// +required
	Target Target `json:"target"`

	// UpstreamValidation defines how to verify the backend service's certificate
	// +optional
	UpstreamValidation *contour_api_v1.UpstreamValidation `json:"validation,omitempty"`

	// Protocol may be used to specify (or override) the protocol used to reach this Service.
	// Values may be h2 or h2c. If omitted, protocol-selection falls back on Service annotations.
	//
	// +optional
	// +kubebuilder:validation:Enum=h2;h2c
	Protocol *string `json:"protocol,omitempty"`

	// The policy for load balancing GRPC service requests. Note that the
	// `Cookie` and `RequestHash` load balancing strategies cannot be used
	// here.
	//
	// +optional
	LoadBalancerPolicy *contour_api_v1.LoadBalancerPolicy `json:"loadBalancerPolicy,omitempty"`

	// The timeout policy for requests to the services.
	//
	// +optional
	TimeoutPolicy *contour_api_v1.TimeoutPolicy `json:"timeoutPolicy,omitempty"`

	// This field sets the version of the GRPC protocol that Envoy uses to
	// send requests to the extension service. Since Contour always uses the
	// v3 Envoy API, this is currently fixed at "v3". However, other
	// protocol options will be available in future.
	//
	// +optional
	// +kubebuilder:validation:Enum=v3
	ProtocolVersion ExtensionProtocolVersion `json:"protocolVersion,omitempty"`
}

type TracingServiceSpec struct {
}
