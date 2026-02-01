.PHONY: build run dev help

# Default target
help:
	@echo "Available targets:"
	@echo "  build    - Build the WhatsApp Chat Viewer application"
	@echo "  run      - Build and run the application with default settings"
	@echo "  dev      - Build and run the application in debug mode"
	@echo "  help     - Show this help message"

# Build the application
build:
	@echo "Building WhatsApp Viewer..."
	go build -o whatsapp-viewer .
	@echo "Build complete!"

# Run the application
run: build
	@echo "Starting WhatsApp Viewer (production mode)..."
	./whatsapp-viewer

# Run in debug mode
dev: build
	@echo "Starting WhatsApp Viewer (debug mode)..."
	LOG_LEVEL=debug ./whatsapp-viewer
