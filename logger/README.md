# Logger configuration

The daemon uses [Zap](https://github.com/uber-go/zap) with optional rotating-file,
email, and Telegram outputs. Settings live under the `log` object. The complete
key reference is in [config/README.md](../config/README.md#logging).

## Levels and formatting

| Key under `log` | Default | Behavior |
|-----------------|---------|----------|
| `level` | `"info"` | `debug`, `info`, `warn` (`warning` alias), `error`, `panic`, or `fatal`. |
| `timezone` | `"UTC"` | Go time-zone name for log-entry timestamps. |
| `formatter.type` | `"text"` | `text` or `json`. |
| `formatter.timestamp_format` | `"2006-01-02T15:04:05.999Z07:00"` | Go time layout, not strftime. Explicit `""` selects RFC3339. |

`panic` and `fatal` records always go to stderr and all configured outputs,
regardless of `log.level`. This includes file-only logging: the complete record,
including structured context, also goes to stderr using the configured formatter.
If stderr is already selected, it is not added a second time. Other levels obey
the configured threshold and outputs.

Invalid core settings, such as an unknown formatter or output type, prevent logger
initialization. Invalid optional hooks are reported and skipped.

## Outputs and rotation

| Key under `log.output` | Default | Behavior |
|------------------------|---------|----------|
| `type` | `["file", "stdout"]` | Nonempty list containing `file`, `stdout`, and/or `stderr`. |
| `file_pattern` | `"./snet-daemon.%Y%m%d.log"` | File name evaluated once during logger initialization. |
| `current_link` | `"./snet-daemon.log"` | Symlink to the active file. Set `""` to disable creation. |
| `max_size_in_mb` | `10` | Size-based rotation threshold in MiB; `0` uses Lumberjack's 100 MiB fallback. |
| `max_age_in_days` | `7` | Retain rotated backups according to their encoded rotation timestamp, not filesystem modification time. `0` removes the age limit. |
| `rotation_count` | `0` | Maximum retained backup count. `0` removes this limit; size-based rotation and age cleanup still apply. |

The file-name formatter supports only `%Y`, `%m`, `%d`, `%H`, `%M`, and
`%S`. A fixed name such as `snet-daemon.log` also supports rotation; a date in
the pattern is not required and does not schedule midnight rotation.

The initial file name uses the process's local time. Backup timestamps use
Lumberjack's UTC default. `log.timezone` changes entry timestamps, not those
file-name clocks. Rotated files are gzip-compressed; the daemon does not expose
a compression switch. See [Lumberjack's retention rules](https://github.com/natefinch/lumberjack/tree/v2.2.1#cleaning-up-old-log-files).

Paths are relative to the process working directory. File logging requires write
permissions, and `current_link` additionally requires permission to create
symlinks. Use separate paths per daemon process. In containers, select stdout
or mount a writable log directory; a read-only config mount is not a log directory.

## Hooks

`log.hooks` defaults to `[]`. Each entry names a sibling object under `log`;
for example, `"hooks": ["mail"]` reads `log.mail`.

Every hook requires:

- `type`: `email` or `telegram_bot`.
- `levels`: a nonempty array of exact levels, such as `["warn", "error"]`.
  These are not minimum-severity thresholds. Ordinary messages filtered out by
  `log.level` do not reach hooks.

### Email hook

| Field | Requirement |
|-------|-------------|
| `host` | Required SMTP hostname. |
| `port` | Required, nonzero SMTP port. |
| `from` | Required sender address. |
| `to` | Required recipient address. |
| `username` | Optional; omitted/empty uses `from`. |
| `password` | Required SMTP credential. Use a provider-specific app password where appropriate. |
| `application_name` | Optional SMTP authentication identity; default `""`. |

### Telegram hook

| Field | Requirement |
|-------|-------------|
| `telegram_api_key` | Required bot token. |
| `telegram_chat_id` | Required, nonzero integer chat ID; group IDs may be negative. |
| `telegram_disable_notifications` | Optional, default `false`. Set `true` for silent messages. |

The config key is `telegram_disable_notifications`, not Telegram's outbound
JSON field `disable_notification`. Do not commit real bot tokens or SMTP passwords.

## Examples

Each block is a valid JSON fragment to merge into the daemon's top-level config.
Credential placeholders must be replaced securely before enabling a hook.

### JSON logs to stdout

```json
{
  "log": {
    "formatter": { "type": "json" },
    "output": { "type": ["stdout"] }
  }
}
```

### Text logs with size-based rotation

The directory must exist and be writable by the daemon user.

```json
{
  "log": {
    "level": "info",
    "timezone": "UTC",
    "formatter": {
      "type": "text",
      "timestamp_format": "2006-01-02T15:04:05.999Z07:00"
    },
    "output": {
      "type": ["file"],
      "file_pattern": "/var/log/snet/daemon.log",
      "current_link": "",
      "max_size_in_mb": 100,
      "max_age_in_days": 7,
      "rotation_count": 5
    }
  }
}
```

### Email hook

```json
{
  "log": {
    "level": "info",
    "hooks": ["mail"],
    "mail": {
      "type": "email",
      "levels": ["warn", "error", "panic", "fatal"],
      "host": "smtp.example.com",
      "port": 587,
      "from": "daemon@example.com",
      "to": "operator@example.com",
      "username": "daemon@example.com",
      "password": "<SMTP_PASSWORD>"
    }
  }
}
```

### Telegram hook

```json
{
  "log": {
    "hooks": ["telegram"],
    "telegram": {
      "type": "telegram_bot",
      "levels": ["warn", "error", "panic", "fatal"],
      "telegram_api_key": "<TELEGRAM_BOT_TOKEN>",
      "telegram_chat_id": -1001234567890,
      "telegram_disable_notifications": true
    }
  }
}
```

## Adding a hook implementation

Implement the package's `hook` interface with `call(zapcore.Entry) error`, provide
a constructor accepting `*viper.Viper`, and register it with `RegisterHookType`.
See [hook.go](hook.go) and its tests. Hooks are distinct from the heartbeat
notification endpoint described in [metrics](../metrics/README.md).
