#!/bin/bash
set -x
CGO_ENABLED=0
pwd 
go test -p 1 ./... -coverprofile="ut.cover" -covermode count -v -json -bench . 2>&1 | tee "test_report.log" 
