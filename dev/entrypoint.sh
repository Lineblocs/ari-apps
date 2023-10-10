#! /bin/bash
# determine proxy host to use

if [[ -z "${ARI_URL}" ]]; then
    : ${CLOUD=""} # One of aws, azure, do, gcp, or empty
    if [ "$CLOUD" != "" ]; then
        PROVIDER="-provider ${CLOUD}"
    fi

    PRIVATE_IPV4=$(netdiscover -field privatev4 ${PROVIDER})
    #PRIVATE_IPV4="172.24.0.1"
    PUBLIC_IPV4=$(netdiscover -field publicv4 ${PROVIDER})

    export PROXY_HOST=${PUBLIC_IPV4}
    export ARI_URL="http://${PROXY_HOST}:8088/ari"
    export ARI_WSURL="ws://${PROXY_HOST}:8088/ari/events"
fi

cd /app
go mod download

go build -o main main.go

./main