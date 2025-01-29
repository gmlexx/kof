package controller

import (
	"encoding/json"
	"fmt"
	"time"

	grafanav1alpha1 "github.com/k0rdent/kof/promxy-operator/api/v1alpha1/grafana"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GrafanaDatasourceConfig struct {
	ObjectName            string
	Namespace             string
	SecretUsernameKey     string
	SecretPasswordKey     string
	CredentialsSecretName string
	DatasourceName        string
	Scheme                string
	TargetHost            string
	PathPrefix            string
}

func (in GrafanaDatasourceConfig) BuildDatasource() (*grafanav1alpha1.GrafanaDatasource, error) {
	secureData := map[string]string{
		"basicAuthPassword": fmt.Sprintf("${%s}", in.SecretPasswordKey),
	}

	secureJSON, err := json.Marshal(secureData)
	if err != nil {
		return nil, err
	}

	return &grafanav1alpha1.GrafanaDatasource{
		TypeMeta: metav1.TypeMeta{
			Kind:       "GrafanaDatasource",
			APIVersion: "grafana.integreatly.org/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      in.ObjectName,
			Namespace: in.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "promxy-operator",
			},
		},
		Spec: grafanav1alpha1.GrafanaDatasourceSpec{
			ValuesFrom: []grafanav1alpha1.ValueFrom{
				{
					TargetPath: "basicAuthUser",
					ValueFrom: grafanav1alpha1.ValueFromSource{
						SecretKeyRef: &coreV1.SecretKeySelector{
							Key: in.SecretUsernameKey,
							LocalObjectReference: coreV1.LocalObjectReference{
								Name: in.CredentialsSecretName,
							},
						},
					},
				},
				{
					TargetPath: "secureJsonData.basicAuthPassword",
					ValueFrom: grafanav1alpha1.ValueFromSource{
						SecretKeyRef: &coreV1.SecretKeySelector{
							Key: in.SecretPasswordKey,
							LocalObjectReference: coreV1.LocalObjectReference{
								Name: in.CredentialsSecretName,
							},
						},
					},
				},
			},
			Datasource: &grafanav1alpha1.GrafanaDatasourceInternal{
				Access:         "proxy",
				Name:           in.DatasourceName,
				Type:           "victoriametrics-logs-datasource",
				URL:            in.buildURL(),
				IsDefault:      boolPtr(false),
				BasicAuth:      boolPtr(true),
				BasicAuthUser:  fmt.Sprintf("${%s}", in.SecretUsernameKey),
				SecureJSONData: secureJSON,
			},
			GrafanaCommonSpec: grafanav1alpha1.GrafanaCommonSpec{
				InstanceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"dashboards": "grafana",
					},
				},
				ResyncPeriod: metav1.Duration{
					Duration: 5 * time.Minute,
				},
			},
		},
		Status: grafanav1alpha1.GrafanaDatasourceStatus{},
	}, nil
}

func (in GrafanaDatasourceConfig) buildURL() string {
	var pathPrefix string = in.PathPrefix

	if len(pathPrefix) > 0 && pathPrefix[0] == '/' {
		pathPrefix = pathPrefix[1:]
	}
	return fmt.Sprintf("%s://%s/%s", in.Scheme, in.TargetHost, pathPrefix)
}

func boolPtr(val bool) *bool {
	return &val
}
