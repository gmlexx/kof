package k8s

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetRemoteSecrets(ctx context.Context, k8sClient client.Client) (*corev1.SecretList, error) {
	secretList := &corev1.SecretList{}

	if err := k8sClient.List(
		ctx,
		secretList,
		client.MatchingLabels(map[string]string{"remote-secret": "true"}),
	); err != nil {
		return secretList, err
	}

	return secretList, nil
}

func GetKubeconfigFromSecretList(secretList *corev1.SecretList) [][]byte {
	kubeconfigList := make([][]byte, 0, len(secretList.Items))

	for _, secret := range secretList.Items {
		key := strings.Replace(secret.Name, "istio-remote-secret-", "", 1)
		kubeconfig, ok := secret.Data[key]
		if !ok {
			continue
		}
		kubeconfigList = append(kubeconfigList, kubeconfig)
	}
	return kubeconfigList
}
