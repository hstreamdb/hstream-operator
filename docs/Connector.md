# Connector

This document describes how to use `Connector` with HStream Operator.

> Before you start, make sure you have already installed HStream Operator. If not, please refer to [README](../README.md) to install it.
>
> **NOTE:** Connectors support is still in an early stage, and the spec may change in the future.

## Connector Types

Currently, HStream Operator supports only one type of connector: `sink-elasticsearch`.

## Create a Connector Template

A connector template is a `ConfigMap` internally that contains the configuration of a connector. It is used to keep a shard configuration for multiple connectors. But even if you need to create only one connector, you still need to create a connector template to store the configuration. Below is an example (with partial configuration) of a connector template:

```yaml
apiVersion: apps.hstream.io/v1beta1
kind: ConnectorTemplate
metadata:
  name: sink-es-template
spec:
  type: sink-elasticsearch
  config: |
    {
      "auth": "basic",
      "username": "elastic",
      "password": "password",
    }
```

The `spec.type` field specifies the type of the connector template. The `spec.config` field specifies the configuration of the connector template. The configuration is a JSON string that will be passed to the connector.

Refer to [apps_v1beta1_connectortemplate.yaml](https://github.com/hstreamdb/hstream-operator/blob/main/config/samples/apps_v1beta1_connectortemplate.yaml) for a complete example.

## Create a Connector

After creating a connector template, you can create a connector by applying a `Connector` resource. Below is an example of a connector:

```yaml
apiVersion: apps.hstream.io/v1beta1
kind: Connector
metadata:
  name: sink-es
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "9400"
spec:
  type: sink-elasticsearch
  templateName: sink-es-template
  streams:
    - stream01
  patches:
    stream01:
      offsetStream: stream01-offset
      index: index01
  hserverEndpoint: hstreamdb-sample-internal-hserver.hstreamdb:6570
  container:
    ports:
      - name: prom
        containerPort: 9400
    resources:
      limits:
        cpu: 500m
        memory: 256Mi
      requests:
        cpu: 100m
        memory: 128Mi
```

View the [Connector Spec](#spec) section for more details.

### Spec

| Field                  | Optional | Description                                                                                                                      |
| ---------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------- |
| `spec.type`            | `false`  | The type of the connector.                                                                                                       |
| `spec.templateName`    | `false`  | The name of the connector template (see [Create a Connector Template](#create-a-connector-template)).                            |
| `spec.streams`         | `false`  | The streams that the connector will consume from.                                                                                |
| `spec.patches`         | `true`   | Patches will merge into the configuration of the connector template. You can use it to override or supplement the configuration. |
| `spec.hserverEndpoint` | `false`  | The endpoint of the HServer.                                                                                                     |
| `spec.container`       | `true`   | Used to override the connector container spec.                                                                                   |
