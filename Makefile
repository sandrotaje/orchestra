.PHONY: build run install clean

build:
	go build -o bin/orchestra cmd/orchestra/main.go

run: build
	./bin/orchestra

install: build
	cp bin/orchestra ~/.local/bin/orchestra

clean:
	rm -rf bin/
