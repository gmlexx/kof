package utils

import (
	"context"
	"fmt"
	"strconv"

	kcmv1alpha1 "github.com/K0rdent/kcm/api/v1alpha1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/record"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const ManagedByLabel = "app.kubernetes.io/managed-by"
const ManagedByValue = "kof-operator"

func GetOwnerReference(owner client.Object, client client.Client) (metav1.OwnerReference, error) {
	gvk := owner.GetObjectKind().GroupVersionKind()

	if gvk.Empty() {
		var err error
		gvk, err = client.GroupVersionKindFor(owner)
		if err != nil {
			return metav1.OwnerReference{}, err
		}
	}

	return metav1.OwnerReference{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Name:       owner.GetName(),
		UID:        owner.GetUID(),
	}, nil
}

func BoolPtr(value bool) *bool {
	// `*bool` fields may point to `true`, `false`, or be `nil`.
	// Direct `&true` is an error.
	return &value
}

func GetEventsAnnotations(cd *kcmv1alpha1.ClusterDeployment) map[string]string {
	var generation string

	if cd.Generation == 0 {
		generation = "nil"
	} else {
		generation = strconv.Itoa(int(cd.Generation))
	}

	return map[string]string{
		"generation": generation,
	}
}

func GetClusterDeploymentStub(name, namespace string) *kcmv1alpha1.ClusterDeployment {
	return &kcmv1alpha1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "k0rdent.mirantis.com/v1alpha1",
			Kind:       kcmv1alpha1.ClusterDeploymentKind,
		},
	}
}

func HandleError(ctx context.Context, reason, message string, cd *kcmv1alpha1.ClusterDeployment, err error, logKeysAndValues ...any) {
	log := log.FromContext(ctx)
	log.Error(err, message, logKeysAndValues...)
	record.Warn(
		cd,
		GetEventsAnnotations(cd),
		reason,
		fmt.Sprintf("%s: %v", message, err),
	)
}
