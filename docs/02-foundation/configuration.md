# Configuration and environment

## Environment (`foundation.Env`)

`framework.WithEnvFile(".env")` loads `.env` files into a `foundation.Env`.
The **process environment always wins** over file values, so real deployment
variables override local files. Missing files are skipped; malformed files
are reported with file name and line number. The process environment is
never modified.

Supported `.env` syntax:

```dotenv
# comments and blank lines are ignored
APP_NAME=Shop
export APP_ENV=local          # optional "export" prefix
APP_DEBUG=true                # trailing " # comment" is stripped
GREETING="Hello\nWorld"       # double quotes: \n \r \t \" \\ escapes
RAW='no $escapes \n here'     # single quotes: literal
```

Values must fit on one line, and `${VAR}` interpolation is intentionally not
supported.

Typed accessors return an error for unparsable values instead of silently
using the default:

```go
env := app.Env()
host := env.String("DB_HOST", "localhost")
port, err := env.Int("DB_PORT", 5432)
debug, err := env.Bool("APP_DEBUG", false)      // true/false/1/0/yes/no/on/off
timeout, err := env.Duration("TIMEOUT", 30*time.Second)
```

## Configuration (`foundation.Config`)

`app.Config()` is a concurrency-safe key/value store using dot-notation keys.
The framework sets these keys from the environment:

| Key | Variable | Default |
| --- | --- | --- |
| `app.name` | `APP_NAME` | `goPanglima` |
| `app.env` | `APP_ENV` | `production` |
| `app.debug` | `APP_DEBUG` | `false` |
| `log.level` | `LOG_LEVEL` | `info` (`debug`, `info`, `warn`, `error`) |
| `log.format` | `LOG_FORMAT` | `text` (`text`, `json`) |

The default environment is `production`, so a missing configuration fails
safe. **Never set `APP_DEBUG=true` in production**: debug mode adds internal
error details to 5xx responses.

Add your own values with `framework.WithConfig` or `Config().Set`:

```go
app := framework.New(
	framework.WithEnvFile(".env"),
	framework.WithConfig(map[string]any{"mail.from": "noreply@example.com"}),
)

from := app.Config().String("mail.from", "")
limit := app.Config().Int("api.rate_limit", 60)
```

Values keep their Go type. For example `app.debug` must be a `bool`:
`WithConfig(map[string]any{"app.debug": "true"})` stores a string and leaves
debug mode off. Use `true`, or set `APP_DEBUG=true` in the environment.

The convenience getters (`String`, `Int`, `Int64`, `Bool`, `Duration`) return
the default when a key is missing **or holds a different type**. When that
must be an error, use the generic strict lookup:

```go
port, err := foundation.Lookup[int](app.Config(), "db.port")
// errors.Is(err, foundation.ErrConfigNotFound) when missing
```

Related: [Lifecycle](lifecycle.md).
