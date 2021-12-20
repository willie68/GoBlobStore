@echo off
echo starting service 
go-blobstore-service.exe -c ./configs/service_local_file_s3.yaml
