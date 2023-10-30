## Release Note 🍻

HStream Operator 0.0.8 has been released.

### Supported version

- `apps.hstream.io/v1alpha2`

  - `HStreamDB` with [rqlite](https://hub.docker.com/layers/hstreamdb/hstream/rqlite_v0.17.3/images/sha256-a7d489f2b33959f6a4326850f143ccbca914240092d0f2f706c23679e369a9b5?context=explore)

### Enhancements 🚀

- `apps.hstream.io/v1alpha2`

  - Moderate readiness probes in hserver and hmeta.

- `apps.hstream.io/v1beta1`

  Support new kinds: `ConnectorTemplate` and `Connector`, which can be used to create and manage HStream IO Connectors. Currently supported connectors are:

  - `sink-elasticsearch`: Sink data to ElasticSearch.

### Warning 🚨

The API version `v1alpha2` isn't compatible with v1alpha1.
