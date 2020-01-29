#!/bin/bash
W=$(pwd)
while true
do
docker run --rm -it \
	-e VAULT_ADDR=$VAULT_ADDR \
	-e KUBERNETES_SERVICE_HOST=$KUBERNETES_SERVICE_HOST \
	-e KUBERNETES_SERVICE_PORT=$KUBERNETES_SERVICE_PORT \
	-v=$TELEPRESENCE_ROOT/var/run/secrets:/var/run/secrets \
	-v=$W:/home/app \
	alpineapp $1 $2 $3 $4 $5 $6 $7 $8 $9
done
