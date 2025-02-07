all: clean build run

build:
	CGO_ENABLED=1 go build -o bin/chatapp .
run: 
	bin/chatapp
clean:
	go mod tidy
	rm bin/* || true