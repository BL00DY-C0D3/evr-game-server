# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get

# Build targets
BINARY_NAME = evragent
BINARY_NAME_WINDOWS = evragent.exe

all: test build build-windows

build:
	$(GOBUILD) -o $(BINARY_NAME) -v

build-windows:
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME_WINDOWS) -v

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME) $(BINARY_NAME_WINDOWS)

run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./...
	./$(BINARY_NAME)

.PHONY: all build build-windows test clean run