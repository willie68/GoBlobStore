@echo off
cls
echo ***** generate
pigeon -o pkg/model/query/parser.go build/pigeon/parser.peg
echo ***** test
go.exe test -timeout 180s -run ^TestSingleQParse$ github.com/willie68/GoBlobStore/pkg/model/query -v