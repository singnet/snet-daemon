#  etcd Payment Channel Storage

This is an implementation of the Payment Channel Storage based on etcd.

# Config file

It is possible to configure snet-daemon to run with or without embedded etcd server.

In both ways the configuration file must include *PAYMENT_CHANNEL_STORAGE_CLUSTER* field which contains a comma
separated list of values *etcd_node_name=host:client_port*.

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

    "PAYMENT_CHANNEL_STORAGE": {
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

    "PAYMENT_CHANNEL_STORAGE": {
        "ID": "storage_2",
        "HOST" : "127.0.0.2",
        "CLIENT_PORT": 2379,
        "PEER_PORT": 2380,
        "TOKEN": "payment_channel_storage_token",
        "ENABLED": false
    }
}
```
