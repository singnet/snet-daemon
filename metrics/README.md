# Daemon Metrics & Monitoring Services

```snet-daemon``` sends heartbeat upon the request from a monitoring service or any other component.
Apart from heartbeat, it will expose few custom metrics as mentioned below.
<br/>
The critical daemon updates, warning, or errors can be reported to a pre-configured email using a default 
notification service offered by SingularityNet. Its just an optional service. Service providers are free to choose thier own endpoints/web hooks for sending alerts.

### Heartbeat

#### Configuration

### Metrics  
Each incoming request, outgoing response will be intercepted and the corresponding metrics will be extracted.
The extracted metrics will be reported immediately to the metrics services as configured in the Daemon configuration.
<br/>

<b>Metrics collection is enabled by default</b> in configuration. Service Provider can disable the metrics anytime, if he/she doesn wanted to collect any metrics.
<br/>

The metrics being collected are
   * **No of Requests** - number of requests received by the service endpoint.
   
   * **No of Successful/Failed Requests** - number of requests successfully handled and number fo requesed rejected/failed. Reason for failures could be
   anything like service errors, rate limited requests, etc.
   
   * **Request Size** 
   
   * **Request Content Type**
   
   * **No of Responses** - number of responses returned by the service endpoint.
   
   * **No of  Successful/Failed Responses** - number of successful/failed responses.
   
   * **Response Size**
   
   * **Response Time**
   

#### Configuration

#### API service endpoints
   
### Alerts/Notifications
It is for alerting the user in case of any issues/errors/warning from the daemon/service, and also pass on some
critical information when needed. Alerts depends on an external webhook or service endpoint o relay the messages to
the configured email address. 

#### configuration  
   * **alert_email** (optional unless metrics enabled. ```must be a valid email```) - an email for the 
   alerts when there is an issue/warning/error/information to be sent. 
   
   * **notification_svc_end_point** (optional unless metrics enabled. ```must be a valid webhook/service end point```) - 
   Its service endpoint or a web hook to receive the alerts/notifications and relay it to alert email, which is configured
   the configuration.
   
#### API service endPoints
    
### Configuration in JSON format
This is the sample configuration to enable metrics and heartbeat
```json
  {
      "enable_metrics"            : true,
      "monitoring_svc_end_point"  : "http://demo3208027.mockable.io",
      "notification_svc_end_point": "http://demo3208027.mockable.io/notify",
      "alert_email"               : "xyz.abc@myorg.io",
      "service_heartbeat_type"    : "http",
      "heartbeat_svc_end_point"   : "http://demo3208027.mockable.io/heartbeat"  
   }
```
