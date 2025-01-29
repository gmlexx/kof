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

	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	kofv1alpha1 "github.com/k0rdent/kof/promxy-operator/api/v1alpha1"
)

const SecretNameLabel = "k0rdent.mirantis.com/promxy-secret-name"

type PromxyConfigReloadFunc func() error

// PromxyServerGroupReconciler reconciles a PromxyServerGroup object
type PromxyServerGroupReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	RemoteWriteUrl     string
	PromxyConfigReload PromxyConfigReloadFunc
}

// +kubebuilder:rbac:groups=kof.k0rdent.mirantis.com,resources=promxyservergroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kof.k0rdent.mirantis.com,resources=promxyservergroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kof.k0rdent.mirantis.com,resources=promxyservergroups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PromxyServerGroup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *PromxyServerGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	promxyServerGroupsList := &kofv1alpha1.PromxyServerGroupList{}
	opts := []client.ListOption{
		client.InNamespace(req.Namespace),
	}

	if err := r.List(ctx, promxyServerGroupsList, opts...); err != nil {
		log.Error(err, "cannot get promxy server group list")
		return ctrl.Result{}, err
	}

	promxyServerGroupsBySecretName := make(map[string][]*kofv1alpha1.PromxyServerGroup)

	for _, promxyServerGroup := range promxyServerGroupsList.Items {
		name, ok := promxyServerGroup.Labels[SecretNameLabel]
		if !ok {
			log.Info("Skipping promxyServerGroup that doesn't have secret name label", "promxyServerGroup", promxyServerGroup)
			continue
		}
		groups, ok := promxyServerGroupsBySecretName[name]
		if !ok {
			groups = make([]*kofv1alpha1.PromxyServerGroup, 0)
		}
		groups = append(groups, &promxyServerGroup)
		promxyServerGroupsBySecretName[name] = groups
	}

	log.Info("Processing promxy server groups", "promxyServerGroupsBySecretName", promxyServerGroupsBySecretName)

	for name, groups := range promxyServerGroupsBySecretName {
		if err := CreateDashboardSource(r, ctx, req, name, groups, log); err != nil {
			return ctrl.Result{}, err
		}

		if err := CreateSecrets(r, ctx, req, name, groups, log); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PromxyServerGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kofv1alpha1.PromxyServerGroup{}).
		Complete(r)
}

func CreateSecrets(r *PromxyServerGroupReconciler, ctx context.Context, req ctrl.Request, name string, groups []*kofv1alpha1.PromxyServerGroup, log logr.Logger) error {
	secretTemplateData := &PromxyConfig{
		RemoteWriteUrl: r.RemoteWriteUrl,
		ServerGroups:   make([]*PromxyConfigServerGroup, 0),
	}
	for _, group := range groups {
		credentialsSecret := &coreV1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{
			Name:      group.Spec.HttpClient.BasicAuth.CredentialsSecretName,
			Namespace: req.Namespace,
		}, credentialsSecret); err != nil {
			log.Error(err, "cannot read auth credentials secret")
			return err
		}
		secretTemplateData.ServerGroups = append(secretTemplateData.ServerGroups, &PromxyConfigServerGroup{
			Targets:               group.Spec.Targets,
			PathPrefix:            group.Spec.PathPrefix,
			Scheme:                group.Spec.Scheme,
			DialTimeout:           group.Spec.HttpClient.DialTimeout.Duration.String(),
			TlsInsecureSkipVerify: group.Spec.HttpClient.TLSConfig.InsecureSkipVerify,
			Username:              string(credentialsSecret.Data[group.Spec.HttpClient.BasicAuth.UsernameKey]),
			Password:              string(credentialsSecret.Data[group.Spec.HttpClient.BasicAuth.PasswordKey]),
			ClusterName:           group.Spec.ClusterName,
		})
	}
	data, err := RenderPromxySecretTemplate(secretTemplateData)
	if err != nil {
		log.Error(err, "cannot render promxy secret template")
		return err
	}
	secret := &coreV1.Secret{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: req.Namespace,
	}, secret)
	if err != nil && errors.IsNotFound(err) {
		secret.ObjectMeta = metav1.ObjectMeta{
			Name:      name,
			Namespace: req.Namespace,
		}
		setOperatorLabels(&secret.ObjectMeta)
		secret.StringData = map[string]string{
			"config.yaml": data,
		}
		log.Info("Creating promxy config secret", "secretName", name)
		if err := r.Create(ctx, secret); err != nil {
			log.Error(err, "cannot create promxy secret")
			return err
		}
		log.Info("Reloading promxy config")
		if err := r.PromxyConfigReload(); err != nil {
			log.Error(err, "cannot reload promxy config")
			return err
		}
		return nil
	}
	if err != nil {
		log.Error(err, "cannot get promxy secret")
		return err
	}
	setOperatorLabels(&secret.ObjectMeta)
	secret.StringData = map[string]string{
		"config.yaml": data,
	}
	log.Info("Updating promxy config secret", "secretName", name)
	if err := r.Update(ctx, secret); err != nil {
		log.Error(err, "cannot update promxy secret")
		return err
	}
	log.Info("Reloading promxy config")
	if err := r.PromxyConfigReload(); err != nil {
		log.Error(err, "cannot reload promxy config")
		return err
	}
	return nil
}

func CreateDashboardSource(r *PromxyServerGroupReconciler, ctx context.Context, req ctrl.Request, name string, groups []*kofv1alpha1.PromxyServerGroup, log logr.Logger) error {
	for _, group := range groups {

		grafanaDatasourceConfig := &GrafanaDatasourceConfig{
			ObjectName:            fmt.Sprintf("%s-logs", name),
			Namespace:             req.Namespace,
			SecretUsernameKey:     group.Spec.HttpClient.BasicAuth.UsernameKey,
			SecretPasswordKey:     group.Spec.HttpClient.BasicAuth.PasswordKey,
			CredentialsSecretName: group.Spec.HttpClient.BasicAuth.CredentialsSecretName,
			Scheme:                group.Spec.Scheme,
			PathPrefix:            group.Spec.PathPrefix,
		}

		for i, target := range group.Spec.Targets {
			grafanaDatasourceConfig.DatasourceName = fmt.Sprintf("%s-%d", group.Spec.ClusterName, i)
			grafanaDatasourceConfig.TargetHost = target
			datasource := grafanaDatasourceConfig.BuildDatasource()

			err := r.Get(ctx, types.NamespacedName{
				Name:      grafanaDatasourceConfig.ObjectName,
				Namespace: grafanaDatasourceConfig.Namespace,
			}, datasource)

			if err != nil && errors.IsNotFound(err) {
				log.Info("Creating grafana data source", "name", grafanaDatasourceConfig.ObjectName)
				if err := r.Create(ctx, datasource); err != nil {
					log.Error(err, "cannot create grafana data source")
					return err
				}

				log.Info("Reloading promxy config")
				if err := r.PromxyConfigReload(); err != nil {
					log.Error(err, "cannot reload promxy config")
					return err
				}
				return nil
			}

			if err != nil {
				log.Error(err, "cannot get grafana data source")
				return err
			}
		}
	}

	return nil
}

func setOperatorLabels(obj *metav1.ObjectMeta) {
	obj.Labels = map[string]string{
		"app.kubernetes.io/managed-by": "promxy-operator",
	}
}
