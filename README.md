# Anthropic Messages API → 火山引擎方舟 转发代理服务

这是一个 API 代理服务,接收 Anthropic Messages API 格式的请求,转换后转发到火山引擎方舟 API,并将响应转换回 Anthropic 格式返回。

## 功能特性

- ✅ 完整的 Anthropic Messages API 兼容
- ✅ 流式和非流式响应支持
- ✅ 工具调用(Tool Use)支持
- ✅ 多模态支持(文本 + 图像)
- ✅ System prompt 支持
- ✅ Extended Thinking 支持(通过 Prompt Engineering 模拟)
- ⚠️ 部分功能限制(见下文)

## 快速开始

### 1. 配置环境变量

复制 `.env.example` 到 `.env` 并填写你的配置:

```bash
cp .env.example .env
```

编辑 `.env`:

```bash
# 火山方舟基础 URL（国际版）
ARK_BASE_URL=https://ark-ap-southeast.byteintl.net/api/v3

# 配置你的火山方舟 Endpoint ID
# DeepSeek R1 推理模型（支持 Extended Thinking）
ARK_ENDPOINT_THINKING=ep-20250424174745-w6pgh   # 推理模型，thinking 参数时使用
ARK_ENDPOINT_OPUS=ep-xxxxxxxxxx-xxxxx           # 可选，高性能模型，未配置时使用 THINKING
ARK_ENDPOINT_FAST=ep-xxxxxxxxxx-xxxxx           # 可选，快速模型（Haiku），未配置时使用 DEFAULT
ARK_ENDPOINT_DEFAULT=ep-20250725172730-pl88c    # 默认模型（不支持推理的标准模型）

# 服务端口
PORT=3000
```

> **💡 提示**:
> - `ARK_ENDPOINT_THINKING`: DeepSeek R1 推理模型，支持 Extended Thinking
> - `ARK_ENDPOINT_DEFAULT`: 普通模型，用于标准对话

### 2. 安装依赖

```bash
go mod download
```

### 3. 运行服务

```bash
go run main.go
```

服务将在 `http://localhost:3000` 启动。

### 4. 使用 Anthropic SDK 测试

```python
from anthropic import Anthropic

client = Anthropic(
    api_key="your-volcengine-ark-api-key",  # 使用你的火山方舟 API Key
    base_url="http://localhost:3000"
)

message = client.messages.create(
    model="claude-sonnet-4-5-20250929",
    max_tokens=1024,
    messages=[
        {"role": "user", "content": "Hello, how are you?"}
    ]
)

print(message.content[0].text)
```

### 5. 使用 curl 测试

**非流式请求:**

```bash
curl http://localhost:3000/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: your-ark-api-key" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-sonnet-4-5-20250929",
    "max_tokens": 100,
    "messages": [
      {"role": "user", "content": "Hello"}
    ]
  }'
```

**流式请求:**

```bash
curl http://localhost:3000/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: your-ark-api-key" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-sonnet-4-5-20250929",
    "max_tokens": 100,
    "messages": [
      {"role": "user", "content": "Hello"}
    ],
    "stream": true
  }'
```

## Docker 部署

### 构建镜像

```bash
docker build -t anthropic-to-ark .
```

### 运行容器

```bash
docker run -p 3000:3000 \
  -e ARK_BASE_URL=https://ark.cn-beijing.volces.com/api/v3 \
  -e ARK_ENDPOINT_THINKING=ep-xxxxxxxxxx-xxxxx \
  -e ARK_ENDPOINT_DEFAULT=ep-xxxxxxxxxx-xxxxx \
  anthropic-to-ark
```

**可选配置（成本优化）：**
```bash
docker run -p 3000:3000 \
  -e ARK_BASE_URL=https://ark.cn-beijing.volces.com/api/v3 \
  -e ARK_ENDPOINT_THINKING=ep-deepseek-r1-xxxxx \
  -e ARK_ENDPOINT_FAST=ep-lite-model-xxxxx \
  -e ARK_ENDPOINT_DEFAULT=ep-standard-model-xxxxx \
  anthropic-to-ark
```

## API 端点

### POST /v1/messages

Anthropic Messages API 兼容端点。

**Headers:**
- `x-api-key`: 你的火山方舟 API Key (必需)
- `anthropic-version`: API 版本,如 `2023-06-01` (可选)
- `Content-Type`: `application/json`

**支持的参数:**
- `model`: Claude 模型名称
- `messages`: 消息列表
- `max_tokens`: 最大输出 token 数
- `system`: System prompt
- `temperature`: 温度参数 (0-1)
- `top_p`: Top-p 采样
- `stop_sequences`: 停止序列
- `stream`: 是否流式输出
- `tools`: 工具定义列表

### GET /health

健康检查端点,返回 `{"status": "ok"}`。

## 功能限制

### Extended Thinking 支持

本代理通过**模型切换**支持 Anthropic 的 Extended Thinking 功能：

**工作原理：**
1. 检测到请求中包含 `thinking` 参数时，自动切换到推理模型 endpoint（`ARK_ENDPOINT_THINKING`）
2. 推理模型原生返回 `reasoning_content` 字段
3. 将 `reasoning_content` 转换为 Anthropic 格式的 `thinking` 类型 content block

**模型配置：**
- 在火山方舟创建推理模型 endpoint（如 DeepSeek R1）
- 配置 `ARK_ENDPOINT_THINKING` 环境变量

