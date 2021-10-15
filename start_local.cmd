@echo off
call ./build.cmd
echo starting service 
go-blobstore-service.exe -c ./configs/service_local.yaml
