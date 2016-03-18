.PHONY:

GO_SOURCES=$(shell find . -name \*.go)
SOURCES=$(GO_SOURCES)
PLATFORM_BINARIES=dist/ec2cluster.Linux.x86_64

IMAGE_NAME=crewjam/ec2cluster
GITHUB_USER=crewjam
GITHUB_REPOSITORY=ec2cluster

all: $(PLATFORM_BINARIES)

clean:
	-rm $(PLATFORM_BINARIES)

cacert.pem:
	curl -o $@ https://curl.haxx.se/ca/cacert.pem

dist/ec2cluster.Linux.x86_64: $(SOURCES)
	[ -d dist ] || mkdir dist
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags '-s' \
	  -o $@ ./cmd/ec2cluster/main.go

container: cacert.pem dist/ec2cluster.Linux.x86_64
	docker build -t $(IMAGE_NAME) .

check:
	go test ./...

lint:
	go fmt ./...
	goimports -w $(GO_SOURCES)

release: lint check container $(PLATFORM_BINARIES)
	@[ ! -z "$(VERSION)" ] || (echo "you must specify the VERSION"; false)
	which ghr >/dev/null || go get github.com/tcnksm/ghr
	ghr -u $(GITHUB_USER) -r $(GITHUB_REPOSITORY) --delete v$(VERSION) dist/
	docker tag $(IMAGE_NAME) $(IMAGE_NAME):$(VERSION)
	docker push $(IMAGE_NAME)
	docker push $(IMAGE_NAME):$(VERSION)