**示例请求：**
```python
message = client.messages.create(
    model="claude-sonnet-4-5-20250929",  # 会自动切换到推理模型
    max_tokens=1024,
    thinking={
        "type": "enabled",
        "budget_tokens": 1000
    },
    messages=[
        {"role": "user", "content": "解释量子纠缠"}
    ]
)
```

**响应结构：**
```json
{
  "content": [
    {
      "type": "thinking",
      "text": "量子纠缠需要从量子态的叠加原理说起..."
    },
    {
      "type": "text",
      "text": "量子纠缠是指..."
    }
  ]
}
```

**优势：**
- ✅ 使用原生推理模型，质量有保证
- ✅ 支持流式输出思考过程
- ✅ 无需复杂的提示词工程
- ✅ API 字段直接映射，实现简洁

**限制：**
- ⚠️ 需要在火山方舟配置推理模型 endpoint
- ⚠️ `budget_tokens` 参数会被忽略（由模型自行控制思考长度）
- ⚠️ DeepSeek R1 等推理模型**不支持** `temperature`、`top_p`、`stop_sequences` 参数
  - 这些参数在使用 thinking 模式时会被自动跳过，避免 API 错误
  - 推理模型使用固定的采样策略以确保推理质量

### 不支持的功能

以下功能在请求时会返回 400 错误:

1. **PDF 文档** (`document` 类型内容)
   - 火山方舟目前只支持图像
   - TODO: 未来可能支持 PDF → 图片转换

2. **结构化输出** (`output_config` 参数)
   - 火山方舟 API 无对应参数

### 静默忽略的功能

以下功能会被静默忽略(不报错):

1. **Prompt Caching** (`cache_control` 参数)
   - 火山方舟自动处理缓存

2. **Citations** (引用功能)
   - 火山方舟不支持

### 自动适配的参数

以下参数会被自动调整以适配 ARK API 限制：

1. **max_tokens 限制**
   - ARK API 限制: <= 16384
   - 如果请求的 `max_tokens` 超过 16384，会自动调整为 16384
   - 这对 Anthropic SDK 的默认行为是透明的

## 模型映射

代理会自动将 Claude 模型映射到你配置的火山方舟 Endpoint:

| Claude Model 模式 | 环境变量 | Fallback 逻辑 |
|------------------|----------|---------------|
| 包含 "opus" | ARK_ENDPOINT_OPUS | → THINKING → DEFAULT |
| 包含 "haiku" | ARK_ENDPOINT_FAST | → DEFAULT |
| 包含 "sonnet" 或其他 | ARK_ENDPOINT_DEFAULT | - |
| 任何模型 + `thinking` 参数 | ARK_ENDPOINT_THINKING | → DEFAULT |

**示例模型映射：**
- `claude-opus-4-5-20251101` → OPUS
- `claude-opus-3-20240229` → OPUS
- `claude-sonnet-4-5-20250929` → DEFAULT
- `claude-3-5-sonnet-20241022` → DEFAULT
- `claude-haiku-4-5-20250925` → FAST (或 DEFAULT)
- `claude-3-5-haiku-20241022` → FAST (或 DEFAULT)
- 任何模型 + `thinking: {type: "enabled"}` → THINKING

## 项目结构

```
.
├── main.go                         # 应用入口
├── go.mod                          # Go 模块定义
├── config/
│   └── config.go                   # 配置管理
├── internal/
│   ├── types/
│   │   ├── anthropic.go            # Anthropic 类型定义
│   │   └── ark.go                  # 火山方舟类型定义
│   ├── handler/
│   │   └── messages.go             # Messages API 处理器
│   ├── transformer/
│   │   ├── request.go              # 请求转换器
│   │   ├── response.go             # 响应转换器
│   │   └── stream.go               # 流式转换器
│   ├── client/
│   │   └── ark.go                  # 火山方舟客户端
│   └── errors/
│       └── errors.go               # 错误定义
├── Dockerfile                      # Docker 构建文件
└── README.md                       # 项目文档
```

## 开发

### 运行测试

项目包含完整的测试套件，使用 DeepSeek R1 endpoint 进行测试。

**配置测试环境：**

```bash
# 1. 复制测试配置文件
cp .env.test.example .env.test

# 2. 编辑并填写你的 API Key
nano .env.test
```

**使用 Makefile（推荐）：**

```bash
# 查看所有可用命令
make help

# 运行完整测试
make test

# 快速测试（仅基础功能）
make test-quick

# Thinking 模式专项测试
make test-thinking

# 流式响应测试
make test-stream

# 错误处理测试
make test-error
```

**使用测试脚本：**

```bash
# 赋予执行权限
chmod +x test.sh

# 运行完整测试
./test.sh

# 快速测试
./test.sh quick

# Thinking 测试
./test.sh thinking
```

**手动运行：**

```bash
# 设置环境变量
export ARK_API_KEY="your-api-key"
export ARK_BASE_URL="https://ark-ap-southeast.byteintl.net/api/v3"
export ARK_ENDPOINT_THINKING="ep-20250424174745-w6pgh"
export ARK_ENDPOINT_DEFAULT="ep-20250424174745-w6pgh"

# 运行测试
go test -v ./internal/handler/
```

详细测试文档请查看 [TEST.md](./TEST.md)。

### 构建二进制

```bash
go build -o anthropic-to-ark
```

## 技术栈

- **Go 1.22**: 编程语言
- **Hertz**: 字节跳动开源的高性能 HTTP 框架
- **BytePlus Go SDK v2**: 官方 ARK Runtime SDK（arkruntime）
  - 类型安全的 API 客户端
  - 内置重试和错误处理
  - 原生支持流式响应和 `reasoning_content` 字段

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request!
