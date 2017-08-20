VERSION	:=	$(shell cat ./VERSION)

install:
	go install -ldflags "-X main.version=$(VERSION)" -v

image:
	docker build -t cirocosta/sdns:latest .
	docker tag cirocosta/sdns:latest cirocosta/sdns:$(VERSION)

fmt:
	go fmt
	cd ./lib && go fmt
	cd ./util && go fmt

test:
	cd ./lib && go test -v
	cd ./util && go test -v

release: image
	git tag -a $(VERSION) -m "Release" || true
	git push origin $(VERSION)
	goreleaser --rm-dist
	docker push cirocosta/sdns:latest
	docker push cirocosta/sdns:$(VERSION)

.PHONY: fmt install fmt release image
