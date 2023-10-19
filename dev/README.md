# Use Docker Compose to develop lineblocs - ari-apps

## Structure of directory
```shell
ari-apps
├── api
├── bitbucket-pipelines.yml
├── dev
|  ├── docker-compose.yaml
|  └── entrypoint.sh
|  └── README.md
|  └── .env
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
asterisk

```

## Simple running
```shell
$ git clone https://github.com/Lineblocs/ari-apps.git
$ cp .env.docker .env
$ cd ari-apps/dev
$ cp .env.example .env
$ docker compose --profile asterisk-cloud up -d
```

## Advance running

### Clone ari-apps project 
Clone docker compose and move to directory.
```shell
$ git clone https://github.com/Lineblocs/ari-apps.git
```

### Make .env file and configure
env file for ari-apps
```shell
$ cp .env.docker .env
```
### Move to dev directory
```
$ cd ari-apps/dev
```

### Make .env file and configure
env file for docker compose and asterisk
```shell
$ cp .env.example .env
```

###  create container
Create and run container with this command below. 

```shell
$ docker compose --profile asterisk-cloud up -d
```

While want to modify asterisk configuration, build asterisk on local. Use profile asterisk-local to do that. Also clone asterisk project, put on same directory with ari-apps project. Create and run container with this command below. 
```shell
$ docker compose --profile asterisk-local up -d
```

### Useful command
Check log  `docker logs -f lineblocs-ari-apps`

Log in to terminal of container  -> `docker exec -it lineblocs-ari-apps bash`

Modify project under `ari-apps` directory

After change configuration, Please run `docker compose restart`
