# Daemon Metrics & Monitoring Services

```snet-daemon``` sends heartbeat upon the request from a monitoring service or any other component.
Apart from heartbeat, it will expose few custom metrics as mentioned below metrics section.
<br/>
The critical daemon updates, warning, or errors can be reported to a pre-configured email using a default
notification service offered by SingularityNet. It's just an optional service. Service providers are free to choose
thier own endpoints/web hooks for sending alerts.

### Registration

Daemon Registration is required for the metrics services to uniquely identify every daemon and store the
metrics accordingly. To register with metrics service, every daemon must have a Unique Identity i.e. DaemonID.
Daemon ID is a SHA256 hash value generated by using - Org Id, Service ID, Group Id of Daemon (Derived from service
metadata) and Daemon Endpoint.

```
daemonID = SHA256(config.GetString(config.OrgnaizationId) + 
                  config.GetString(config.ServiceId) + 
                  group_Id + //Groupd Id of daemon
                  config.GetString(config.DaemonEndPoint)
```

Post beta, this ID will be used to enable Token based authentication for accessing metrics services.

### Heartbeat

Heartbeat indicates the state of the Daemon and the service associated with it. The heartbeat message from the daemon
wraps the heartbeat of the service and sends it when requested. Heartbeat is always pull based, i.e. The monitoring
service
will have to call the heartbeat service to get the Daemon state. <br/>

When the monitoring services calls daemon heartbeat, the daemon internally makes a call to service heartbeat endpoint.
Upon receiving the service heartbeat, Daemon wraps it in the final heartbeat and sends it to the calling service.

For the service heartbeat, implementation followed the standard health checking protocol as defined
in [gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md).
The service must use the same proto and implement the heartbeat functionality.

```
Heartbeats will be enabled by default and they cant be disabled.
```

##### Configuration

* **service_heartbeat_type** (mandatory. ```http|grpc```) - though All services must be using grpc, for the
  simplicity of heartbeat implementation, both http and gRPC based heartbeat end points are supported.

* **heartbeat_endpoint** (mandatory ```must be a valid http|https|grpc url```) - It is service heartbeat endpoint.
  Based on the service heart beat type, if the type is http, then heartbeat service endpoint must be a HTTP endpoint,
  similarly, if the type is gRPC, then the heartbeat service must follow the health protocol as mentioned above and the
  gRPC endpoint has to be configured here.

##### Service endpoints

Heartbeat service is exposed from Daemon endpoint itself, but with different
route <b>```{daemon_endpoint}/heartbeat```</b>
It is pull based service i.e. the monitoring service must call this endpoint to get the heartbeat.

GET http://127.0.0.1:8080/heartbeat

Sample heartbeat from the daemon, which contains the service heartbeat as well

```json
{
  "daemonID": "3a4ebeb75eace1857a9133c7a50bdbb841b35de60f78bc43eafe0d204e523dfe",
  "timestamp": "1544916260",
  "status": "Online",
  "serviceheartbeat": "{\"serviceID\":\"sample1\", \"status\":\"SERVING\"}"
}
```

Daemon must call the services as configured in heartbeat_endpoint and the type to get the service heartbeat.
Sample Heartbeat Service result is

GET http://127.0.0.1:25000/heartbeat

```json
{
  "serviceID": "sample1",
  "status": "SERVING"
}
```

### Daemon Monitoring

Each incoming request, outgoing response will be intercepted and the corresponding metrics will be extracted.
The extracted metrics will be reported immediately to the metrics services as configured in the Daemon configuration.
<br/>

<b>Metrics collection is enabled by default</b> in configuration. Service Provider can disable the metrics anytime, if
he/she doesn wanted to collect any metrics.
<br/>

Sample metrics being collected are as below

```json
{
  "request_id": "bggd8ipod0kv0c9a3fs0",
  "input_data_size": 8,
  "content-type": "application/grpc",
  "service_method": "/example_service.Calculator/div",
  "user-agent": "grpc-python/1.17.1 grpc-c/7.0.0 (manylinux; chttp2; gizmo)",
  "request_received_time": "2018-12-24T12:42:51Z",
  "organization_id": "ExampleOrganizationID",
  "service_id": "ExampleServiceID",
  "Group_id": "B6r6a/TvJ36SvOrvyZHxQtDJDYNmWm3Y1/tqhJrKqFM=",
  "Daemon_end_point": "localhost:8080"
}
```

Sample Payload for response stats

```json

{
  "request_id": "bggdcdhod0kv0c9a3ft0",
  "organization_id": "ExampleOrganizationID",
  "service_id": "ExampleServiceID",
  "Group_id": "B6r6a/TvJ36SvOrvyZHxQtDJDYNmWm3Y1/tqhJrKqFM=",
  "Daemon_end_point": "localhost:8080",
  "response_sent_time": "2018-12-24T12:59:51Z",
  "response_time": "23.724177879s",
  "response_code": "OK",
  "error_message": ""
}

```

##### Configuration

* **monitoring_svc_end_point** (optional. ```must be valid http|https url```) - It is the service endpoint to which we
  will have to post all the captured metrics.

* **monitoring_enabled** (optional. ```true|false, default value is true```) - Enables or disables daemon monitoring. By
  default it is set to true. When enabled DAemon captures the request and response metrics and post to the end point.

##### Service endpoint

POST http://127.0.0.1/beta/event

### Alerts/Notifications

It is for alerting the user in case of any issues/errors/warning from the daemon/service, and also pass on some
critical information when needed. Alerts depends on an external webhook or service endpoint o relay the messages to
the configured email address.

##### configuration

* **alerts_email** (optional unless metrics enabled. ```must be a valid email address```) - an email for the
  alerts when there is an issue/warning/error/information to be sent.

* **notification_endpoint** (optional unless metrics enabled. ```must be a valid webhook/service end point```) -
  Its service endpoint or a web hook to receive the alerts/notifications and relay it to alert email, which is
  configured
  the configuration.

##### Service endpoint

POST http://127.0.0.1/beta/notify

Sample Payload

```json
{
  "recipient": "abc.comm@gmail.com",
  "message": "From the API",
  "details": "From the API",
  "component": "daemon",
  "component_id": "ad",
  "type": "INFO",
  "level": "10"
}
```

and the Daemon ID will be added to the request header as Access-Token

```gotemplate
req.Header.Set("Access-Token", daemonID)
```

### Configuration in JSON format

This is the sample configuration to enable metrics and heartbeat

```json
  {
  "monitoring_enabled": true,
  "monitoring_svc_end_point": "http://demo8325345.mockable.io/metrics",
  "notification_endpoint": "http://demo8325345.mockable.io/notify",
  "alerts_email": "xyz.abc@myorg.io",
  "service_heartbeat_type": "http",
  "heartbeat_endpoint": "http://localhost:25000/heartbeat"
}
```
