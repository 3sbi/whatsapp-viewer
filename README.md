# WhatsApp Chat Viewer <img src="./assets/icon.svg" alt="logo" width="20"/>

A web interface for viewing your WhatsApp chat history exports. Upload your WhatsApp chat ZIP file and view your archived conversations.

## Quick Start

### Using Docker

```bash
git clone https://github.com/3sbi/whatsapp-viewer.git
cd whatsapp-viewer
docker-compose up -d
open http://localhost:5556
```

### Local development

#### Prerequisites

- Go 1.25+
- Make (optional)

#### Running locally

```bash
git clone https://github.com/3sbi/whatsapp-viewer.git
cd whatsapp-viewer
make run
```

You can see other Make commands using

```bash
make help
```

### Memory management

The application has memory limits for safety. Max Memory: `500MB`.  Very large chat exports may hit the memory limit, consider splitting them into smaller chunks. (Max memory can configure it in `session_store.go` though)
