#!/bin/bash
set -x
execute_tests() {
    (CGO_ENABLED=0 
    go test -p 1 ./... -coverprofile="ut.cover" -covermode count -v -json -bench . 2>&1 | tee "test_report.log" | jq '.' || true) && \
    go-junit-report -parser gojson -in "test_report.log" -out "report.xml" && \
    gocover-cobertura < ut.cover > coverage.xml && \
    gosec -no-fail -fmt=sonarqube -out=gosec.json ./...
}

# call all functions passed to this entrypoint
for i in "$@"; do
    "$i"
done