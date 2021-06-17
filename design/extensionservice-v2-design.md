# Background

`ExtensionService` was added as a `v1alpha1` Contour custom resource definition (CRD) as part of the [External Authorization feature](https://github.com/projectcontour/contour/blob/main/design/external-authorization-design.md).
It provides a way to identify an [xDS gRPC service](https://www.envoyproxy.io/docs/envoy/latest/api-v3/service/service) and program Envoy to use it.
`ExtensionService` was subsequently used to implement [global rate limiting](https://github.com/projectcontour/contour/blob/main/design/ratelimit-design.md) and as part of the design for both [access logging](https://github.com/projectcontour/contour/blob/84f8223a692d2125b61f5dfe4bfdb321f10c2613/design/als-design.md) and [tracing](https://github.com/projectcontour/contour/blob/8a1fed2b3f22118401fc75aed26355ee195730dd/design/tracing-design.md).

Each use/proposed use of `ExtensionService` has been different.
For example, external auth supports an auth server `ExtensionService` per root `HTTPProxy` (but only for TLS-enabled proxies), while global rate limiting requires a single global rate limit `ExtensionService`.
Access logging requires the Envoy cluster for the access log `ExtensionService` to be statically defined in the bootstrap config.
And tracing does not use an `xDS gRPC service` at all.

Additionally, one of the key initial motivations for defining a new `ExtensionService` CRD was to enable writing status information to it, in order to provide feedback to the user about the health of the service.
However, to date this has not been designed or implemented.

This design document highlights the current limitations and shortcomings of the initial `ExtensionService` design, and proposes several options for the next iteration of the design, based on what we've learned from each use of it to date. 

# Current State
An `ExtensionService` identifies 1+ Kubernetes services to be used to program an Envoy cluster to be used as `xDS gRPC service`.
It includes settings for upstream validation, protocol, load balancing policy, timeout policy, and xDS protocol version.

The `ExtensionServiceSpec` is defined as:
```go
// ExtensionServiceSpec defines the desired state of an ExtensionService resource.
type ExtensionServiceSpec struct {
	// Services specifies the set of Kubernetes Service resources that
	// receive GRPC extension API requests.
	// If no weights are specified for any of the entries in
	// this array, traffic will be spread evenly across all the
	// services.
	// Otherwise, traffic is balanced proportionally to the
	// Weight field in each entry.
	//
	// +required
	// +kubebuilder:validation:MinItems=1
	Services []ExtensionServiceTarget `json:"services"`

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
```

Per the original design document:
> There are a number of benefits to creating a CRD to represent a supporting service:
> - The CRD directly generates an Envoy Cluster that Contour can use for any purpose. This means that with a single new API, Contour can add support for authorization, rate limiting and logging support services.
> - A CRD gives Contour a way to communicate the operational status of the support service, which was a desired goal.
> - A CRD allows the team operating the support service and the team operating Contour to collaborate more loosely.

The following subsections provide additional detail on each use/proposed use of `ExtensionService`.

## External Auth
Each TLS-enabled root `HTTPProxy` can optionally use a different external auth `ExtensionService` by defining an `authorization` block in the `virtualhost` definition.
It includes:
    - `extensionRef` (namespace/name of the `ExtensionService`)
    - `authPolicy` (includes `disabled`, `context` map)
    - `responseTimeout` (overrides the `ExtensionService's` `spec.timeoutPolicy.response` if specified)
    - `failOpen`

## Rate Limiting
Contour optionally supports *one* global rate limit `ExtensionService` by defining the `rateLimitService` block in the Contour config file.
It includes:
    - `extensionService` (namespace/name of the `ExtensionService`)
    - `domain`
    - `failOpen`
    - `enableXRateLimitHeaders`

## Access Logging
The proposed design for access logging allows one access log `ExtensionService`.
The access log Envoy cluster must be defined statically, as part of the bootstrap config.
Because of this, the design proposes using an annotation on the `ExtensionService` to identify it as an access log service, so `contour bootstrap` can find it and program a static cluster for it.
This avoids having to specify it in configuration for both (a) `contour bootstrap` and (b) `contour serve`.

## Tracing
The proposed design for tracing allows one tracing configuration, defined as part of the Contour config file.
The initial design did not propose using `ExtensionService` due to its limitations.
In particular, the tracing backends that we are planning to support are not xDS gRPC services; Zipkin supports JSON over HTTP, while OpenCensus only supports the Google gRPC client (which doesn't accept an Envoy cluster as a target).
Zipkin does expect an Envoy cluster, while OpenCensus does not.
Additionally, OpenCensus can only be configured once per Envoy lifetime; attempts to modify the configuration will result in Envoy refusing to accept new config until it's restarted.

# Issues/Limitations
The following is a summary of the issues & limitations with the current ExtensionService implementation:

- Many uses of ExtensionService hang off the HTTP Connection Manager (HCM). Each TLS vhost has its own HCM, but all non-TLS vhosts share the same HCM. So, all non-TLS vhosts must share a single ExtensionService while TLS vhosts can potentially each use their own. 
    - https://github.com/envoyproxy/envoy/issues/8853
- ExtensionService correlates to an xDS gRPC service; what about things that are not in there? Should they be supported?
    - non-gRPC
    - gRPC but can’t be accessed via Envoy gRPC client
    - etc
- ExtensionService is generic and can't store type-specific configuration. We’ve got an AuthorizationServer type, a RateLimitService type, etc. that wraps an ExtensionService ref and adds type-specific info, at the place where they're used. This means the config is split across two places and if e.g. an ExtAuth ExtensionService is used for many vhosts, config has to be repeated across them.
- How about ExtensionServices that are deployed as Envoy sidecars? Current idea is to use an ExternalName=localhost service (though right now the code actually blocks this). Alternately could allow ExtensionService to specify an address instead of a Kube cluster
- What about ExtensionServices that don’t run in Kubernetes? Is this an important use case? Could similarly just use an address field as an alternative to Kubernetes services. 
- Access log service requires the cluster to be defined in bootstrap config as a static cluster
    - https://github.com/envoyproxy/envoy/issues/3660 indicates this is a general policy, though with exceptions (e.g. ExtAuth)
- A big part of the original motivation for ExtensionService was to be able to capture status. How can we do this?
- for globals (e.g. RateLimit), since it's referenced via config file, the ExtensionService must exist before Contour starts up, and the reference can't be changed without a restart of Contour. If we used watchers/a controller, changes could be more dynamic.
- Is it actually necessary to support multiple Kubernetes services per ExtensionService? Since ExtAuth can only reference a single cluster (unlike a Route which can reference multiple weighted clusters), supporting this necessitated building a whole new DAG structure and ClusterLoadAssignment logic where a single cluster is built corresponding to multiple services' endpoints.
- How do other Envoy control planes do this?

# Alternate Approaches

The big design questions to answer are:
- Do we want to enforce consistency across the different extension service implementations?
    - Better UX for Contour users - plugging in external infra-level services is similar
    - It's not 100% consistent in Envoy, so requires papering over some differences
- Should contextual configuration about a particular service be part of the CR that represents it, or part of the HTTPProxy/Contour config file where it's referenced?
    - If the services are global, then it doesn't really matter either way; they're singletons
    - If services are per-vhost, then there's an argument for having settings at the reference site
    - Could have default settings on the service itself, overridden by vhost settings
- Should ExtensionService be able to represent non-xDS gRPC services?
    - From a UX perspective it would make sense


## No ExtensionService - inline the necessary fields wherever one of these is needed
- AuthorizationServer would inline []services, upstream validation, protocol, etc
- RateLimitService would do the same
- They would all look similar but could be customized according to the context in which they're used
- downside: wouldn't necessarily have a place to write status about the service
- downside: if used in many places, would have to repeat a lot of info (really only applies to ExtAuth at this point)
- downside: tracing would potentially require config in two places (for `bootstrap` and `serve`)

## ExtAuthService, RateLimitService, TraceService, etc.
- a number of different CRDs, each context-specific
- would have some common fields (could reuse types), but also context-specific fields
- would still have a place to write status
- could still be defined once and reused (if necessary)
- downside: anything global would be a singleton; high CRD-to-CR ratio

## Use a union type with a type discriminator and one non-nil subtype
e.g.
```yaml
kind: ExtensionService
spec:
  # This defines the specific type of ExtensionService. Alternately,
  # the presence of a particular sub-field could be used to identify
  # the type, but this is more explicit.
  type: [ExtAuthService|RateLimitService|AccessLogService]
  
  # There may be fields that are common to all types of ExtensionServices.
  <maybe some common fields>
  
  # ExtAuthz HTTP filter.
  externalAuth: {...}

  # Global rate limiting filter.
  rateLimiting: {...}

  # Access logging.
  accessLogging: {...}
```

- All the benefits of a unique type per-service, but don't have as many CRDs.
- Perhaps less user-friendly than a specific CRD per service?


## All global, or all per-vhost?
- ExtAuth is per-vhost, but *only* supported on TLS vhosts
- RateLimiting is global
- Tracing is global
- Access Logging is global

- If Contour is truly multi-tenant, then you'd sort of want per-vhost settings
- But *Envoy* is not necessarily truly multi-tenant and I don't think we want to be in the business of wrapping it in proxies to support multi-tenancy

## Sidecars/non-Kubernetes services
- We could add more than one "kind" of ExtensionService. The current "kind" is KubernetesService, but could also have ExternalName.

## Identifying globals/singletons
- Could use a controller that identifies the oldest (?) instance of a given type, e.g. the oldest RateLimitService, and programs that. Changes would trigger DAG rebuilds. That way the service doesn't need to exist at startup.
- TODO: how to deal with multiple Contours per cluster, how to identify the right ExtensionService?
    - could use a similar approach to what's done for HTTPProxy, where an ingress class must be specified for the Contour and on each HTTPProxy that it should process.

## Status
One of the initial stated goals was to have status information about the extension service.
There are many possible ways to source some information about the status of the service:
- existence of the referenced Kubernetes service
- existence of endpoints for the Kubernetes service
- Envoy cluster statistics (https://www.envoyproxy.io/docs/envoy/latest/configuration/upstream/cluster_manager/cluster_stats#general)
- Envoy health checks (https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/upstream/health_checking, https://www.envoyproxy.io/docs/envoy/latest/configuration/upstream/cluster_manager/cluster_stats#health-check-statistics)
- Envoy statistics specific to the type of ExtensionService
    - e.g. https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/rate_limit_filter#statistics
    - e.g. https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_authz_filter#statistics

