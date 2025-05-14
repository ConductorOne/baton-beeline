![Baton Logo](./baton-logo.png)

# `baton-beeline` [![Go Reference](https://pkg.go.dev/badge/github.com/conductorone/baton-beeline.svg)](https://pkg.go.dev/github.com/conductorone/baton-beeline) ![main ci](https://github.com/conductorone/baton-beeline/actions/workflows/main.yaml/badge.svg)

`baton-beeline` is a connector for built using the [Baton SDK](https://github.com/conductorone/baton-sdk).

Check out [Baton](https://github.com/conductorone/baton) to learn more the project in general.

# Getting Started

## Configuration

This connector requires the following configuration:

- `base-url`: The Beeline base URL (default: "https://client.beeline.com")
- `beeline-client-site-id`: The Beeline client site ID
- `beeline-client-id`: The OAuth2 client ID for Beeline API access
- `beeline-client-secret`: The OAuth2 client secret for Beeline API access

You can provide these values as environment variables:

```
export BATON_BASE_URL=https://client.beeline.com
export BATON_BEELINE_CLIENT_SITE_ID=your-site-id
export BATON_BEELINE_CLIENT_ID=your-client-id
export BATON_BEELINE_CLIENT_SECRET=your-client-secret
```

## Installation Options

### Homebrew

```bash
brew install conductorone/baton/baton conductorone/baton/baton-beeline
baton-beeline
baton resources
```

### Docker

```bash
docker run --rm -v $(pwd):/out \
  -e BATON_BASE_URL=https://client.beeline.com \
  -e BATON_BEELINE_CLIENT_SITE_ID=your-site-id \
  -e BATON_BEELINE_CLIENT_ID=your-client-id \
  -e BATON_BEELINE_CLIENT_SECRET=your-client-secret \
  ghcr.io/conductorone/baton-beeline:latest -f "/out/sync.c1z"

docker run --rm -v $(pwd):/out ghcr.io/conductorone/baton:latest -f "/out/sync.c1z" resources
```

### From Source

```bash
go install github.com/conductorone/baton/cmd/baton@main
go install github.com/conductorone/baton-beeline/cmd/baton-beeline@main

baton-beeline
baton resources
```

# Data Model

`baton-beeline` will pull down information about the following resources:
- Users
- Organizations
- Roles

# Contributing, Support and Issues

We started Baton because we were tired of taking screenshots and manually
building spreadsheets. We welcome contributions, and ideas, no matter how
small&mdash;our goal is to make identity and permissions sprawl less painful for
everyone. If you have questions, problems, or ideas: Please open a GitHub Issue!

See [CONTRIBUTING.md](https://github.com/ConductorOne/baton/blob/main/CONTRIBUTING.md) for more details.

# `baton-beeline` Command Line Usage

```
baton-beeline

Usage:
  baton-beeline [flags]
  baton-beeline [command]

Available Commands:
  capabilities       Get connector capabilities
  completion         Generate the autocompletion script for the specified shell
  help               Help about any command

Flags:
      --base-url string              The Beeline base URL ($BATON_BASE_URL) (default "https://client.beeline.com")
      --beeline-client-site-id string The Beeline client site ID ($BATON_BEELINE_CLIENT_SITE_ID)
      --beeline-client-id string     The OAuth2 client ID for Beeline API access ($BATON_BEELINE_CLIENT_ID)
      --beeline-client-secret string The OAuth2 client secret for Beeline API access ($BATON_BEELINE_CLIENT_SECRET)
      --client-id string             The client ID used to authenticate with ConductorOne ($BATON_CLIENT_ID)
      --client-secret string         The client secret used to authenticate with ConductorOne ($BATON_CLIENT_SECRET)
  -f, --file string                  The path to the c1z file to sync with ($BATON_FILE) (default "sync.c1z")
  -h, --help                         help for baton-beeline
      --log-format string            The output format for logs: json, console ($BATON_LOG_FORMAT) (default "json")
      --log-level string             The log level: debug, info, warn, error ($BATON_LOG_LEVEL) (default "info")
  -p, --provisioning                 If this connector supports provisioning, this must be set in order for provisioning actions to be enabled ($BATON_PROVISIONING)
      --ticketing                    This must be set to enable ticketing support ($BATON_TICKETING)
  -v, --version                      version for baton-beeline

Use "baton-beeline [command] --help" for more information about a command.
```
