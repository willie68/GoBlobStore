@echo off
echo building service
docker build -f ./build/package/unittest.Dockerfile --tag mcs/go-blob-store-unit:V1 ./

docker run --name UnitGoBlobStore -v h:/temp/blbstg:/data/storage -p 8443:8443 mcs/go-blob-store-unit:V1
