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

package cloud_test

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"reflect"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
)

var _ = Describe("Helpers", func() {

	It("should compress and encode string", func() {
		str := "Hello World"

		compressedAndEncodedData, err := cloud.CompressAndEncodeString(str)

		compressedData, _ := base64.StdEncoding.DecodeString(compressedAndEncodedData)
		reader, _ := gzip.NewReader(bytes.NewReader(compressedData))
		result, _ := io.ReadAll(reader)

		Ω(err).Should(BeNil())
		Ω(string(result)).Should(Equal(str))
	})
})

// This matcher is used to make gomega matching compatible with gomock parameter matching.
// It's pretty awesome!
//
// This sort of hacks the gomock interface to inject a gomega matcher.
//
// Gomega matchers are far more flexible than gomock matchers, but they normally can't be used on parameters.

type paramMatcher struct {
	matcher types.GomegaMatcher
}

func ParamMatch(matcher types.GomegaMatcher) gomock.Matcher {
	return paramMatcher{matcher}
}

func (p paramMatcher) String() string {
	return "a gomega matcher to match, and said matcher should have panicked before this message was printed."
}

func (p paramMatcher) Matches(x interface{}) (retVal bool) {
	return Ω(x).Should(p.matcher)
}

// This generates translating matchers.
//
// The CloudStack Go API uses param interfaces that can't be accessed except through builtin methods.
//
// This generates translation matchers:
//
//	   Essentially it will generate a matcher that checks the value from p.Get<some field>() is Equal to an input String.
//
//			DomainIDEquals = FieldMatcherGenerator("GetDomainid")
//	     p := &CreateNewSomethingParams{Domainid: "FakeDomainID"}
//	     Ω(p).DomainIDEquals("FakeDomainID")
func FieldMatcherGenerator(fetchFunc string) func(string) types.GomegaMatcher {
	return func(expected string) types.GomegaMatcher {
		return WithTransform(
			func(x interface{}) string {
				meth := reflect.ValueOf(x).MethodByName(fetchFunc)
				fmt.Println(meth.Call(nil)[0])

				return meth.Call(nil)[0].String()
			}, Equal(expected))
	}
}

var (
	DomainIDEquals = FieldMatcherGenerator("GetDomainid")
	AccountEquals  = FieldMatcherGenerator("GetAccount")
	IDEquals       = FieldMatcherGenerator("GetId")
	NameEquals     = FieldMatcherGenerator("GetName")
)
