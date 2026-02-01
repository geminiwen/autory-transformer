# Autory Transformer

A multi-provider API proxy that transforms Anthropic Messages API requests to various LLM providers (BytePlus Ark, Alibaba DashScope) and converts responses back to Anthropic format.

## Features

- ✅ Full Anthropic Messages API compatibility
- ✅ Multiple LLM provider support (Ark, DashScope)
- ✅ Streaming and non-streaming responses
- ✅ Tool calling (Function calling) support
- ✅ Extended Thinking / Reasoning mode support
- ✅ Multimodal support (text + images, PDFs, videos)
- ✅ Dynamic provider selection via HTTP headers
- ✅ System prompt support
- ⚠️ Some limitations (see below)

## Supported Providers

### BytePlus Ark (火山引擎方舟)
- Chat Completions API for text generation
- Responses API for multimodal content (PDF, images, videos)
- Extended thinking support via reasoning models

### Alibaba DashScope (阿里云百炼)
- Text generation API
- Multimodal generation API (images)
- Native thinking/reasoning mode
- Tool calling support

## Quick Start

### 1. Run the Service

```bash
go run main.go
```

The service will start on `http://localhost:10096`.

### 2. Configure via HTTP Headers

The service uses HTTP headers for dynamic configuration (no environment variables needed):

**Common Headers:**
- `Authorization: Bearer <API_KEY>` - Provider API key (required)
- `X-Autory-Provider: <provider>` - Provider selection: `ark` or `dashscope` (optional, defaults to `ark`)
  - Case-insensitive (`ARK`, `Ark`, `ark` all work)
- `anthropic-version: 2023-06-01` - API version (optional)

**Ark-specific Headers:**
- `X-Autory-Ark-Endpoint` - Ark API base URL (required)
  - Example: `https://ark.cn-beijing.volces.com/api/v3`
- `X-Autory-Ark-MultiModal` - Multimodal model for PDF/images/videos (optional)
  - Example: `seed-1-6-250915`

**DashScope-specific Headers:**
- `X-Autory-DashScope-Endpoint` - DashScope API base URL (optional)
  - Default: `https://dashscope.aliyuncs.com/api/v1`
- `X-Autory-DashScope-MultiModal` - Multimodal model for images (optional)
  - Example: `qwen3-vl-plus`

### 3. Example Usage

**Basic Text Request (DashScope):**

```bash
curl http://localhost:10096/v1/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $DASHSCOPE_API_KEY" \
  -H "X-Autory-Provider: dashscope" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "qwen-plus",
    "max_tokens": 1024,
    "messages": [
      {"role": "user", "content": "Hello, how are you?"}
    ]
  }'
```

**Streaming with Thinking Mode (DashScope):**

```bash
curl http://localhost:10096/v1/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $DASHSCOPE_API_KEY" \
  -H "X-Autory-Provider: dashscope" \
  -d '{
    "model": "qwen-plus",
    "max_tokens": 1024,
    "stream": true,
    "thinking": {
      "type": "enabled",
      "budget_tokens": 2000
    },
    "messages": [
      {"role": "user", "content": "Explain quantum entanglement"}
    ]
  }'
```

**Image Understanding (DashScope):**

```bash
curl http://localhost:10096/v1/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $DASHSCOPE_API_KEY" \
  -H "X-Autory-Provider: dashscope" \
  -H "X-Autory-DashScope-MultiModal: qwen3-vl-plus" \
  -d '{
    "model": "qwen3-vl-plus",
    "messages": [{
      "role": "user",
      "content": [
        {
          "type": "image",
          "source": {
            "type": "base64",
            "media_type": "image/jpeg",
            "data": "<base64-encoded-image>"
          }
        },
        {"type": "text", "text": "Describe this image"}
      ]
    }]
  }'
```

**PDF Understanding (Ark):**

```bash
curl http://localhost:10096/v1/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARK_API_KEY" \
  -H "X-Autory-Provider: ark" \
  -H "X-Autory-Ark-Endpoint: https://ark.cn-beijing.volces.com/api/v3" \
  -H "X-Autory-Ark-MultiModal: seed-1-6-250915" \
  -d '{
    "model": "ep-20250201-xxxxx",
    "messages": [{
      "role": "user",
      "content": [
        {
          "type": "document",
          "source": {
            "type": "base64",
            "media_type": "application/pdf",
            "data": "<base64-encoded-pdf>"
          }
        },
        {"type": "text", "text": "Summarize this document"}
      ]
    }]
  }'
```

