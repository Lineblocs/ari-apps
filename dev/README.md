# Use Docker Compose to develop lineblocs - ari-apps

## Structure of directory
```shell
.
├── api
├── bitbucket-pipelines.yml
├── dev
|  ├── docker-compose.yaml
|  └── entrypoint.sh
├── Dockerfile
├── entrypoint.sh
├── go.mod
├── go.sum
├── grpc
├── helpers
├── keys
├── lineblocs.proto
├── logger
├── log.txt
├── main
├── main.go
├── mngrs
├── netdiscover
├── README.md
├── router
├── router.proto
├── types
└── utils

```

## Simple running
```shell
$ git clone https://github.com/Lineblocs/ari-apps.git
$ cp .env.docker .env
$ cd ari-apps/dev
$ docker compose up -d
```

## Advance running

### Clone ari-apps project 
Clone docker compose and move to directory.
```shell
$ git clone https://github.com/Lineblocs/ari-apps.git
```

### Make .env file and confige
```shell
$ cp .env.docker .env
```
### Move to dev directory
```
$ cd ari-apps/dev
```

###  create container
Create and run container with this command below. 

```shell
$ docker compose up -d
```

### Useful command
Check log  `docker logs -f lineblocs-ari-apps`

Log in to terminal of container  -> `docker exec -it lineblocs-ari-apps bash`

Modify project under `ari-apps` directory

After change configuration, Please run `docker compose restart`
