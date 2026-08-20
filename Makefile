.PHONY: test-arm64 test-pdfraster test-arm64-pdfraster

# Run Go tests in Docker ARM64 container
test-arm64:
	docker run --rm --platform linux/arm64 -v $(PWD):/app -w /app golang:1.26 go test ./...

# Run the PDF-rasterization tests locally. Needs MuPDF available at runtime;
# go-fitz dlopen's libmupdf — install it via the system package manager
# (`brew install mupdf-tools` on macOS, `apt-get install -y libmupdf-dev` on
# Debian/Ubuntu) before running.
test-pdfraster:
	go test -tags pdfraster ./...

# Run the PDF tests under the same Docker arm64 image used for test-arm64,
# with libmupdf installed in the image.
test-arm64-pdfraster:
	docker run --rm --platform linux/arm64 -v $(PWD):/app -w /app golang:1.26 \
		bash -c "apt-get update && apt-get install -y --no-install-recommends libmupdf-dev mupdf && go test -tags pdfraster ./..."
