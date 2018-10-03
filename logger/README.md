# Logger configuration

```snet-daemon``` uses [logrus](https://github.com/sirupsen/logrus) log
library. Logger configuration is a set of properties started from ```log.```
prefix. If configuration file is formatted using JSON then all logger
configuration is one JSON object located in ```log``` field.

# log

## log.level (default: info)

Log level. Possible values are [logrus](https://github.com/sirupsen/logrus) log
levels
* debug
* info
* warn
* error
* fatal * panic 
## log.timezone (default: UTC)

Timezone to format timestamps and log file names. It should be name of the
time.Location, see
[time.LoadLocation](https://golang.org/pkg/time/#LoadLocation).

## log.formatter

Set of properties with ```log.formatter.``` prefix describes logger formatter
configuration.

### log.formatter.type (default: json)

Type of the log formatter. Two types are supported, which correspond to
```logrus``` formatter types, see [logrus
Formatter](https://github.com/sirupsen/logrus#formatters)
* json
* text

## log.output

Set of properties with ```log.output.``` prefix describes logger output
configuration.

### log.output.type (default: file)

Type of the logger output. Two types are supported:
* file - [file-rotatelogs](https://github.com/lestrrat-go/file-rotatelogs)
  output which supports log rotation
* stdout - os.Stdout

### log.output.file_pattern (default: ./snet-daemon.%Y%m%d.log)

Log file name which may include date/time patterns in ```strftime (3)```
format. Time and date in file name are necessary to support log rotation.

### log.output.current_link (default: ./snet-daemon.log)

Link to the latest log file.

### log.output.rotation_time_in_sec (default: 86400 (1 day))

Number of seconds before log rotation happens.

### log.output.max_age_in_sec (default: 604800 (1 week))

Number of seconds since last modification time before log file is removed.

### log.output.rotation_count (default: 0 (disabled))

Max number of rotation files. When number of log files becomes greater then
oldest log file is removed.

## log.hooks (default: [])

List of names of the hooks which will be executed when message with specified
log level appears in log. See [logrus
hooks](https://github.com/sirupsen/logrus#hooks). List contains names of the
hooks and hook configuration can be found by name prefix.  Thus for hook named
```<hook-name>``` properties will start from ```log.<hook-name>.``` prefix.

### log.\<hook-name\>.type (required)

Type of the hook. This type is used to find actual hook implementation. Hook
types supported:
* mail_auth - [logrus_mail](https://github.com/zbindenren/logrus_mail)

### log.\<hook-name\>.levels (required)

List of log levels to trigger the hook. 

### log.\<hook-name\>.config (depends on hook implementation)

Set of properties with ```log.<hook-name>.config``` prefix are passed to the
hook implementation when it is initialized. This list of properties is hook
specific.

### logrus_mail hook

Its configuration should contains all of the properties which are required to
call [NewMailAuthHook method](https://godoc.org/github.com/zbindenren/logrus_mail#NewMailAuthHook)
* application_name
* host
* port
* from
* to
* username
* password

# Default logger configuration in JSON format

```
  "log": {
    "level": "info",
    "timezone": "UTC",
    "formatter": {
      "type": "json"
    },
    "output": {
      "current_link": "./snet-daemon.log",
      "file_pattern": "./snet-daemon.%Y%m%d.log",
      "max_age_in_sec": 604800,
      "rotation_count": 0,
      "rotation_time_in_sec": 86400,
      "type": "file"
    },
    "hooks": []
  }
```

# Adding new hooks implementations

Adding new hook implementation is trivial. You should implement factory method
which inputs hook configuration as [Viper](https://godoc.org/github.com/spf13/viper#Viper)
config and returns new instance of the Hook structure. Then register new hook
type by calling RegisterHookType() function from init() method. 

Please see "mail_auth" hook implementation as example:
* [factory method implementation](https://github.com/singnet/snet-daemon/blob/7b897738b17a21fd105a8a69d4d6841fa5f88dbd/logger/hook.go#L106)
* [registering new hook type](https://github.com/singnet/snet-daemon/blob/7b897738b17a21fd105a8a69d4d6841fa5f88dbd/logger/hook.go#L43)
