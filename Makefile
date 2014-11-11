all: bindata binaries 

bindata:
	~/bin/go-bindata -pkg=main assets

binaries: linux32 linux64 darwin64 win32

linux32: bindata
	GOOS=linux GOARCH=386 go build -o bin/mytotp32 mytotp.go bindata.go

linux64: bindata
	GOOS=linux GOARCH=amd64 go build -o bin/mytotp mytotp.go bindata.go

darwin64: bindata
	GOOS=darwin GOARCH=amd64 go build -o bin/mytotpOSX mytotp.go bindata.go

win32: bindata
	GOOS=windows GOARCH=386 go build -o bin/mytotp.exe mytotp.go bindata.go

dev: bindata
	go run mytotp.go bindata.go -w -l 0.0.0.0:8000
