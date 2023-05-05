#!/bin/bash

cwd=$(pwd)

mkdir -p /tmp/letsencrypt
cd /tmp/letsencrypt
cp /etc/letsencrypt/live/*/*.pem .
openssl pkcs12 -export -out cert.pfx -inkey privkey.pem -in fullchain.pem -certfile chain.pem -passout pass:
openssl pkcs12 -in cert.pfx -out certfile.crt -nokeys -passin pass:
openssl pkcs12 -in cert.pfx -out keyfile.key -nocerts -nodes -passin pass:

cd $cwd
cp /tmp/letsencrypt/certfile.crt /tmp/letsencrypt/keyfile.key .