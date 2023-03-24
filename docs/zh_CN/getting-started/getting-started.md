## 部署 Operator 控制器

通过 yaml 方式安装
```shell
kubectl apply -f https://github.com/hstreamdb/hstream-operator/releases/download/0.0.2/hstream-operator.yaml
```

等待 Hstream Operator 控制器就绪
```shell
kubectl get pods -l "control-plane=controller-manager" -n hstream-operator-system
NAME                                                   READY   STATUS    RESTARTS   AGE
hstream-operator-controller-manager-5f4db4654c-hq5d5   2/2     Running   0          25s
```
## 部署 HstreamDB
```shell
kubectl apply -f https://github.com/hstreamdb/hstream-operator/blob/main/config/samples/hstreamdb.yaml
```

完整示例请查看[hstreamdb_complete_sample](https://github.com/hstreamdb/hstream-operator/blob/main/config/samples/hstreamdb_complete_sample.yaml)

Operator 将部署1个节点的`rqlite`集群、3个节点的`hstore`集群，以及1个节点的`admin-server`
```shell
> kubectl get po
NAME                                             READY   STATUS              RESTARTS   AGE
hstreamdb-sample-admin-server-57c4fbd996-kqlv4   1/1     Running             0          3s
hstreamdb-sample-hstore-0                        1/1     Running             0          3s
hstreamdb-sample-hstore-1                        0/1     ContainerCreating   0          3s
hstreamdb-sample-hstore-2                        0/1     ContainerCreating   0          3s
hstreamdb-sample-rqlite-0                        1/1     Running             0          78s
```

查看资源状态以等待`hstore`完成 boostrap
```shell
kubectl get hstreamdb  hstreamdb-sample -o json | jq ".status.hstore.bootstrapped"
false
```

待`bootstrapped`状态转为`true`时即表示`hstore`已初始化成功，此时 operator 将继续部署1个节点的`hserver`
```shell
kubectl get po | grep hserver
hstreamdb-sample-hserver-0   1/1   Running   0   15m
```

查看资源状态以等待`hserver`完成 boostrap
```shell
kubectl get hstreamdb  hstreamdb-sample -o json | jq ".status.hserver.bootstrapped"
false
```

待`bootstrapped`状态转为`true`时即表示`hserver`已完成初始化。

