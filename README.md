[![Alt text](https://github.com/Lineblocs/ari-apps/actions/workflows/ci.yml/badge.svg)](https://github.com/Lineblocs/ari-apps/actions/workflows/ci.yml/badge.svg)
```

export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
protoc --go_out=plugins=grpc:. *.proto
protoc --go-grpc_out=grpc lineblocs.proto
```