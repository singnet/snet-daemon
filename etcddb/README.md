#  etcd Payment Channel Storage


To enable etcd server as a payment channel storage in snet-daemon configure the following properties
in the JSON config file: ( Please note this is applicable only when the block chain is enabled)

* *payment_channel_storage_type*
* *payment_channel_storage_client*
* *payment_channel_storage_server*

## etcd storage type

There are two payment channel storage types which are currently supported by snet daemon: *memory* and *etcd*.
*memory* storage type is used in configuration where only one service replica is used by snet-daemon or
for testing purposes.

To run snet-daemon with several replicas set the payment_channel_storage_type is now initialized from Organization metadata:
```json
{
  "payment_channel_storage_type": "etcd"
}
```

## etcd client configuration

*payment_channel_storage_client* JSON map can be used to configure payment channel storage etcd client.

| Field name         | Description                                   | Default Value               |
|--------------------|-----------------------------------------------|-----------------------------|
| connection_timeout | timeout for failing to establish a connection | from org metadata           |
| request_timeout    | per request timeout                           | from org metadata           |
| endpoints          | list of etcd cluster endpoints (host:port)    | from org metadata           |


**endpoints from the config are ignored and taken only from ipfs metadata**

Endpoints consist of a list of URLs which points to etcd cluster servers.


The following config describes a client which connects to 3 etcd server nodes and the data is 
retrieved from Organization Metadata:
```json
{
	"payment_channel_storage_client": {
		"connection_timeout": "5s",
		"request_timeout": "3s",
		"endpoints": ["http://127.0.0.1:2379", "http://127.0.0.2:2379", "http://127.0.0.3:2379"]
	}
}
```

## etcd client configuration ( https mode)
if the client end point is https, then you will need to add the following on your configuration to use
the certificates to connect 
  "payment_channel_cert_path": "<locationToFile>",
  "payment_channel_ca_path": "<locationToFile>",
  "payment_channel_key_path": "<locationToFile>",


## etcd server configuration 
The latest Daemon expects an etcd cluster setup already available , in case you wish to set up your own
cluster , please go over the documentation below
To use embedded etcd server in snet-daemon the configuration file needs to
contain the  *payment_channel_storage_server* JSON map with fields:

| Field name      | Description                                             | Default Value                  |
|-----------------|---------------------------------------------------------|--------------------------------|
| id              | unique name of the etcd server node                     | storage-1                      |
| schema          | URL schema used to create client and peer and urls      | http                           |
| host            | host where the etcd server is executed                  | 127.0.0.1                      |
| client_port     | port to listen clients requests                         | 2379                           |
| peer_port       | port to listen etcd peers                               | 2380                           |
| token           | unique initial cluster token                            | unique-token                   |
| cluster         | initial cluster configuration for bootstrapping         | storage-1=http://127.0.0.1:2380|
| startup_timeout | time to wait that etcd server is successfully started   | 1 minute                       |
| data_dir        | directory where etcd server stores its data             | storage-data-dir-1.etcd        |
| log_level       | etcd server logging level (error, warning, info, debug) | info                           |
| log_outputs     | file path to append server logs to or stderr/stdout     | stderr                         |
| enabled         | enable running embedded etcd server                     | true                           |

**log_outputs can accept array**

The cluster field is a comma-separated list of one or more etcd peer URLs in form of *id=host:peer_port*.

schema, host, and client_port/peer_port are used together to compose etcd listen-client-urls/listen-peer-urls
(see the link below).

Using unique token, etcd can generate unique cluster IDs and member IDs for the clusters even if they otherwise have
the exact same configuration. This can protect etcd from cross-cluster-interaction, which might corrupt the clusters.

To disable etcd server log messages set *log_level* field to *error*.

For more details see
[etcd Clustering Guide](https://github.com/etcd-io/etcd/blob/master/Documentation/op-guide/clustering.md) link.

It is possible to configure snet-daemon to run with or without embedded etcd server using the *enabled* property.

Config for snet-daemon that runs embedded etcd server:

* *enabled* field is set to _false_ , ETCD cluster set up is retrieved from the Organization metadata , If you 
want to set up a local cluster , then payment_channel_storage_server.enabled configuration needs to be set to true.

```json
{
    "payment_channel_storage_server": {
        "id": "storage-1",
        "host" : "127.0.0.1",
        "client_port": 2379,
        "log_level": "info",
        "log_outputs": [
          "./etcd-server.log"
        ],
        "peer_port": 2380,
        "token": "unique-token",
        "cluster": "storage-1=http://127.0.0.1:2380",
        "enabled": true
    }
}
```


Config for snet-daemon that does not run embedded etcd node:
* *enabled* field is set to _false_
```json
{
  "payment_channel_storage_server": {
    "id": "storage-2",
    "host": "127.0.0.2",
    "client_port": 2379,
    "peer_port": 2380,
    "log_level": "info",
    "log_outputs": [
      "./etcd-server.log"
    ],
    "token": "unique-token",
    "cluster": "storage-1=http://127.0.0.1:2380,storage-2=http://127.0.0.2:2380,storage-3=http://127.0.0.3:2380",
    "enabled": false
  }
}
```
