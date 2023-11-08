# HStream Operator

## Introduction

HStream Operator is a Kubernetes operator designed to manage and maintain the HStreamDB cluster within a Kubernetes environment. The HStreamDB cluster comprises of several components including:

- [HServer](https://docs.hstream.io/reference/architecture/hserver.html)
- [HStore](https://docs.hstream.io/reference/architecture/hstore.html)
- HMeta
- AdminServer

It simplifies the deployment, scaling, and operations of HStreamDB clusters on Kubernetes, making it easier for users to manage their HStream components effectively. We use and get benefits from [kubebuilder](https://book.kubebuilder.io/) to simplify the development of the operator.

## Installation

We recommend using the [Helm](https://helm.sh/) package manager to install the HStreamDB operator on your Kubernetes cluster.

> Currently, we haven't released the chart because of this operator is still at an early stage. So you
> need to clone this repo and install the chart from the local directory.

```sh
git clone https://github.com/hstreamdb/hstream-operator.git && cd hstream-operator
helm install hstream-operator deploy/charts/hstream-operator -n hstream-operator-system --create-namespace
```

Every releases will be published to [GitHub Releases](https://github.com/hstreamdb/hstream-operator/releases), you
can also install the operator with the following command:

```sh
kubectl apply -f https://github.com/hstreamdb/hstream-operator/releases/download/0.0.8/hstream-operator.yaml
```

Replace `0.0.8` with the version you want to install.

### Check the status

You can check the status of the operator by running:

```sh
kubectl get pods -l "control-plane=hstream-operator-manager" -n hstream-operator-system
```

Expected output:

```sh
NAME                                                  READY   STATUS    RESTARTS      AGE
hstream-operator-controller-manager-f989476d4-qllfs   1/1     Running   1 (16h ago)   16h
```

### Bootstrap a HStreamDB cluster

After installing the operator, you can bootstrap a HStreamDB cluster by applying `config/samples/hstreamdb.yaml`:

```sh
kubectl apply -f config/samples/hstreamdb.yaml
```

> Note: you need to provide `volumeClaimTemplate` which comments out in the sample file.

You can check the status of the HStreamDB cluster by running:

```sh
kubectl get po -n hstreamdb
```

Expected output:

```sh
NAME                                             READY   STATUS    RESTARTS   AGE
hstreamdb-sample-hmeta-2                         1/1     Running   0          7m45s
hstreamdb-sample-hmeta-0                         1/1     Running   0          7m45s
hstreamdb-sample-hmeta-1                         1/1     Running   0          7m45s
hstreamdb-sample-admin-server-6c547b85c7-7h9gw   1/1     Running   0          7m34s
hstreamdb-sample-hstore-0                        1/1     Running   0          7m34s
hstreamdb-sample-hstore-1                        1/1     Running   0          7m34s
hstreamdb-sample-hstore-2                        1/1     Running   0          7m34s
hstreamdb-sample-hserver-0                       1/1     Running   0          7m18s
hstreamdb-sample-hserver-2                       1/1     Running   0          7m18s
hstreamdb-sample-hserver-1                       1/1     Running   0          7m18s
```

As a result, we have a HStreamDB cluster with 3 HMeta nodes, 3 HStore nodes, 3 HServer nodes, and 1 AdminServer node.

## Testing in a local environment

You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.

**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster

1. Build and push your image to the location specified by `IMG`:

   ```sh
   make docker-build docker-push IMG=<some-registry>/hstream-operator:tag
   ```

2. Deploy the controller to the cluster with the image specified by `IMG`:

   ```sh
   make deploy IMG=<some-registry>/hstream-operator:tag
   ```

### Uninstall CRDs

To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller

UnDeploy the controller from the cluster:

```sh
make undeploy
```

## Contributing

// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works

This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

### Test it out

1. Install the CRDs into the cluster:

   ```sh
   make install
   ```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

   ```sh
   make run
   ```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions

If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](./LICENSE) for full content.
