@echo off
echo building service
docker build --tag go-blob-store ./

docker run --name GoBlobStore -v h:/temp/blbstg:/data/storage -p 8443:8443 go-blob-store
