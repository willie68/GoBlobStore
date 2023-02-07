set gotestroot=R:/
go test -p 1 ./... -coverprofile="ut.cover" -covermode count -v -json -bench