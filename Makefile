.PHONY: build build-all clean vet
BINARY := mcp-vtt
CMD := ./cmd/my-vtt
LDFLAGS := -s -w

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

build-all:
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-darwin-amd64      $(CMD)
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-darwin-arm64      $(CMD)
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-linux-amd64       $(CMD)
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-linux-arm64       $(CMD)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-windows-amd64.exe $(CMD)

clean:
	rm -f $(BINARY) $(BINARY)-*

vet:
	go vet ./...
