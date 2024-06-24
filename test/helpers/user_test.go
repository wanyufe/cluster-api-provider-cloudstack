/*
Copyright 2023 The Kubernetes Authors.

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

package helpers_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	"sigs.k8s.io/cluster-api-provider-cloudstack/test/helpers"
)

var _ = Describe("Test helper methods", func() {
	csClient, err := helpers.NewCSClient()
	Ω(err).ShouldNot(HaveOccurred())

	// Get the root domain's ID.
	rootDomainID, err, found := helpers.GetDomainByPath(csClient, "ROOT/")
	Ω(err).ShouldNot(HaveOccurred())
	Ω(rootDomainID).ShouldNot(BeEmpty())
	Ω(found).Should(BeTrue())

	AfterEach(func() {
		for _, path := range []string{"ROOT/someNewDomain", "ROOT/blah"} {
			// Delete any created domains.
			id, err, found := helpers.GetDomainByPath(csClient, path)
			Ω(err).ShouldNot(HaveOccurred())
			if found {
				Ω(helpers.DeleteDomain(csClient, id)).Should(Succeed())
			}
		}
	})

	Context("Domain Creation and Deletion.", func() {
		It("Can get the ROOT domain's ID.", func() {
			id, err, found := helpers.GetDomainByPath(csClient, "ROOT/")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(id).ShouldNot(BeEmpty())
			Ω(found).Should(BeTrue())
		})

		It("Doesn't error when unable to get a domain's ID.", func() {
			id, err, found := helpers.GetDomainByPath(csClient, "ROOT/blahnotpresent")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(found).Should(BeFalse())
			Ω(id).Should(BeEmpty())
		})

		It("Can create a domain under a parent domain.", func() {
			id, err := helpers.CreateDomainUnderParent(csClient, rootDomainID, "someNewDomain")
			Ω(id).ShouldNot(BeEmpty())
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("Returns an appropriate error when the domain already exists.", func() {
			someDomain := &cloud.Domain{Name: "blah", Path: "blah"}
			Ω(helpers.GetOrCreateDomain(csClient, someDomain)).Should(Succeed())
			Ω(someDomain.Name).Should(Equal("blah"))
			Ω(someDomain.Path).Should(Equal("ROOT/blah"))
			Ω(someDomain.ID).ShouldNot(BeEmpty())
			_, err = helpers.CreateDomainUnderParent(csClient, rootDomainID, "blah")
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(ContainSubstring("already exists"))
		})

		It("Doesn't error if the domain already exists.", func() {
			someDomain := &cloud.Domain{Name: "blah", Path: "blah"}
			Ω(helpers.GetOrCreateDomain(csClient, someDomain)).Should(Succeed())
			Ω(someDomain.Name).Should(Equal("blah"))
			Ω(someDomain.Path).Should(Equal("ROOT/blah"))
			Ω(someDomain.ID).ShouldNot(BeEmpty())

			Ω(helpers.GetOrCreateDomain(csClient, someDomain)).Should(Succeed())
			Ω(someDomain.Name).Should(Equal("blah"))
			Ω(someDomain.Path).Should(Equal("ROOT/blah"))
			Ω(someDomain.ID).ShouldNot(BeEmpty())
		})

		It("Can create a wholly new multi-level sub-domain path.", func() {
			someDomain := &cloud.Domain{Name: "tooBlah", Path: "ROOT/someNewDomain/tooBlah"}
			Ω(helpers.GetOrCreateDomain(csClient, someDomain)).Should(Succeed())
			Ω(someDomain.Name).Should(Equal("tooBlah"))
			Ω(someDomain.Path).Should(Equal("ROOT/someNewDomain/tooBlah"))
			Ω(someDomain.ID).ShouldNot(BeEmpty())
		})
	})

	Context("Account Creation.", func() {
		It("Can create a new account in a new domain.", func() {
			domain := cloud.Domain{Path: "ROOT/someNewDomain/tooBlah"}
			account := cloud.Account{Name: "TempTestAccount", Domain: domain}
			Ω(helpers.GetOrCreateAccount(csClient, &account)).Should(Succeed())
		})
		// already exists
		It("Doesn't fail if the account already exists.", func() {
			domain := cloud.Domain{Path: "ROOT/someNewDomain/tooBlah"}
			account := cloud.Account{Name: "TempTestAccount", Domain: domain}
			Ω(helpers.GetOrCreateAccount(csClient, &account)).Should(Succeed())
			Ω(helpers.GetOrCreateAccount(csClient, &account)).Should(Succeed())
		})
	})

	Context("User Creation w/Keys.", func() {
		It("Can create a new user with keys.", func() {
			domain := cloud.Domain{Path: "ROOT/someNewDomain/tooBlah"}
			account := cloud.Account{Name: "TempTestAccount", Domain: domain}
			user := cloud.User{Account: account}
			Ω(helpers.GetOrCreateUserWithKey(csClient, &user)).Should(Succeed())
			Ω(user.ID).ShouldNot(BeEmpty())
			Ω(user.APIKey).ShouldNot(BeEmpty())
		})
	})
})
