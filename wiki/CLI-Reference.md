# CLI Reference

## Commands

### `rally run`

Start the proxy server.

```bash
rally run [--config rally.yaml]
```

| Flag | Default | Description |
|---|---|---|
| `--config`, `-c` | `./rally.yaml` | Path to config file |

Config search order:
1. `--config` flag value
2. `./rally.yaml`
3. `/etc/rally.yaml`

### `rally check`

Validate configuration without starting the server.

```bash
rally check [--config rally.yaml]
```

### `rally list`

List all configured VPS backends.

```bash
rally list [--config rally.yaml]
```

### `rally version`

Print version information.

```bash
rally version
```

## Exit Codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | General error |
| 2 | Configuration error |