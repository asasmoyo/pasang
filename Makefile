PKG:=github.com/asasmoyo/pasang

.PHONY: build
build:
	go build -o dist/pasang $(PKG)

.PHONY: dist
dist:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dist/pasang-linux_amd64 $(PKG)
	GOOS=linux GOARCH=386 CGO_ENABLED=0 go build -o dist/pasang-linux_i386 $(PKG)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o dist/pasang-mac_amd64 $(PKG)
	GOOS=darwin GOARCH=386 CGO_ENABLED=0 go build -o dist/pasang-mac_i386 $(PKG)
