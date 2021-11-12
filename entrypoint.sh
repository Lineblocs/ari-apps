#! /bin/bash
# determine proxy host to use

export PROXY_HOST="165.227.35.228"
export ARI_URL="http://${PROXY_HOST}:8088/ari"
export ARI_WSURL="ws://${PROXY_HOST}:8088/ari/events"

./main