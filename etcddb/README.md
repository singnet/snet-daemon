#  etcd Payment Channel Storage

This is an implementation of the Payment Channel Storage based on etcd.

# etcd server configuration

To use embedded etcd server in snet-daemon the configuration file needs to
contain the JSON map with name *PAYMENT_CHANNEL_STORAGE_SERVER* and values:

| Field name  | Description                                        |Default Value                  |
|-------------|----------------------------------------------------|-------------------------------|
| id          | unique name of the etcd server node                |storage-1                      |
| schema      | URL schema used to create client and peer and urls |http                           |
| host        | host where the etcd server is executed             |127.0.0.1                      |
| client_port | port to listen clients requests                    |2379                           |
| peer_port   | port to listen etcd peers                          |2380                           |
| token       | unique initial cluster token                       |unique-token                   |
| cluster     | initial cluster configuration for bootstrapping    |storage-1=http://127.0.0.1:2380|
| enabled     | enable running embedded etcd server                |true                           |


The cluster field is a comma-separated list of one or more etcd peer URLs in form of *id=host:peer_port*.

schema, host, and client_port/peer_port are used together to compose etcd listen-client-urls/listen-peer-urls
(see the link below).

Using unique token, etcd can generate unique cluster IDs and member IDs for the clusters even if they otherwise have
the exact same configuration. This can protect etcd from cross-cluster-interaction, which might corrupt the clusters.

For more details see
[etcd Clustering Guide](https://github.com/etcd-io/etcd/blob/master/Documentation/op-guide/clustering.md) link.

It is possible to configure snet-daemon to run with or without embedded etcd server.


If *PAYMENT_CHANNEL_STORAGE_SERVER* field is not set in the configuration file its *enabled*
field is set to *false* by default and in this case the etcd server is not started.

Config for snet-daemon that does not run embedded etcd node:
* *PAYMENT_CHANNEL_STORAGE_SERVER* field is not set in the configuration file
```json
{}
```
* *enabled* field is set to _false_
```json
{
    "payment_channel_storage_server": {
        "id": "storage-2",
        "host" : "127.0.0.2",
        "client_port": 2379,
        "peer_port": 2380,
        "token": "unique-token",
        "cluster": "storage-1=http://127.0.0.1:2380,storage-2=http://127.0.0.2:2380,storage-3=http://127.0.0.3:2380",
        "enabled": false
    }
}

Config for snet-daemon that runs embedded etcd node:

* *enabled* field is set to _true_

```json
{
    "PAYMENT_CHANNEL_STORAGE_SERVER": {
        "id": "storage-1",
        "host" : "127.0.0.1",
        "client_port": 2379,
        "peer_port": 2380,
        "token": "unique-token",
        "cluster": "storage-1=http://127.0.0.1:2380",
        "enabled": true
    }
}
```

* *enabled* field is omitted

```json
{
    "PAYMENT_CHANNEL_STORAGE_SERVER": {
        "id": "storage-1",
        "host" : "127.0.0.1",
        "client_port": 2379,
        "peer_port": 2380,
        "token": "unique-token",
        "cluster": "storage-1=http://127.0.0.1:2380",
    }
}
```

Note that it is possible to disable running an embedded etcd server in the config file by setting *enabled* field to false:


# etcd client configuration

*PAYMENT_CHANNEL_STORAGE_CLIENT* JSON map can be used to configure payment channel storage etcd client.

| Field name         | Description                                   |Default Value            |
|--------------------|-----------------------------------------------|-------------------------|
| connection_timeout | timeout for failing to establish a connection |5000                     |
| request_timeout    | per request timeout                           |3000                     |
| endpoints          | list of etcd cluster endpoints                |["http://127.0.0.1:2379"]|

Connection and request timeoutes are measured in milliseconds.

```json
{
    "PAYMENT_CHANNEL_STORAGE_CLIENT": {
        "connection_timeout": 5000,
        "request_timeout": 3000,
        "endpoints": ["http://127.0.0.1:2379"]
    }
}
```
