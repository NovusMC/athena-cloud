.PHONY: all clean proto plugin master slave format

all: master slave

clean:
	cd plugin && ./gradlew clean
	rm -rf out/
	rm -rf master/assets/

proto:
	cd protobuf && protoc \
		--go_out=../protocol/ \
		--go_opt=paths=source_relative \
		--java_out=../plugin/common/src/main/java/ \
		*.proto

plugin:
	cd plugin && ./gradlew build
	mkdir -p master/assets
	cp plugin/build/athena-velocity.jar plugin/build/athena-paper.jar plugin/build/athena-kotlin-stdlib.jar master/assets

master: plugin
	mkdir -p out/
	go build -o out/master -ldflags "-w -s" -trimpath ./master

slave:
	mkdir -p out/
	go build -o out/slave -ldflags "-w -s" -trimpath ./slave

format:
	cd plugin && ./gradlew ktfmtFormat
	gofmt -w .
