# Logger configuration

```snet-daemon``` uses [logrus](https://github.com/sirupsen/logrus) log
library. Logger configuration is a set of properties started from ```log.```
prefix. If configuration file is formatted using JSON then all logger
configuration is one JSON object located in ```log``` field.

* **log** - log configuration section

  * **level** (default: info) - log level. Possible values are
    [logrus](https://github.com/sirupsen/logrus) log levels
    * debug
    * info
    * warn
    * error
    * fatal
    * panic

  * **timezone** (default: UTC) - timezone to format timestamps and log
    file names. It should be name of the time.Location, see
    [time.LoadLocation](https://golang.org/pkg/time/#LoadLocation).

  * **formatter** - set of properties with ```log.formatter.``` prefix
    describes logger formatter configuration.

    * **type** (default: json) - type of the log formatter. Two types are
      supported, which correspond to ```logrus``` formatter types, see [logrus
      Formatter](https://github.com/sirupsen/logrus#formatters)
      * json
      * text

    * **timestamp_format** (default:  "2006-01-02T15:04:05.999999999Z07:00") -
      timestamp format to use in log lines, standard time.Time formats are
      supported, see [time.Time.Format](https://golang.org/pkg/time/#Time.Format)

  * **output** - set of properties with ```log.output.``` prefix describes
    logger output configuration.

    * **type** (default: file) - type of the logger output. Two types are
      supported:
      * file -
        [file-rotatelogs](https://github.com/lestrrat-go/file-rotatelogs)
        output which supports log rotation
      * stdout - os.Stdout

    * **file_pattern** (default: ./snet-daemon.%Y%m%d.log) - log file name
      which may include date/time patterns in ```strftime (3)``` format. Time
      and date in file name are necessary to support log rotation.

    * **current_link** (default: ./snet-daemon.log) - link to the latest log
      file.

    * **rotation_time_in_sec** (default: 86400 (1 day)) - number of seconds
      before log rotation happens.

    * **max_age_in_sec** (default: 604800 (1 week)) - number of seconds since
      last modification time before log file is removed.

    * **rotation_count** (default: 0 (disabled)) - max number of rotation
      files. When number of log files becomes greater then oldest log file is
      removed.

  * **hooks** (default: []) - list of names of the hooks which will be executed
    when message with specified log level appears in log. See [logrus
    hooks](https://github.com/sirupsen/logrus#hooks). List contains names of
    the hooks and hook configuration can be found by name prefix.  Thus for
    hook named ```<hook-name>``` properties will start from
    ```log.<hook-name>.``` prefix.

  * **```<hook-name>```** - configuration of log hook with `<hook-name>` name

    * **type** (required) - Type of the hook. this type is used to find actual
      hook implementation. Hook types supported:
      * mail_auth - [logrus_mail](https://github.com/zbindenren/logrus_mail)

    * **levels** (required) - list of log levels to trigger the hook. 

    * **config** (depends on hook implementation) - set of properties with
      ```log.<hook-name>.config``` prefix are passed to the hook implementation
      when it is initialized. This list of properties is hook specific.

## logrus_mail hook config

Its configuration should contain all of the properties which are required to
call [NewMailAuthHook method](https://godoc.org/github.com/zbindenren/logrus_mail#NewMailAuthHook)
* application_name
* host
* port
* from
* to
* username
* password

Resulting log configuration using logrus_mail hook:
```json
  "log": {
    ...
    "hooks": [ "send-mail" ],
    "send-mail": {
      "type": "mail_auth",
      "levels": ["Error", "Warn"],
      "config": {
		"application_name": "test-application-name",
		"host": "smtp.gmail.com",
		"port": 587,
		"from": "from-user@gmail.com",
		"to": "to-user@gmail.com",
		"username": "smtp-username",
		"password": "secret"
	  }
    },
  }
```

# Default logger configuration in JSON format

```json
  "log": {
    "formatter": {
      "timestamp_format": "2006-01-02T15:04:05.999999999Z07:00",
      "type": "text"
    },
    "hooks": [],
    "level": "info",
    "output": {
      "current_link": "./snet-daemon.log",
      "file_pattern": "./snet-daemon.%Y%m%d.log",
      "max_age_in_sec": 604800,
      "rotation_count": 0,
      "rotation_time_in_sec": 86400,
      "type": "file"
    },
    "timezone": "UTC"
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
