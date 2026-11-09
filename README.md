# dunebot

This implements a Github Application that can approve and merge Pull Requests.
The major use case is to automatically approve and merge DependeBot Pull Requests.

## Installation

New to DuneBot? Watch the **[Video Tutorial](docs/video-tutorial.md)**
for a full installation walkthrough.

![DuneBot installation walkthrough](docs/assets/install-walkthrough.gif)

A minimal repository configuration (`.github/dunebot.yaml`) looks like this — see
[`example/dunebot.yml`](example/dunebot.yml) for all available options:

```yaml
version: 1.0.0

approve:
  approver: "your-bot-username"
  include:
    - authors:
      - "dependabot[bot]"

merge:
  method: squash
```

## Conflict Comments

When a pull request is not mergeable because of merge conflicts (GitHub
reports `mergeable_state: dirty`), DuneBot can post a configurable comment
instead of attempting the merge. This is useful for Dependabot PRs, where
posting `@dependabot recreate` makes Dependabot recreate the PR and resolve
the conflict automatically.

Configure it per merge branch using the `conflict_comment` field in
`.github/dunebot.yaml`:

```yaml
merge:
  method: squash
  include:
    branches:
      dependabot:
        prefix: "dependabot"
        conflict_comment: "@dependabot recreate"
```

When the branch head matches the configured `prefix` and the PR has merge
conflicts, the `conflict_comment` text is posted as a PR comment. If no
`conflict_comment` is configured for the matching branch, DuneBot falls back
to its default merge-error behaviour.

## Development

### Prerequisites

- `brew install protobuf`
- Then install Go plugins for the protocol compiler by following [This guide](https://grpc.io/docs/languages/go/quickstart/#prerequisites)
- Update your PATH so that the protoc compiler can find the plugins: `export PATH="$PATH:$(go env GOPATH)/bin"`

### Lint

- [Install golangci-lint](https://golangci-lint.run/usage/install/)
- Run `make lint` to lint the code for this service
- consider setting up [editor integration](https://golangci-lint.run/usage/integrations) for quicker feedback.

### Test

Run `make test` to run available tests for this service.

### Build main binary

Run `make build` to build [main.go](main.go) into [build/dunebot](build/dunebot).

### Build docker image

Run `make build-image` to package your [main binary](build/dunebot) in a docker image using the [Dockerfile](Dockerfile).

### Run DuneBot application locally

#### Docker-Compose

Pre-requisites:

- install envtor `go install github.com/fr12k/envtor@latest`

Then you can run the following command to setup the needed environment variables and secrets and start the application.

```bash
teller env | envtor | docker-compose -f - up
```

The location of the application secrets are defined in the `.teller.yml` and the environment variables are defined in the `.env` file.

### Debugging Dunebot application locally

To debug the application locally, change the docker-compose.yaml file to use the `Dockerfile.debug` and run the following command:

```bash
teller env | envtor | docker-compose -f - up
```

This will start the application in debug mode and you can attach your debugger to the port `40000`.
For vscode add the following to the launch configuration.

```json
{
  "name": "Connect to server",
  "type": "go",
  "request": "attach",
  "mode": "remote",
  "substitutePath": [
    {
      "from": "${workspaceFolder}",
      "to": "/app"
    },
    {
      "from": "%{HOME}/go/pkg/mod/",
      "to": "/gomod-cache"
    }
  ],
  "port": 40000,
  "host": "127.0.0.1"
}
```
