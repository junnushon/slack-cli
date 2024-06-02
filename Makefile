# Makefile

# Define the output filenames for each OS/ARCH combination
OUTPUT_WINDOWS=slack.exe
OUTPUT_DARWIN=slack-darwin
OUTPUT_LINUX=slack

# Define the build command for each OS/ARCH combination
build: build-linux build-windows build-darwin

build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build -ldflags "-X 'main.buildTime=$(shell date '+%Y-%m-%d %H:%M:%S')'" -o $(OUTPUT_LINUX) main.go

build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 go build -ldflags "-X 'main.buildTime=$(shell date '+%Y-%m-%d %H:%M:%S')'" -o $(OUTPUT_WINDOWS) main.go

build-darwin:
	@echo "Building for Darwin..."
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X 'main.buildTime=$(shell date '+%Y-%m-%d %H:%M:%S')'" -o $(OUTPUT_DARWIN) main.go

# Define the all target to build for all OS/ARCH combinations
all: build-windows build-darwin build-linux

# Clean up the build artifacts
clean:
	rm -f $(OUTPUT_WINDOWS) $(OUTPUT_DARWIN) $(OUTPUT_LINUX)

.PHONY: all clean build-windows build-darwin build-linux
