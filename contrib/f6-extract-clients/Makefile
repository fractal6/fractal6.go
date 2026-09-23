BIN := build/f6-extract-clients

.PHONY: all windows mac-arm64 mac-intel linux test clean

all: windows mac-arm64 mac-intel linux

windows:
	GOOS=windows GOARCH=amd64 go build -ldflags -H=windowsgui -o $(BIN).exe .

mac-arm64:
	GOOS=darwin GOARCH=arm64 go build -o $(BIN)-mac-arm64 .

mac-intel:
	GOOS=darwin GOARCH=amd64 go build -o $(BIN)-mac-intel .

linux:
	GOOS=linux GOARCH=amd64 go build -o $(BIN)-linux .

test:
	go test .

clean:
	rm -rf build
