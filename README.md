# dunebot

This implements a Github Application that can approve and merge Pull Requests.
The major use case is to automatically approve and merge DependeBot Pull Requests.

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
