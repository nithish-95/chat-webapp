all: clean build run

build:
	go build -o bin/chatapp .
run: 
	bin/chatapp
clean:
	go mod tidy
	rm bin/* || true