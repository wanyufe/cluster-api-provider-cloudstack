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

package utils

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capiControlPlanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	clientPkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// getMachineSetFromCAPIMachine attempts to fetch a MachineSet from CAPI machine owner reference.
func getMachineSetFromCAPIMachine(
	ctx context.Context,
	client clientPkg.Client,
	capiMachine *clusterv1.Machine,
) (*clusterv1.MachineSet, error) {

	ref := GetManagementOwnerRef(capiMachine)
	if ref == nil {
		return nil, errors.New("management owner not found")
	}
	gv, err := schema.ParseGroupVersion(ref.APIVersion)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if gv.Group == clusterv1.GroupVersion.Group {
		key := clientPkg.ObjectKey{
			Namespace: capiMachine.Namespace,
			Name:      ref.Name,
		}

		machineSet := &clusterv1.MachineSet{}
		if err := client.Get(ctx, key, machineSet); err != nil {
			return nil, err
		}

		return machineSet, nil
	}
	return nil, nil
}

// getKubeadmControlPlaneFromCAPIMachine attempts to fetch a KubeadmControlPlane from a CAPI machine owner reference.
func getKubeadmControlPlaneFromCAPIMachine(
	ctx context.Context,
	client clientPkg.Client,
	capiMachine *clusterv1.Machine,
) (*capiControlPlanev1.KubeadmControlPlane, error) {

	ref := GetManagementOwnerRef(capiMachine)
	if ref == nil {
		return nil, errors.New("management owner not found")
	}
	gv, err := schema.ParseGroupVersion(ref.APIVersion)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if gv.Group == capiControlPlanev1.GroupVersion.Group {
		key := clientPkg.ObjectKey{
			Namespace: capiMachine.Namespace,
			Name:      ref.Name,
		}

		controlPlane := &capiControlPlanev1.KubeadmControlPlane{}
		if err := client.Get(ctx, key, controlPlane); err != nil {
			return nil, err
		}

		return controlPlane, nil
	}
	return nil, nil
}

// IsOwnerDeleted returns a boolean if the owner of the CAPI machine has been deleted.
func IsOwnerDeleted(ctx context.Context, client clientPkg.Client, capiMachine *clusterv1.Machine) (bool, error) {
	if util.IsControlPlaneMachine(capiMachine) {
		// The controlplane sticks around after deletion pending the deletion of its machines.
		// As such, need to check the deletion timestamp thereof.
		if cp, err := getKubeadmControlPlaneFromCAPIMachine(ctx, client, capiMachine); cp != nil && cp.DeletionTimestamp == nil {
			return false, nil
		} else if err != nil && !strings.Contains(strings.ToLower(err.Error()), "not found") {
			return false, err
		}
	} else {
		// The machineset is deleted immediately, regardless of machine ownership.
		// It is sufficient to check for its existence.
		if ms, err := getMachineSetFromCAPIMachine(ctx, client, capiMachine); ms != nil {
			return false, nil
		} else if err != nil && !strings.Contains(strings.ToLower(err.Error()), "not found") {
			return false, err
		}
	}
	return true, nil
}

// fetchOwnerRef simply searches a list of OwnerReference objects for a given kind.
func fetchOwnerRef(refList []meta.OwnerReference, kind string) *meta.OwnerReference {
	for _, ref := range refList {
		if ref.Kind == kind {
			return &ref
		}
	}
	return nil
}

// GetManagementOwnerRef returns the reference object pointing to the CAPI machine's manager.
func GetManagementOwnerRef(capiMachine *clusterv1.Machine) *meta.OwnerReference {
	if util.IsControlPlaneMachine(capiMachine) {
		return fetchOwnerRef(capiMachine.OwnerReferences, "KubeadmControlPlane")
	} else if ref := fetchOwnerRef(capiMachine.OwnerReferences, "MachineSet"); ref != nil {
		return ref
	}
	return fetchOwnerRef(capiMachine.OwnerReferences, "EtcdadmCluster")
}

// GetOwnerOfKind returns the Cluster object owning the current resource of passed kind.
func GetOwnerOfKind(ctx context.Context, c clientPkg.Client, owned clientPkg.Object, owner clientPkg.Object) error {
	gvks, _, err := c.Scheme().ObjectKinds(owner)
	if err != nil {
		return errors.Wrapf(err, "finding owner kind for %s/%s", owned.GetName(), owned.GetNamespace())
	} else if len(gvks) != 1 {
		return errors.Errorf(
			"found more than one GVK for owner when finding owner kind for %s/%s", owned.GetName(), owned.GetNamespace())
	}
	gvk := gvks[0]

	for _, ref := range owned.GetOwnerReferences() {
		if ref.Kind != gvk.Kind {
			continue
		}
		key := clientPkg.ObjectKey{Name: ref.Name, Namespace: owned.GetNamespace()}
		if err := c.Get(ctx, key, owner); err != nil {
			return errors.Wrapf(err, "finding owner of kind %s in namespace %s", gvk.Kind, owned.GetNamespace())
		}
		return nil
	}

	return errors.Errorf("couldn't find owner of kind %s in namespace %s", gvk.Kind, owned.GetNamespace())
}

func ContainsNoMatchSubstring(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "no match")
}

func ContainsAlreadyExistsSubstring(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "already exists")
}

// WithClusterSuffix appends a hyphen and the cluster name to a name if not already present.
func WithClusterSuffix(name string, clusterName string) string {
	newName := name
	if !strings.HasSuffix(name, "-"+clusterName) { // Add cluster name suffix if missing.
		newName = name + "-" + clusterName
	}
	return newName
}
