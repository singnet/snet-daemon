#  etcd Payment Channel Storage

This is an implementation of the Payment Channel Storage based on etcd.

The configuration file must include *PAYMENT_CHANNEL_STORAGE_CLUSTER* field which contains a comma
separated list of values *etcd_node_name=host:client_port*.

This field is used both by etcd server and client:

```json
{
  "PAYMENT_CHANNEL_STORAGE_CLUSTER": "storage_2=http://127.0.0.2:2380,storage_3=http://127.0.0.3:2380"
}
```

# etcd server configuration

To use embedded etcd server in snet-daemon the configuration file needs to
contain the JSON map with name *PAYMENT_CHANNEL_STORAGE_SERVER* and values:

| Field name  | Description                                        |Default Value|
|-------------|----------------------------------------------------|-------------|
| ID          | unique name of the etcd server node                |             |
| SCHEMA      | URL schema used to create client and peer and urls |http         |
| HOST        | host where the etcd server is executed             |             |
| CLIENT_PORT | port to listen clients requests                    |2379         |
| PEER_PORT   | port to listen etcd peers                          |2380         |
| TOKEN       | unique initial cluster token                       |             |
| ENABLED     | enable running embedded etcd server                |true         |


Schema, host, and CLIENT_PORT/PEER_PORT are used together to compose etcd listen-client-urls/listen-peer-urls
(see the link below).

Using unique token, etcd can generate unique cluster IDs and member IDs for the clusters even if they otherwise have
the exact same configuration. This can protect etcd from cross-cluster-interaction, which might corrupt the clusters.

For more details see
[etcd Clustering Guide](https://github.com/etcd-io/etcd/blob/master/Documentation/op-guide/clustering.md) link.

It is possible to configure snet-daemon to run with or without embedded etcd server.

Config for snet-daemon that does not run embedded etcd node:

```json
{
  "PAYMENT_CHANNEL_STORAGE_CLUSTER": "storage_2=http://127.0.0.2:2380,storage_3=http://127.0.0.3:2380"
}
```

Config for snet-daemon that runs embedded etcd node:

```json
{
    "PAYMENT_CHANNEL_STORAGE_CLUSTER": "storage_1=http://127.0.0.1:2380",

    "PAYMENT_CHANNEL_STORAGE_SERVER": {
        "ID": "storage_1",
        "HOST" : "127.0.0.1",
        "CLIENT_PORT": 2379,
        "PEER_PORT": 2380,
        "TOKEN": "payment_channel_storage_token",
        "ENABLED": true
    }
}
```

Note that it is possible to disable running an embedded etcd server in the config file by setting *ENABLED* field to false:

```json
{
    "PAYMENT_CHANNEL_STORAGE_CLUSTER": "storage_1=http://127.0.0.1:2380,storage_2=http://127.0.0.2:2380,storage_3=http://127.0.0.3:2380",

    "PAYMENT_CHANNEL_STORAGE_SERVER": {
        "ID": "storage_2",
        "HOST" : "127.0.0.2",
        "CLIENT_PORT": 2379,
        "PEER_PORT": 2380,
        "TOKEN": "payment_channel_storage_token",
        "ENABLED": false
    }
}
```

# etcd client configuration

*PAYMENT_CHANNEL_STORAGE_CLIENT* JSON map can be used to configure payment channel storage etcd client.

| Field name         | Description                                   |Default Value|
|--------------------|-----------------------------------------------|-------------|
| CONNECTION_TIMEOUT | timeout for failing to establish a connection |5            |
| REQUEST_TIMEOUT    | per request timeout                           |3            |


```json
{
    "PAYMENT_CHANNEL_STORAGE_CLUSTER": "storage_1=http://127.0.0.1:2380,storage_2=http://127.0.0.2:2380,storage_3=http://127.0.0.3:2380",

    "PAYMENT_CHANNEL_STORAGE_CLIENT": {
        "CONNECTION_TIMEOUT": 2379,
        "REQUEST_TIMEOUT": 2380
    }
}
```