## Provider-Specific Features

### DashScope

**Text Generation:**
- Model: `qwen-plus`, `qwen-max`, `qwen3-max`, etc.
- Endpoint: `/services/aigc/text-generation/generation`
- Max tokens limit: 8192 (automatically validated)

**Thinking/Reasoning Mode:**
- Native support via `enable_thinking` parameter
- Returns `reasoning_content` field in responses
- Automatically converts to Anthropic `thinking` blocks
- Works with `qwen-plus` and other reasoning models

**Tool Calling:**
- Full support for function calling
- Tools defined in `parameters.tools` (DashScope format)
- Automatic conversion between Anthropic and DashScope formats

**Multimodal (Images):**
- Model: `qwen3-vl-plus`, etc.
- Endpoint: `/services/aigc/multimodal-generation/generation`
- Auto-detection when request contains images
- Format conversion: Anthropic base64 → DashScope data URI

### BytePlus Ark

**Chat Completions:**
- Standard text generation
- Model specified via endpoint ID

**Responses API (Multimodal):**
- PDF documents (`document` type)
- Images (`image` type)
- Videos (`video` type)
- Base64 format, max 50MB

**Extended Thinking:**
- Via reasoning models (e.g., DeepSeek R1)
- Native `reasoning_content` support

## API Endpoints

### POST /v1/messages

Anthropic Messages API compatible endpoint.

**Supported Parameters:**
- `model` - Model name or endpoint ID
- `messages` - Message list
- `max_tokens` - Maximum output tokens
- `system` - System prompt
- `temperature` - Temperature (0-1)
- `top_p` - Top-p sampling
- `stop_sequences` - Stop sequences
- `stream` - Enable streaming
- `tools` - Tool definitions
- `thinking` - Extended thinking configuration

### GET /health

Health check endpoint, returns `{"status": "ok"}`.

## Architecture

```
.
├── main.go                    # Application entry point
├── config/
│   └── config.go             # Configuration
├── adapter/
│   ├── ark/                  # BytePlus Ark adapter
│   │   ├── types.go         # Type definitions
│   │   ├── client.go        # HTTP client
│   │   ├── request_transformer.go
│   │   ├── response_transformer.go
│   │   └── stream_transformer.go
│   └── dashscope/           # Alibaba DashScope adapter
│       ├── types.go
│       ├── client.go
│       ├── request_transformer.go
│       ├── response_transformer.go
│       └── stream_transformer.go
├── internal/
│   ├── types/
│   │   └── anthropic.go     # Anthropic API types
│   ├── handler/
│   │   └── messages.go      # Request handler & router
│   └── errors/
│       └── errors.go        # Error definitions
└── README.md
```

## Provider Selection Logic

The service routes requests based on the `X-Autory-Provider` header (case-insensitive):

1. **ark** → BytePlus Ark adapter
   - Requires `X-Autory-Ark-Endpoint` header
   - Detects multimodal content (PDF/images/videos)
   - Switches to Responses API when needed

2. **dashscope** → Alibaba DashScope adapter
   - Optional `X-Autory-DashScope-Endpoint` header
   - Detects image content
   - Switches to multimodal API when needed

## Limitations

### Unsupported Features

The following features return 400 errors:

1. **Structured Output** (`output_config` parameter)
   - Not supported by provider APIs

2. **DashScope-specific:**
   - Documents and videos (only images supported)
   - Max tokens > 8192 (automatically skipped with warning)

### Silently Ignored Features

1. **Prompt Caching** (`cache_control`)
   - Providers handle caching automatically

2. **Citations**
   - Not supported by providers

## Development

### Build

```bash
go build ./...
```

### Run Tests

```bash
go test ./...
```

### Docker

```bash
# Build
docker build -t autory-transformer .

# Run
docker run -p 10096:10096 autory-transformer
```

## Tech Stack

- **Go 1.22+** - Programming language
- **Hertz** - High-performance HTTP framework (ByteDance)
- **Direct HTTP clients** - No SDK dependencies
- **SSE** - Server-Sent Events for streaming

## License

MIT License

## Contributing

Issues and Pull Requests are welcome!
