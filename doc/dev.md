getting all dependencies, which can be updated

`go list -u -m -json all | go-mod-outdated -update -direct`