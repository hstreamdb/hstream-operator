## 部署 RQLite
```shell
cat << "EOF" | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: rqlite-svc-internal
spec:
  clusterIP: None
  publishNotReadyAddresses: True
  selector:
    app: rqlite
  ports:
    - protocol: TCP
      port: 4001
      targetPort: 4001
---
apiVersion: v1
kind: Service
metadata:
  name: rqlite-svc
spec:
  selector:
    app: rqlite
  ports:
    - protocol: TCP
      port: 4001
      targetPort: 4001
---

apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: rqlite
spec:
  selector:
    matchLabels:
      app: rqlite # has to match .spec.template.metadata.labels
  serviceName: rqlite-svc-internal
  replicas: 3
  podManagementPolicy: "Parallel"
  template:
    metadata:
      labels:
        app: rqlite # has to match .spec.selector.matchLabels
    spec:
      terminationGracePeriodSeconds: 5
      containers:
      - name: rqlite
        image: rqlite/rqlite
        imagePullPolicy: IfNotPresent
        args: ["-disco-mode=dns","-disco-config={\"name\":\"rqlite-svc-internal\"}","-bootstrap-expect","3", "-join-interval=1s", "-join-attempts=120"]
        ports:
        - containerPort: 4001
          name: rqlite
        readinessProbe:
          httpGet:
            scheme: HTTP
            path: /readyz
            port: 4001
          periodSeconds: 5
          timeoutSeconds: 2
          initialDelaySeconds: 2
        livenessProbe:
          httpGet:
            scheme: HTTP
            path: /readyz?noleader
            port: rqlite
          initialDelaySeconds: 2
          timeoutSeconds: 2
          failureThreshold: 3
        volumeMounts:
        - name: rqlite-file
          mountPath: /rqlite/file
      volumes:
      - name: rqlite-file
        emptyDir:
EOF
```
## 部署 Operator 控制器

通过 yaml 方式安装
```shell
kubectl apply -f https://github.com/hstreamdb/hstream-operator/releases/download/0.0.1/hstream-operator.yaml
```

等待 Hstream Operator 控制器就绪
```shell
kubectl get pods -l "control-plane=controller-manager" -n hstream-operator-system
NAME                                                   READY   STATUS    RESTARTS   AGE
hstream-operator-controller-manager-5f4db4654c-hq5d5   2/2     Running   0          25s
```
## 部署 HstreamDB
```shell
cat << "EOF" | kubectl apply -f -
apiVersion: apps.hstream.io/v1alpha1
kind: HStreamDB
metadata:
  name: hstreamdb-sample
spec:
  config:
    metadata-replicate-across: 1
    nshards: 1
    logDeviceConfig:
      {
        "rqlite": {
          "rqlite_uri": "ip://rqlite-svc.default:4001"
        }
      }
  image: hstreamdb/hstream:rqlite
  imagePullPolicy: IfNotPresent

  hserver:
    replicas: 1
    container:
      name: hserver
      command:
        - bash
        - "-c"
        - |
          set -ex
          [[ `hostname` =~ -([0-9]+)$ ]] || exit 1
          ordinal=${BASH_REMATCH[1]}
          /usr/local/bin/hstream-server \
          --config-path /etc/hstream/config.yaml \
          --bind-address 0.0.0.0 \
          --advertised-address $(POD_IP) \
          --port 6570 \
          --internal-port 6571 \
          --seed-nodes "hstreamdb-sample-hserver-0.hstreamdb-sample-hserver:6571" \
          --server-id $((100 + $ordinal)) \
          --metastore-uri rq://rqlite-svc.default:4001 \
          --store-config /etc/logdevice/config.json \
          --store-admin-host hstreamdb-sample-admin-server
  hstore:
    replicas: 3
    container:
      name: hstore
  adminServer:
    replicas: 1
    container:
      name: admin-server
EOF
```

完整示例请查看[complete_apps_v1alpha1_hstreamdb](https://github.com/hstreamdb/hstream-operator/blob/main/config/samples/complete_apps_v1alpha1_hstreamdb.yaml)

Operator 将首先部署3个节点的`hstore`集群，以及1个节点的`admin-server`
```shell
> kubectl get po
NAME                                             READY   STATUS              RESTARTS   AGE
hstreamdb-sample-admin-server-57c4fbd996-kqlv4   1/1     Running             0          3s
hstreamdb-sample-hstore-0                        1/1     Running             0          3s
hstreamdb-sample-hstore-1                        0/1     ContainerCreating   0          3s
hstreamdb-sample-hstore-2                        0/1     ContainerCreating   0          3s
rqlite-0                                         1/1     Running             0          78s
rqlite-1                                         1/1     Running             0          78s
rqlite-2                                         1/1     Running             0          77s
```

查看资源状态以等待`hstore`完成 boostrap
```shell
kc get hstreamdb  hstreamdb-sample -o json | jq ".status.hstoreConfigured"
false
```

待`hstoreConfigured`状态转为`true`时即表示`hstore`已初始化成功，此时 operator 将继续部署1个节点的`hserver`
```shell
kc get po | grep hserver
hstreamdb-sample-hserver-0   1/1   Running   0   15m
```

查看资源状态以等待`hserver`完成 boostrap
```shell
kc get hstreamdb  hstreamdb-sample -o json | jq ".status.hserverConfigured"
false
```

待`hserverConfigured`状态转为`true`时即表示`hserver`已完成初始化。

