/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1_test

import (
	"context"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/test/dummies"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var errString = func(err error) string { return err.Error() }

func BeErrorAndMatchRegexp(regexp string, args ...interface{}) types.GomegaMatcher {
	return SatisfyAll(HaveOccurred(), WithTransform(errString, MatchRegexp(regexp, args)))
}

var _ = Describe("CloudStackCluster webhooks", func() {
	var ctx context.Context
	forbiddenRegex := "admission webhook.*denied the request.*Forbidden\\: %s"
	requiredRegex := "admission webhook.*denied the request.*Required value\\: %s"

	BeforeEach(func() { // Reset test vars to initial state.
		ctx = context.Background()
		dummies.SetDummyVars()                       // Reset cluster var.
		_ = k8sClient.Delete(ctx, dummies.CSCluster) // Delete any remnants.
		dummies.SetDummyVars()                       // Reset again since the k8s client can set this on delete.
	})

	Context("When creating a CloudStackCluster", func() {
		It("Should accept a CloudStackCluster with all attributes present", func() {
			Ω(k8sClient.Create(ctx, dummies.CSCluster)).Should(Succeed())
		})

		It("Should reject a CloudStackCluster with missing Zones.Network attribute", func() {
			dummies.CSCluster.Spec.Zones = []infrav1.Zone{{Name: "ZoneWNoNetwork"}}
			Ω(k8sClient.Create(ctx, dummies.CSCluster).Error()).Should(
				MatchRegexp(requiredRegex, "each Zone requires a Network specification"))
		})

		It("Should reject a CloudStackCluster with missing Zones attribute", func() {
			dummies.CSCluster.Spec.Zones = []infrav1.Zone{}
			Ω(k8sClient.Create(ctx, dummies.CSCluster).Error()).Should(MatchRegexp(requiredRegex, "Zones"))
		})

		It("Should reject a CloudStackCluster with IdentityRef not of kind 'Secret'", func() {
			dummies.CSCluster.Spec.IdentityRef.Kind = "ConfigMap"
			Ω(k8sClient.Create(ctx, dummies.CSCluster).Error()).Should(MatchRegexp(forbiddenRegex, "must be a Secret"))
		})
	})

	Context("When updating a CloudStackCluster", func() {
		BeforeEach(func() {
			Ω(k8sClient.Create(ctx, dummies.CSCluster)).Should(Succeed())
		})

		It("Should reject updates to CloudStackCluster Zones", func() {
			dummies.CSCluster.Spec.Zones = []infrav1.Zone{dummies.Zone1}
			Ω(k8sClient.Update(ctx, dummies.CSCluster)).Should(BeErrorAndMatchRegexp(forbiddenRegex, "Zones and sub"))
		})
		It("Should reject updates to CloudStackCluster Zones", func() {
			dummies.CSCluster.Spec.Zones[0].Network.Name = "ArbitraryUpdateNetworkName"
			Ω(k8sClient.Update(ctx, dummies.CSCluster)).Should(BeErrorAndMatchRegexp(forbiddenRegex, "Zones and sub"))
		})
		It("Should reject updates to CloudStackCluster controlplaneendpoint.host", func() {
			dummies.CSCluster.Spec.ControlPlaneEndpoint.Host = "1.1.1.1"
			Ω(k8sClient.Update(ctx, dummies.CSCluster)).
				Should(BeErrorAndMatchRegexp(forbiddenRegex, "controlplaneendpoint\\.host"))
		})

		It("Should reject updates to CloudStackCluster controlplaneendpoint.port", func() {
			dummies.CSCluster.Spec.ControlPlaneEndpoint.Port = int32(1234)
			Ω(k8sClient.Update(ctx, dummies.CSCluster)).
				Should(BeErrorAndMatchRegexp(forbiddenRegex, "controlplaneendpoint\\.port"))
		})
		It("Should reject updates to the CloudStackCluster identity reference kind", func() {
			dummies.CSCluster.Spec.IdentityRef.Kind = "ConfigMap"
			Ω(k8sClient.Update(ctx, dummies.CSCluster)).
				Should(BeErrorAndMatchRegexp(forbiddenRegex, "identityRef\\.Kind"))
		})
		It("Should reject updates to the CloudStackCluster identity reference name", func() {
			dummies.CSCluster.Spec.IdentityRef.Name = "ConfigMap"
			Ω(k8sClient.Update(ctx, dummies.CSCluster)).
				Should(BeErrorAndMatchRegexp(forbiddenRegex, "identityRef\\.name"))
		})
	})
})
