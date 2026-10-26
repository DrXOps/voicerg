.PHONY: all build-linux build-windows-docker test clean

# Сборка под текущую ОС (Linux: test, Windows: production)
all: build-linux

build-linux:
	CGO_ENABLED=0 go build -o bin/vasya-linux ./cmd/protocol-assistant

# Windows-сборка через Docker с mingw-w64 (CGO для malgo/vosk).
build-windows-docker:
	docker run --rm \
		-v "$(PWD)":/src \
		-w /src \
		-e GOOS=windows \
		-e GOARCH=amd64 \
		-e CGO_ENABLED=1 \
		-e CC=x86_64-w64-mingw32-gcc \
		-e GOCACHE=/tmp/gocache \
		vasya-winbuild \
		go build -buildvcs=false -ldflags "-H=windowsgui -s -w" \
			-o bin/protocol-voice-assistant.exe ./cmd/protocol-assistant
	@echo ""
	@file bin/protocol-voice-assistant.exe || true

# Windows-сборка с debug-символами (для отладки).
build-windows-docker-debug:
	docker run --rm \
		-v "$(PWD)":/src \
		-w /src \
		-e GOOS=windows \
		-e GOARCH=amd64 \
		-e CGO_ENABLED=1 \
		-e CC=x86_64-w64-mingw32-gcc \
		-e GOCACHE=/tmp/gocache \
		vasya-winbuild \
		go build -buildvcs=false -ldflags "-H=windowsgui" \
			-o bin/protocol-voice-assistant-debug.exe ./cmd/protocol-assistant

# Docker-образ для кросс-компиляции.
docker-image:
	docker build -t vasya-winbuild -f Dockerfile.win .

test:
	go test -count=1 -v ./internal/... 2>&1 | grep -E '^(=== RUN|--- PASS|--- FAIL|FAIL|ok)' || true

clean:
	rm -rf bin/ protocol-voice-assistant.exe

full: clean docker-image build-linux build-windows-docker test