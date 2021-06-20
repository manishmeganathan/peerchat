help:
	@echo "PeerChat Makefile (Requires Go v1.16+)"
	@echo "'help' - Displays the command usage"
	@echo "'build' - Builds the application into a binary/executable" 
	@echo "'install' - Installs the application"
	@echo "'build-windows' - Builds the application for Windows platforms"
	@echo "'build-darwin' - Builds the application for MacOSX platforms"
	@echo "'build-linux' - Builds the application for Linux platforms"
	@echo "'build-all' - Builds the application for all platforms"

build:
	@echo Compiling PeerChat
	@go build .
	@echo Compile Complete. Run './peerchat(.exe)'

install:
	@echo Installing PeerChat
	go install .
	@echo install Complete. Run 'peerchat'.

build-windows:
	@echo Cross Compiling PeerChat for Windows x86
	@GOOS=windows GOARCH=386 go build -o ./bin/windows/x32/
	@echo Cross Compiling PeerChat for Windows x64
	@GOOS=windows GOARCH=amd64 go build -o ./bin/windows/x64/

build-darwin:
	@echo Cross Compiling PeerChat for MacOSX x64
	@GOOS=darwin GOARCH=amd64 go build -o ./bin/darwin/x64/

build-linux:
	@echo Cross Compiling PeerChat for Linux x64
	@GOOS=linux GOARCH=386 go build -o ./bin/linux/x32/
	@echo Cross Compiling PeerChat for Linux x64
	@GOOS=linux GOARCH=amd64 go build -o ./bin/linux/x64/
	@echo Cross Compiling PeerChat for Linux Arm32
	@GOOS=linux GOARCH=arm go build -o ./bin/linux/arm32/
	@echo Cross Compiling PeerChat for Linux Arm64
	@GOOS=linux GOARCH=arm64 go build -o ./bin/linux/arm64/

build-all: build-windows build-darwin build-linux
	@echo Cross Compiled PeerChat for all platforms

