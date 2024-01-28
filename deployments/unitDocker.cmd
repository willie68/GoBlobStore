@echo off
echo building service
docker build -f ./build/package/unittest.Dockerfile --tag go-blob-store-unit:V1 ./

docker run --name UnitGoBlobStore -v h:/temp/blbstg:/data/storage -p 8443:8443 go-blob-store-unit:V1
docker cp UnitGoBlobStore:/src/test_report.log  ./
docker rm -f UnitGoBlobStore
docker rmi -f go-blob-store-unit:V1