@echo off
echo building service
docker build --tag mcs/go-blob-store:V1 ./

docker run --name GoBlobStore -v h:/temp/blbstg:/data/storage -p 8443:8443 mcs/go-blob-store:V1
