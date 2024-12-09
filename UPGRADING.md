# Upgrading AtomOne

This guide provides instructions for upgrading to specific versions of AtomOne.

## Go version bump

AtomOne v2 build requires a more recent version of the Go compiler: 1.22.10. If
you already have go installed but with an other version, you can install
go1.22.10 with the following command:

```sh
$ go install golang.org/dl/go1.22.10@latest
$ go1.22.10 download
```

Then you need to update some env variables to invoke the makefile commands of
AtomOne. For example, to run `make build` :
```
$ GOROOT=$(go1.22.10 env GOROOT) PATH=$GOROOT/bin:$PATH make build
```
