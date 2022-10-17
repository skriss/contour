package dag

import (
	"fmt"
	"strings"

	envoy_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_hcm_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	envoy_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/ghodss/yaml"
	"github.com/projectcontour/contour/internal/protobuf"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	v1 "k8s.io/api/core/v1"
)

type EnvoyConfigProcessor struct {
	logrus.FieldLogger
}

var _ Processor = &EnvoyConfigProcessor{}

func (p *EnvoyConfigProcessor) Run(dag *DAG, cache *KubernetesCache) {
	if cache.configmap == nil {
		p.Info("configmap projectcontour/envoyconfig not found")
		return
	}

	// Listeners
	listenersYAML, ok := cache.configmap.Data["listeners.yaml"]
	if !ok || len(listenersYAML) == 0 {
		p.Info("no listeners config found")
		return
	}

	listeners, err := decodeYAML(listenersYAML, func() *envoy_listener_v3.Listener { return &envoy_listener_v3.Listener{} })
	if err != nil {
		p.WithError(err).Error("error decoding listeners")
		return
	}

	// Set config sources
	for _, listener := range listeners {
		for _, fc := range listener.FilterChains {
			if fc.TransportSocket != nil {
				tls := &envoy_tls_v3.DownstreamTlsContext{}
				if err := fc.TransportSocket.GetTypedConfig().UnmarshalTo(tls); err != nil {
					p.WithError(err).Error("error unmarshalling TLS transport socket config")
				}

				for _, sds := range tls.CommonTlsContext.TlsCertificateSdsSecretConfigs {
					if sds.SdsConfig == nil {
						sds.SdsConfig = &envoy_core_v3.ConfigSource{
							ResourceApiVersion: envoy_core_v3.ApiVersion_V3,
							ConfigSourceSpecifier: &envoy_core_v3.ConfigSource_ApiConfigSource{
								ApiConfigSource: &envoy_core_v3.ApiConfigSource{
									ApiType:             envoy_core_v3.ApiConfigSource_GRPC,
									TransportApiVersion: envoy_core_v3.ApiVersion_V3,
									GrpcServices: []*envoy_core_v3.GrpcService{
										{
											TargetSpecifier: &envoy_core_v3.GrpcService_EnvoyGrpc_{
												EnvoyGrpc: &envoy_core_v3.GrpcService_EnvoyGrpc{
													ClusterName: "contour",
													Authority:   "contour",
												},
											},
										},
									},
								},
							},
						}
					}
				}

				fc.TransportSocket.ConfigType = &envoy_core_v3.TransportSocket_TypedConfig{
					TypedConfig: protobuf.MustMarshalAny(tls),
				}
			}

			for _, f := range fc.Filters {
				// TODO type-switch
				if f.Name == "envoy.filters.network.http_connection_manager" {
					hcm := &envoy_hcm_v3.HttpConnectionManager{}
					if err := f.GetTypedConfig().UnmarshalTo(hcm); err != nil {
						p.WithError(err).Error("error unmarshalling HCM config")
					}

					rds := hcm.GetRds()
					rds.ConfigSource = &envoy_core_v3.ConfigSource{
						ResourceApiVersion: envoy_core_v3.ApiVersion_V3,
						ConfigSourceSpecifier: &envoy_core_v3.ConfigSource_ApiConfigSource{
							ApiConfigSource: &envoy_core_v3.ApiConfigSource{
								ApiType:             envoy_core_v3.ApiConfigSource_GRPC,
								TransportApiVersion: envoy_core_v3.ApiVersion_V3,
								GrpcServices: []*envoy_core_v3.GrpcService{
									{
										TargetSpecifier: &envoy_core_v3.GrpcService_EnvoyGrpc_{
											EnvoyGrpc: &envoy_core_v3.GrpcService_EnvoyGrpc{
												ClusterName: "contour",
												Authority:   "contour",
											},
										},
									},
								},
							},
						},
					}

					f.ConfigType = &envoy_listener_v3.Filter_TypedConfig{
						TypedConfig: protobuf.MustMarshalAny(hcm),
					}
				}
			}
		}
	}

	dag.DynamicListeners = listeners

	p.Infof("Listeners: %d", len(dag.DynamicListeners))

	// Clusters
	clustersYAML, ok := cache.configmap.Data["clusters.yaml"]
	if !ok || len(clustersYAML) == 0 {
		p.Info("no clusters config found")
		return
	}

	clusters, err := decodeYAML(clustersYAML, func() *envoy_cluster_v3.Cluster { return &envoy_cluster_v3.Cluster{} })
	if err != nil {
		p.WithError(err).Error("error decoding clusters")
		return
	}

	dag.DynamicClusters = clusters

	for nsName, service := range cache.services {
		if nsName.Namespace != "default" {
			continue
		}

		for _, port := range service.Spec.Ports {
			c := &envoy_cluster_v3.Cluster{
				Name: fmt.Sprintf("%s.%s.%d", nsName.Name, nsName.Namespace, port.Port),
				ClusterDiscoveryType: &envoy_cluster_v3.Cluster_Type{
					Type: envoy_cluster_v3.Cluster_STRICT_DNS,
				},
				LbPolicy: envoy_cluster_v3.Cluster_ROUND_ROBIN,
				LoadAssignment: &endpointv3.ClusterLoadAssignment{
					ClusterName: fmt.Sprintf("%s.%s.%d", nsName.Name, nsName.Namespace, port.Port),
					Endpoints: []*endpointv3.LocalityLbEndpoints{
						{
							LbEndpoints: []*endpointv3.LbEndpoint{
								{
									HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
										Endpoint: &endpointv3.Endpoint{
											Address: &envoy_core_v3.Address{
												Address: &envoy_core_v3.Address_SocketAddress{
													SocketAddress: &envoy_core_v3.SocketAddress{
														Protocol: envoy_core_v3.SocketAddress_TCP,
														Address:  fmt.Sprintf("%s.%s", nsName.Name, nsName.Namespace),
														PortSpecifier: &envoy_core_v3.SocketAddress_PortValue{
															PortValue: uint32(port.Port),
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			dag.DynamicClusters = append(dag.DynamicClusters, c)
		}
	}
	p.Infof("Clusters: %d", len(dag.DynamicClusters))

	// RouteConfigs
	routeConfigsYAML, ok := cache.configmap.Data["routeconfigs.yaml"]
	if !ok || len(routeConfigsYAML) == 0 {
		p.Info("no routeconfigs config found")
		return
	}

	routeConfigs, err := decodeYAML(routeConfigsYAML, func() *envoy_route_v3.RouteConfiguration { return &envoy_route_v3.RouteConfiguration{} })
	if err != nil {
		p.WithError(err).Error("error decoding routeconfigs")
		return
	}

	dag.DynamicRouteConfigs = routeConfigs

	p.Infof("RouteConfigurations: %d", len(dag.DynamicRouteConfigs))

	// Secrets
	for _, secret := range cache.secrets {
		if secret.Namespace != "default" {
			continue
		}
		if secret.Type != v1.SecretTypeTLS {
			continue
		}

		dag.DynamicSecrets = append(dag.DynamicSecrets, &envoy_tls_v3.Secret{
			Name: fmt.Sprintf("%s.%s", secret.Name, secret.Namespace),
			Type: &envoy_tls_v3.Secret_TlsCertificate{
				TlsCertificate: &envoy_tls_v3.TlsCertificate{
					PrivateKey: &envoy_core_v3.DataSource{
						Specifier: &envoy_core_v3.DataSource_InlineBytes{
							InlineBytes: secret.Data[v1.TLSPrivateKeyKey],
						},
					},
					CertificateChain: &envoy_core_v3.DataSource{
						Specifier: &envoy_core_v3.DataSource_InlineBytes{
							InlineBytes: secret.Data[v1.TLSCertKey],
						},
					},
				},
			},
		})
	}
	p.Infof("Secrets: %d", len(dag.DynamicSecrets))
}

func decodeYAML[T proto.Message](yamlString string, newObj func() T) ([]T, error) {
	var res []T

	for _, yamlDoc := range strings.Split(yamlString, "---") {
		jsonData, err := yaml.YAMLToJSON([]byte(yamlDoc))
		if err != nil {
			return nil, err
		}

		obj := newObj()
		if err := protojson.Unmarshal(jsonData, obj); err != nil {
			return nil, err
		}

		res = append(res, obj)
	}

	return res, nil
}
