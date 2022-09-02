# JWT Verification Support

## Abstract
This document describes a design for performing JSON Web Token (JWT) verification for requests to virtual hosts hosted by Contour.

## Background
JSON Web Token (JWT) is an open standard (RFC 7519) that defines a compact and self-contained way for securely transmitting information between parties as a JSON object. 
This information can be verified and trusted because it is digitally signed. (ref. https://jwt.io/introduction)

Envoy Proxy has built-in support for verifying JWTs that are attached to incoming requests, via the [JWT Authentication HTTP filter](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/jwt_authn_filter#config-http-filters-jwt-authn). (Note, this document will use the term **JWT verification** to describe this process).
Specifically, Envoy can verify the signature, audience, issuer and time restrictions of a JWT.
If verification fails, the request will be rejected.
It's important to note that this filter does not itself obtain a JWT for an incoming request; the request must already have one attached.

Contour does not currently have support for configuring the JWT authentication filter.
This document proposes a design for adding that support to Contour's custom resource, `HTTPProxy`.

## Goals
- JWT verification for requests to TLS-enabled virtual hosts.
- Expose a subset of the Envoy filter configuration to cover the most common use cases.
- Be able to easily expose additional configuration settings if/when needed in the future.

## Non Goals
- JWT verification for requests to non-TLS enabled virtual hosts.
- Exposing all possible Envoy configuration settings.
- Supporting end-to-end OAuth2/OIDC flows.

## High-Level Design
Contour's `HTTPProxy` resource will get a new optional field, `spec.virtualhost.jwtVerificationPolicy`, to define the details of how to verify JWTs for requests to a given virtual host.
This field will only be supported for virtual hosts for which Envoy is terminating TLS.
The structure of this field will be similar to the [Envoy filter's configuration](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/http/jwt_authn/v3/config.proto#envoy-v3-api-msg-extensions-filters-http-jwt-authn-v3-jwtauthentication), with some simplifications.

Specifically, the `jwtVerificationPolicy` will define two key subfields:
- `providers` defines one or more sets of issuers, audiences, and JSON Web Key Sets (JWKS) that can be used to verify a JWT (see [the Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/http/jwt_authn/v3/config.proto#envoy-v3-api-msg-extensions-filters-http-jwt-authn-v3-jwtprovider) for more information).
- `rules` defines which routes are required to be verified by which providers (see [the Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/http/jwt_authn/v3/config.proto#envoy-v3-api-msg-extensions-filters-http-jwt-authn-v3-requirementrule) for more information).

Contour will validate the contents of `jwtVerificationPolicy` if present, and will configure the JWT authentication filter on the HTTP Connection Manager for the relevant virtual host.
Contour will also add a CDS cluster for the remote JWKS, as required by [the Envoy configuration](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/http/jwt_authn/v3/config.proto#envoy-v3-api-msg-extensions-filters-http-jwt-authn-v3-remotejwks).


## Detailed Design
The detailed structure of the new `jwtVerificationPolicy` field is shown via YAML below:

```yaml
apiVersion: projectcontour.io/v1
kind: HTTPProxy
metadata:
  name: jwt-verification
spec:
  virtualhost:
    fqdn: example.com
    tls:
      secretName: tls-cert
    jwtVerificationPolicy:
      providers:
        - 
          # name is a unique name for the provider.
          name: provider-1
          # issuer (optional) must match the "iss" field in the JWT.
          # If not specified, the "iss" field is not checked.
          issuer: foo.com
          # audiences (optional) allowlist for the "aud" field in the JWT.
          # If not specified, the "aud" field is not checked.
          audiences:
            - audience-1
            - audience-2
          # remoteJWKS is an HTTP endpoint that returns the JWKS
          # to use to verify the JWT signature.
          remoteJWKS:
            httpURI:
              uri: https://example.com/jwks.json
              timeout: 1s
            # cacheDuration is how long to cache the fetched JWKS
            # locally.
            cacheDuration: 5m
          # localJWKS can be used instead of remoteJWKS and defines
          # an in-cluster secret containing the JWKS for this provider.
          localJWKS:
            secretName: my-jwks
            key: jwks.json
      # rules define which routes are required to be verified by which
      # providers. They are matched to requests in the order in which
      # they are provided.
      rules:
        # This match excludes any requests with a path starting with /js
        # from JWT verification.
        - match:
            prefix: /js
        # This match requires all other requests to have a JWT that
        # can be verified by provider "provider-1" (must have issuer=foo.com,
        # audience of either "audience-1" or "audience-2", and signature
        # must be able to be verified using the JWKS at https://example.com/jwks.json).
        - match:
            prefix: /
          providerName: provider-1
  routes:
    # ...
```

It is worth highlighting some of the configuration options that Envoy's filter has, that Contour will *not* expose, at least not initially:
- non-default extract locations for the JWT (the default locations are (1) the `Authorization` header using the Bearer schema; and (2) the `access_token` query parameter, in that order).
- complex rule route matches (i.e. only path `prefix` matching will be supported initially)
- complex rule requirements (e.g. `requires_any`, `requires_all`)

Contour's API is structured such that these and other more complex/less common options may be added at a later date, if there is user demand for them.

A complete/valid HTTPProxy using JWT verification is shown below:

```yaml
apiVersion: projectcontour.io/v1
kind: HTTPProxy
metadata:
  name: jwt-verification-proxy
spec:
  virtualhost:
    fqdn: example.com
    tls:
      secretName: tls-cert
    jwtVerificationPolicy:
      providers:
        - name: provider-1
          issuer: example.com
          audiences:
            - audience-1
            - audience-2
          remoteJWKS:
            httpURI:
              uri: https://example.com/jwks.json
              timeout: 1s
            cacheDuration: 5m
      rules:
        - match:
            prefix: /js
        - match:
            prefix: /
          providerName: provider-1
  routes:
    - conditions:
      - prefix: /
      services:
      - name: s1
        port: 80
```


## Alternatives Considered
Envoy also has an [OAuth2 HTTP filter](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/oauth2_filter), which supports end-to-end OAuth2/OIDC flows.
This provides related but separate functionality to the JWT authentication filter.
This [excellent blog post](https://www.jpmorgan.com/technology/technology-blog/protecting-web-applications-via-envoy-oauth2-filter) shows an example of how to use the OAuth2 and JWT filters together in Envoy.
Contour [may pursue adding support for the OAuth2 filter](https://github.com/projectcontour/contour/issues/2664), but it will be designed and implemented separately.

## Security Considerations
If this proposal has an impact to the security of the product, its users, or data stored or transmitted via the product, they must be addressed here.

## Compatibility
JWT verification will be an optional feature that is disabled by default.
Existing users should not be affected by its addition.

## Implementation
A description of the implementation, timelines, and any resources that have agreed to contribute.

## Open Issues
A discussion of issues relating to this proposal for which the author does not know the solution. This section may be omitted if there are none.
