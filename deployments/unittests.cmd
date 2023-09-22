set gotestroot=R:/
go test -p 1 -coverprofile cover.out ./... -covermode count -v -json
go tool cover -func cover.out