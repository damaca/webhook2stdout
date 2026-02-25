# webhook2stdout

A configurable Fiber webhook service that prints received request data to stdout.

## Features

- Accepts any HTTP method on a configurable route
- Prints JSON output to stdout
- Configurable field mapping from request sources to output keys
- Supports YAML or JSON configuration

## Run

```bash
go mod tidy
go run . -config config.yaml
```

Requires Go 1.25+.

If the config file is missing, sensible defaults are used.

## Example request

```bash
curl -X POST "http://localhost:8080/webhook?source=test" \
  -H "Content-Type: application/json" \
  -H "X-Trace-Id: abc123" \
  -d '{"event":"build.finished"}'
```

With the provided `config.yaml`, stdout will include keys like `payload` and `headers_received`.

## Makefile

```bash
make tidy
make fmt
make test
make build
make run
```

Docker-related targets:

```bash
make docker-build GHCR_IMAGE=ghcr.io/<owner>/webhook2stdout DOCKERHUB_IMAGE=<dockerhub-user>/webhook2stdout TAG=latest
make docker-push GHCR_IMAGE=ghcr.io/<owner>/webhook2stdout DOCKERHUB_IMAGE=<dockerhub-user>/webhook2stdout TAG=latest
# alias:
make docker-publish GHCR_IMAGE=ghcr.io/<owner>/webhook2stdout DOCKERHUB_IMAGE=<dockerhub-user>/webhook2stdout TAG=latest
```

## Docker

Build and run directly:

```bash
docker build -t webhook2stdout:latest .
docker run --rm -p 8080:8080 webhook2stdout:latest
```

The container defaults to loading `/app/config.yaml`.

## Configuration

### Top-level fields

- `port` (int): server port
- `route` (string): endpoint path (must start with `/`)
- `pretty` (bool): pretty-print JSON to stdout
- `ack_status` (int): HTTP status returned to caller
- `ack_body` (object): JSON body returned to caller
- `mappings` (list): mappings from request source to output key

### Supported mapping sources (`from`)

- `body`
- `headers`
- `query`
- `params`
- `method`
- `path`
- `ip`

### Mapping example

```yaml
mappings:
  - from: body
    root: true
  - from: headers
    to: headers_received
```

`root: true` merges an object source directly into the top-level output JSON.

Example: if body is `{"event":"build.finished","id":"123"}`, output root gets `event` and `id` directly.

Rules:

- A mapping must set either `to` or `root: true`
- A mapping cannot set both
- `root: true` only works when source resolves to an object
- Root merge fails on key collisions

Example with named fields only:

```yaml
mappings:
  - from: body
    to: payload
  - from: headers
    to: headers_received
```

This maps the request body to `payload` and headers to `headers_received` in stdout output.

## GitHub Actions

Workflows are included for:

- CI on pull requests targeting `master` and `release/*`
- CI on pushes to `master` and `release/*`
- Docker image build on PRs
- Docker image build and publish to GHCR and Docker Hub on pushes to `master`, `release/*`, and `v*` tags

Required GitHub secrets for Docker Hub publishing:

- `DOCKERHUB_USERNAME`
- `DOCKERHUB_TOKEN`

Docker tag strategy in workflow:

- Branch `release/vx.y.z` -> tag `vx.y.z`
- Git tag `vx.y.z` -> tag `vx.y.z`
- Any other branch -> tag `sha-<short-commit-sha>`
