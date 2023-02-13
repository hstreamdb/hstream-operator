# hstream-operator

The HStream Operator provides [Kubernetes](https://kubernetes.io/) native deployment and management of [hstream](https://hstream.io). The purpose of this project is to simplify and automate the configuration of the hstream cluster.

## Prerequisites

- Kubernetes 1.20+

## Installing the Chart

To install the chart with the release name `hstream-operator`:

```console
## Add the hstream Helm repository
$ helm repo add hstream https://repos.emqx.io/charts
$ helm repo update

## Install the hstream-operator helm chart
$ helm install hstream-operator hstream/hstream-operator \
      --namespace hstream \
      --create-namespace
```

> **Tip**: List all releases using `helm ls -A`

## Uninstalling the Chart

To uninstall/delete the `hstream-operator` deployment:

```console
$ helm delete hstream-operator -n hstream
```

## Configuration

| Parameter                    | Description                                                                                                                   | Default |
|------------------------------|-------------------------------------------------------------------------------------------------------------------------------| ------- |
| `image.repository`           | Image repository                                                                                                              | `hstreamdb/hstream-operator-controller` |
| `image.tag`                  | Image tag                                                                                                                     | `{{RELEASE_VERSION}}` |
| `image.pullPolicy`           | Image pull policy                                                                                                             | `IfNotPresent` |
| `nameOverride`               | Override chart name                                                                                                           | `""` |
| `fullnameOverride`           | Default fully qualified app name                                                                                             | `""` |
| `namespace`                  | namespace                                                                                                                    | `""` |
| `replicaCount`               | Number of cert-manager replicas                                                                                               | `1` |
| `serviceAccount.create`      | If `true`, create a new service account                                                                                       | `true` |
| `serviceAccount.name`        | Service account to be used. If not set and `serviceAccount.create` is `true`, a name is generated using the fullname template |  |
| `serviceAccount.annotations` | Annotations to add to the service account                                                                                     |  |
| `resources`                  | CPU/memory resource requests/limits                                                                                           | `{}` |
| `nodeSelector`               | Node labels for pod assignment                                                                                                | `{}` |
| `affinity`                   | Node affinity for pod assignment                                                                                              | `{}` |
| `tolerations`                | Node tolerations for pod assignment                                                                                           | `[]` |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`.

Alternatively, a YAML file that specifies the values for the above parameters can be provided while installing the chart. For example,

```console
$ helm install hstream-operator -f values.yaml .
```
> **Tip**: You can use the default [values.yaml](https://github.com/hstreamdb/hstream-operator/tree/main/deploy/charts/hstream-operator/values.yaml)

## Contributing

This chart is maintained at [github.com/hstreamdb/hstream-operator](https://github.com/hstreamdb/hstream-operator/tree/main/deploy/charts/hstream-operator).
