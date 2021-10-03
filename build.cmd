@echo off
echo building service
go build -ldflags="-s -w" -o go-blobstore-service.exe cmd/service/main.go
