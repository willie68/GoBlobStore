@echo off
echo building service
docker build --tag go-blob-store ./

docker run --name GoBlobStore -v h:/temp/blbstg:/data/storage -p go-blob-store