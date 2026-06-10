BINARY := mcp-vtt
CMD := ./cmd/mcp-vtt
LDFLAGS := -s -w
.PHONY: build build-all clean vet dist dist-darwin-arm64 dist-windows-amd64
VERSION := 0.1.0
DIST_DIR := dist
WHISPER_TAG := v1.8.6
YTDLP_TAG := 2026.06.09
GH_PROXY ?= https://gh-proxy.com/
WHISPER_WIN_URL := $(GH_PROXY)https://github.com/ggml-org/whisper.cpp/releases/download/$(WHISPER_TAG)/whisper-bin-x64.zip
FFMPEG_WIN_URL := $(GH_PROXY)https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip
YTDLP_WIN_URL := $(GH_PROXY)https://github.com/yt-dlp/yt-dlp/releases/download/$(YTDLP_TAG)/yt-dlp.exe

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

build-all:
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-darwin-amd64      $(CMD)
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-darwin-arm64      $(CMD)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-windows-amd64.exe $(CMD)

dist: dist-darwin-arm64 dist-windows-amd64

# --- macOS arm64 ---
dist-darwin-arm64: build
	rm -rf $(DIST_DIR)/mcp-vtt-darwin-arm64
	mkdir -p $(DIST_DIR)/mcp-vtt-darwin-arm64/bin
	mkdir -p $(DIST_DIR)/mcp-vtt-darwin-arm64/models
	# mcp-vtt binary
	cp $(BINARY) $(DIST_DIR)/mcp-vtt-darwin-arm64/mcp-vtt
	# whisper-cli (build from source — no pre-built macOS binary available)
	# whisper-cli: static build from source (Homebrew version has dynamic libs that won't work standalone)
	@if ! which cmake > /dev/null 2>&1; then \
		echo "Installing cmake via Homebrew..."; \
		brew install cmake; \
	fi
	@echo "Building whisper-cli from source (static)..."
	rm -rf /tmp/whisper-cpp-build
	git clone --depth 1 --branch $(WHISPER_TAG) https://github.com/ggml-org/whisper.cpp.git /tmp/whisper-cpp-build
	cd /tmp/whisper-cpp-build && cmake -B build \
		-DCMAKE_BUILD_TYPE=Release \
		-DGGML_BLAS=ON \
		-DGGML_BLAS_VENDOR=Apple \
		-DBUILD_SHARED_LIBS=OFF && \
		cmake --build build --config Release -j
	cp /tmp/whisper-cpp-build/build/bin/whisper-cli $(DIST_DIR)/mcp-vtt-darwin-arm64/bin/whisper-cli
	rm -rf /tmp/whisper-cpp-build
	# ffmpeg & ffprobe (evermeet.cx static builds)
	curl -sL https://evermeet.cx/ffmpeg/getrelease/ffmpeg/zip -o /tmp/ffmpeg-macos.zip
	curl -sL https://evermeet.cx/ffmpeg/getrelease/ffprobe/zip -o /tmp/ffprobe-macos.zip
	unzip -o -j /tmp/ffmpeg-macos.zip -d $(DIST_DIR)/mcp-vtt-darwin-arm64/bin/
	unzip -o -j /tmp/ffprobe-macos.zip -d $(DIST_DIR)/mcp-vtt-darwin-arm64/bin/
	rm -f /tmp/ffmpeg-macos.zip /tmp/ffprobe-macos.zip
	# yt-dlp (macOS standalone binary, no Python needed)
	curl -sL https://github.com/yt-dlp/yt-dlp/releases/download/$(YTDLP_TAG)/yt-dlp_macos \
		-o $(DIST_DIR)/mcp-vtt-darwin-arm64/bin/yt-dlp
	chmod +x $(DIST_DIR)/mcp-vtt-darwin-arm64/bin/whisper-cli
	chmod +x $(DIST_DIR)/mcp-vtt-darwin-arm64/bin/ffmpeg
	chmod +x $(DIST_DIR)/mcp-vtt-darwin-arm64/bin/ffprobe
	chmod +x $(DIST_DIR)/mcp-vtt-darwin-arm64/bin/yt-dlp
	# model
	cp models/ggml-tiny-q5_1.bin $(DIST_DIR)/mcp-vtt-darwin-arm64/models/
	# package
	tar -czf $(DIST_DIR)/mcp-vtt-darwin-arm64.tar.gz -C $(DIST_DIR) mcp-vtt-darwin-arm64
	@echo "Done: $(DIST_DIR)/mcp-vtt-darwin-arm64.tar.gz"

# --- Windows amd64 ---
dist-windows-amd64:
	rm -rf $(DIST_DIR)/mcp-vtt-windows-amd64
	mkdir -p $(DIST_DIR)/mcp-vtt-windows-amd64/bin
	mkdir -p $(DIST_DIR)/mcp-vtt-windows-amd64/models
	# mcp-vtt binary (cross-compile)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/mcp-vtt-windows-amd64/mcp-vtt.exe $(CMD)
	# whisper-cli (pre-built from releases)
	curl -fL --retry 3 --connect-timeout 20 "$(WHISPER_WIN_URL)" -o /tmp/whisper-win-x64.zip
	unzip -o -j /tmp/whisper-win-x64.zip Release/whisper-cli.exe 'Release/*.dll' -d $(DIST_DIR)/mcp-vtt-windows-amd64/bin/
	rm -f /tmp/whisper-win-x64.zip
	# ffmpeg & ffprobe (BtbN static build; faster than gyan.dev from this network)
	curl -fL --retry 3 --connect-timeout 20 "$(FFMPEG_WIN_URL)" -o /tmp/ffmpeg-win.zip
	@mkdir -p /tmp/ffmpeg-win-extract
	unzip -o /tmp/ffmpeg-win.zip -d /tmp/ffmpeg-win-extract
	# find the bin directory inside the extracted folder
	@cp /tmp/ffmpeg-win-extract/ffmpeg-*-win64-gpl/bin/ffmpeg.exe $(DIST_DIR)/mcp-vtt-windows-amd64/bin/
	@cp /tmp/ffmpeg-win-extract/ffmpeg-*-win64-gpl/bin/ffprobe.exe $(DIST_DIR)/mcp-vtt-windows-amd64/bin/
	rm -rf /tmp/ffmpeg-win.zip /tmp/ffmpeg-win-extract
	# yt-dlp (Windows standalone exe)
	curl -fL --retry 3 --connect-timeout 20 "$(YTDLP_WIN_URL)" \
		-o $(DIST_DIR)/mcp-vtt-windows-amd64/bin/yt-dlp.exe
	# model
	cp models/ggml-tiny-q5_1.bin $(DIST_DIR)/mcp-vtt-windows-amd64/models/
	# package
	cd $(DIST_DIR) && zip -r mcp-vtt-windows-amd64.zip mcp-vtt-windows-amd64
	@echo "Done: $(DIST_DIR)/mcp-vtt-windows-amd64.zip"

clean:
	rm -f $(BINARY) $(BINARY)-*
	rm -rf $(DIST_DIR)

vet:
	go vet ./...
