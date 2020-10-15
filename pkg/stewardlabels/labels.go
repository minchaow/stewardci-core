package stewardlabels

import (
	"fmt"

	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LabelAsSystemManaged sets label `steward.sap.com/system-managed` at
// the given object.
func LabelAsSystemManaged(obj metav1.Object) {
	if obj == nil {
		return
	}
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[stewardv1alpha1.LabelSystemManaged] = ""
	obj.SetLabels(labels)
}

// LabelAsOwnedByClientNamespace sets some labels on `obj` that identify it
// as owned by the Steward client represented by the given namespace.
// Fails if there's a conflict with existing labels, e.g. `obj` is labelled
// as owned by another Steward client.
func LabelAsOwnedByClientNamespace(obj metav1.Object, owner *corev1.Namespace) error {
	if obj == nil {
		return nil
	}
	labels := []string{
		stewardv1alpha1.LabelOwnerClientName,
		stewardv1alpha1.LabelOwnerClientNamespace,
	}
	return propagate(labels, obj, owner, map[string]string{
		stewardv1alpha1.LabelOwnerClientName:      owner.GetName(),
		stewardv1alpha1.LabelOwnerClientNamespace: owner.GetName(),
	})
}

// LabelAsOwnedByTenant sets some labels on `obj` that identify it
// as owned by the given Steward tenant.
// Fails if there's a conflict with existing labels, e.g. `obj` is labelled
// as owned by another Steward client or tenant.
func LabelAsOwnedByTenant(obj metav1.Object, owner *stewardv1alpha1.Tenant) error {
	if obj == nil {
		return nil
	}
	labels := []string{
		stewardv1alpha1.LabelOwnerClientName,
		stewardv1alpha1.LabelOwnerClientNamespace,
		stewardv1alpha1.LabelOwnerTenantName,
		stewardv1alpha1.LabelOwnerTenantNamespace,
	}
	return propagate(labels, obj, owner, map[string]string{
		stewardv1alpha1.LabelOwnerClientName:      owner.GetNamespace(),
		stewardv1alpha1.LabelOwnerClientNamespace: owner.GetNamespace(),
		stewardv1alpha1.LabelOwnerTenantName:      owner.GetName(),
		stewardv1alpha1.LabelOwnerTenantNamespace: owner.Status.TenantNamespaceName,
	})
}

// LabelAsOwnedByPipelineRun sets some labels on `obj` that identify it
// as owned by the given Steward pipeline run.
// Fails if there's a conflict with existing labels, e.g. `obj` is labelled
// as owned by another Steward client, tenant or pipeline run.
func LabelAsOwnedByPipelineRun(obj metav1.Object, owner *stewardv1alpha1.PipelineRun) error {
	if obj == nil {
		return nil
	}
	labels := []string{
		stewardv1alpha1.LabelOwnerClientName,
		stewardv1alpha1.LabelOwnerClientNamespace,
		stewardv1alpha1.LabelOwnerTenantName,
		stewardv1alpha1.LabelOwnerTenantNamespace,
		stewardv1alpha1.LabelOwnerPipelineRunName,
	}
	return propagate(labels, obj, owner, map[string]string{
		stewardv1alpha1.LabelOwnerTenantNamespace: owner.GetNamespace(),
		stewardv1alpha1.LabelOwnerPipelineRunName: owner.GetName(),
	})
}

func propagate(labels []string, to metav1.Object, from metav1.Object, add map[string]string) error {
	// don't modify original object until merge finished without errors
	temp := make(map[string]string)
	for k, v := range to.GetLabels() {
		temp[k] = v
	}

	merge := func(key string, to, from map[string]string) error {
		fromVal := from[key]
		toVal := to[key]
		if fromVal != "" {
			if toVal != "" && toVal != fromVal {
				return fmt.Errorf(
					"label %q: cannot overwrite existing value %q with %q",
					key, toVal, fromVal,
				)
			}
			to[key] = fromVal
		}
		return nil
	}

	for _, key := range labels {
		if err := merge(key, temp, from.GetLabels()); err != nil {
			return err
		}
		if err := merge(key, temp, add); err != nil {
			return err
		}
	}

	// keep original nil or empty map if merge did not add anything
	if len(temp) > 0 {
		to.SetLabels(temp)
	}
	return nil
}
