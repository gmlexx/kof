/*
Copyright 2025.

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

package controller

import (
	"context"
	"fmt"

	kcmv1alpha1 "github.com/K0rdent/kcm/api/v1alpha1"
	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	v1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const istioCANamespace = "istio-system"
const istioReleaseName = "kof-istio"

// ClusterDeploymentReconciler reconciles a ClusterDeployment object
type ClusterDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=k0rdent.mirantis.com,resources=clusterdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k0rdent.mirantis.com,resources=clusterdeployments/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *ClusterDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	clusterDeployment := &kcmv1alpha1.ClusterDeployment{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      req.Name,
		Namespace: req.Namespace,
	}, clusterDeployment); err != nil {
		if errors.IsNotFound(err) {
			// Put code to handle deletion case
			return ctrl.Result{}, nil
		}
		log.Error(err, "cannot read clusterDeployment")
		return ctrl.Result{}, err
	}

	if clusterDeployment.Spec.Config == nil {
		return ctrl.Result{}, nil
	}
	config, err := ReadClusterDeploymentConfig(clusterDeployment.Spec.Config.Raw)
	if err != nil {
		log.Error(err, "cannot read cluster config labels")
		return ctrl.Result{}, err
	}

	if istioRole, ok := config.ClusterLabels["k0rdent.mirantis.com/istio-role"]; ok {
		if istioRole != "child" {
			return ctrl.Result{}, nil
		}
		certName := fmt.Sprintf("kof-istio-%s-ca", clusterDeployment.Name)
		cert := &cmv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      certName,
				Namespace: istioCANamespace,
			},
		}
		if err := r.Get(ctx, types.NamespacedName{
			Name:      certName,
			Namespace: istioCANamespace,
		}, cert); err != nil {
			if !errors.IsNotFound(err) {
				log.Error(err, "cannot read cluster config labels")
				return ctrl.Result{}, err
			}
			cert.Labels = map[string]string{
				"app.kubernetes.io/managed-by": "kof-operator",
			}
			cert.Spec = cmv1.CertificateSpec{
				IsCA:       true,
				CommonName: fmt.Sprintf("%s CA", clusterDeployment.Name),
				Subject: &cmv1.X509Subject{
					Organizations: []string{"Istio"},
				},
				PrivateKey: &cmv1.CertificatePrivateKey{
					Algorithm: "ECDSA",
					Size:      256,
				},
				SecretName: certName,
				IssuerRef: v1.ObjectReference{
					Name:  fmt.Sprintf("%s-root", istioReleaseName),
					Kind:  "Issuer",
					Group: "cert-manager.io",
				},
			}
			log.Info("Creating Intermediate Istio CA certificate", "certificateName", cert.Name)
			if err := r.Create(ctx, cert); err != nil {
				log.Error(err, "cannot create certificate")
				return ctrl.Result{}, err
			}
		}

	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kcmv1alpha1.ClusterDeployment{}).
		Complete(r)
}
