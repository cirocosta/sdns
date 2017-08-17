install:
	go install -v

fmt:
	go fmt
	cd ./lib && go fmt
	cd ./util && go fmt

test:
	cd ./lib && go test -v
	cd ./util && go test -v

.PHONY: fmt install fmt
