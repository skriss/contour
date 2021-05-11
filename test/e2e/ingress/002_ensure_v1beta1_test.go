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

// +build e2e

package ingress

import (
	"context"

	. "github.com/onsi/ginkgo"
	"github.com/projectcontour/contour/test/e2e"
	"github.com/stretchr/testify/require"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("002-ingress-ensure-v1beta1", func() {
	It("ensures ingress v1beta1 works", func() {
		testIngressV1Beta1(f)
	})
})

// Explicitly ensure v1beta1 resources continue to work.
func testIngressV1Beta1(f *e2e.Framework) {
	namespace := "002-ingress-ensure-v1beta1"

	f.CreateNamespace(namespace)
	defer f.DeleteNamespace(namespace)

	f.Fixtures.Echo.Deploy(namespace, "ingress-conformance-echo")

	ingressHost := "v1beta1.projectcontour.io"
	i := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "echo",
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: ingressHost,
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Backend: v1beta1.IngressBackend{
										ServiceName: "ingress-conformance-echo",
										ServicePort: intstr.FromInt(80),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	require.NoError(f.T(), f.Client.Create(context.TODO(), i))

	res, ok := f.HTTP.RequestUntil(&e2e.HTTPRequestOpts{
		Host:      ingressHost,
		Path:      "/echo",
		Condition: e2e.HasStatusCode(200),
	})
	require.Truef(f.T(), ok, "expected 200 response code, got %d", res.StatusCode)
}
