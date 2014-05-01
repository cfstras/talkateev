GOPATH := $(CURDIR)
export GOPATH

all: build

.PHONY: build
build:
	mkdir -p bin
	go build -o talkateev

.PHONY: clean
clean:
	rm -rf talkateev pkg bin chain.json

run: build start

start:
	./talkateev

deps:
	go get github.com/ChimeraCoder/anaconda
