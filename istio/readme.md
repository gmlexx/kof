Installation order based on this doc
https://istio.io/latest/docs/setup/install/multicluster/multi-primary_multi-network/

Managed cluster:

```bash
helm install istio-base istio/base -n istio-system --create-namespace
helm install istiod istio/istiod -n istio-system -f istio/managed.yaml

helm install istio-eastwestgateway istio/gateway -n istio-system -f istio/managed-gateway.yaml

kubectl apply -n istio-system -f istio/expose.yaml

./bin/istio-1.24.2/bin/istioctl create-remote-secret --name=alarkov-aws-standalone-istio-managed > dev/alarkov-aws-standalone-istio-managed-remote-secret.yaml

# complete remote secret setup
kubectl apply -f dev/alarkov-aws-standalone-istio-storage-remote-secret.yaml

kubectl label namespace kof istio-injection=enabled
kubectl delete pod --all -n kof
```
Storage cluster:

```bash
helm install istio-base istio/base -n istio-system --create-namespace
helm install istiod istio/istiod -n istio-system -f istio/storage.yaml

helm install istio-eastwestgateway istio/gateway -n istio-system -f istio/storage-gateway.yaml

kubectl apply -n istio-system -f istio/expose.yaml

./bin/istio-1.24.2/bin/istioctl create-remote-secret --name=alarkov-aws-standalone-istio-storage > dev/alarkov-aws-standalone-istio-storage-remote-secret.yaml

# complete remote secret setup
kubectl apply -f dev/alarkov-aws-standalone-istio-managed-remote-secret.yaml

kubectl label namespace kof istio-injection=enabled
kubectl delete pod --all -n kof
```
