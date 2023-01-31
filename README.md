[![Alt text](https://github.com/Lineblocs/ari-apps/actions/workflows/ci.yml/badge.svg)](https://github.com/Lineblocs/ari-apps/actions/workflows/ci.yml/badge.svg)
```

export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
protoc --go_out=plugins=grpc:. *.proto
protoc --go-grpc_out=grpc lineblocs.proto
```

## Prerequisites

Minimum software required to run the service:
* [go](https://go.dev/doc/install)

## Clone repository

```bash
git clone https://github.com/Lineblocs/ari-apps.git
```

## Structure of Code

1. api
   includes api functions which are connected to internals-api
2. grpc
   includes protobuf files
3. mngrs
   includes managing files  
4. types
   includes basic model types files
5. utils
   includes common utils functions

## Testing

### Unit test with builtin Testing package

```bash
cd types
go test -v
```

## Debugging

### Configure log channels
Debugging issues by tracking logs

There are 4 log channels including console, file, cloudwatch, logstash
Set LOG_DESTINATIONS variable in .env file

ex: export LOG_DESTINATIONS=file,cloudwatch

## Linting and pre-comit hook

### Go lint
```bash
sudo snap install golangci-lint
```
Config .golangci.yaml file to add or remote lint options

### pre-commit hook
```bash
sudo snap install pre-commit --classic
```
Config .pre-commit-config.yaml file to enable or disable pre-commit hook

## Deploy

### Deploy Steps
1. Install Docker on the machines you want to use it;
2. Set up a registry at Docker Hub;
3. Initiate Docker build to create your Docker Image;
4. Set up your ’Dockerized‘ machines;
5. Deploy your built image or application.

### Deploy Command

```bash
docker build -t ari-apps
```
