# vault-proxy-exporter

The exporter allows you to safely scrape metrics from Hashicorp Vault approle + kube-rbac-proxy.

# How to use

### Configure vault
1. Create policy
```
$ vault policy write prometheus-monitoring - << EOF
path "/sys/metrics" {
  capabilities = ["read"]
}
EOF
```
2. Create approle
```
$ vault auth enable -path=approle approle
$ vault write auth/approle/role/prometheus-monitoring \
    token_type=batch \
    token_ttl=5m \
    token_max_ttl=5m \
    policies="prometheus-monitoring"

## get role-id
$ vault read auth/approle/role/prometheus-monitoring/role-id
Key        Value
---        -----
role_id    xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

## get secret-id
$ vault write -f auth/approle/role/prometheus-monitoring/secret-id
Key                   Value
---                   -----
secret_id             yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy
secret_id_accessor    zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz
secret_id_num_uses    0
secret_id_ttl         0s
```

### Deploy vault-proxy-exporter

1. Add helm repo
```
$ helm repo add vault-proxy-exporter https://legolego621.github.io/vault-proxy-exporter/helm/charts
$ helm repo update vault-proxy-exporter
```
2. Configure values
```
$ cat << 'EOF' > values-exporter.yaml
env:
  # address of vault
  - name: VAULT_PROXY_EXPORTER_VAULT_ENDPOINT
    value: https://vault.example.com:8200
  # use insecure skip verify tls connection
  - name: VAULT_PROXY_EXPORTER_TLS_INSECURE_SKIP_VERIFY
    value: "false"
  # method of auth
  - name: VAULT_PROXY_EXPORTER_VAULT_AUTH_METHOD
    value: approle # you can use token or approle, default is approle
  # path of approle
  - name: VAULT_PROXY_EXPORTER_VAULT_APPROLE_PATH
    value: approle
  # id of approle
  - name: VAULT_PROXY_EXPORTER_VAULT_APPROLE_ID
    value: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
  # secret of approle
  - name: VAULT_PROXY_EXPORTER_VAULT_APPROLE_SECRET
    value: yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy
  # time interval of update token by approle (if use approle)
  - name: VAULT_PROXY_EXPORTER_VAULT_APPROLE_TOKEN_UPDATE_PERIOD_SECONDS
    value: "60"

serviceMonitor:
  exporterMetrics:
    enabled: true
    namespace: ""
    namespaceSelector: {}
    labels: {}
    targetLabels: []
    relabelings: []
    metricRelabelings: []
    scrapeInterval: 30s
    tlsConfig:
      insecureSkipVerify: true
    bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
  vaultMetrics:
    enabled: true
    namespace: ""
    namespaceSelector: {}
    labels: {}
    targetLabels: []
    relabelings: []
    metricRelabelings: []
    scrapeInterval: 30s
    tlsConfig:
      insecureSkipVerify: true
    bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
EOF
```
3. Deploy exporter

```
$ helm upgrade --install instance vault-proxy-exporter/vault-proxy-exporter -f ./values-exporter.yaml
```

### Configure prometheus
1. Configure rbac for prometheus/vmagent/...etc.
```
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: vmagent-vault-metrics
rules:
  - nonResourceURLs:
      - /exporter/metrics
      - /vault/metrics
    verbs:
      - get
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: vmagent-vault-metrics
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: vmagent-vault-metrics
subjects:
- kind: ServiceAccount
  name: vmagent
  namespace: monitoring
```
