# OpenAI Responses API 知识文档

## 1. 概述
- Responses API 是 OpenAI 在 2025 年 3 月发布的新 API
- 结合了 Chat Completions API 和 Assistants API 的优点
- Assistants API 于 2025 年 8 月 26 日弃用，2026 年 8 月 26 日下线
- Responses API 是 Chat Completions 的超集
- 端点：`POST https://api.openai.com/v1/responses`

## 2. 核心端点

### 2.1 创建响应 (Create Response)
```
POST /v1/responses
```

### 2.2 获取响应 (Get Response)
```
GET /v1/responses/{response_id}
```

### 2.3 取消响应 (Cancel Response)
```
POST /v1/responses/{response_id}/cancel
```
- 只能取消 `background: true` 创建的响应

### 2.4 删除响应 (Delete Response)
```
DELETE /v1/responses/{response_id}
```
- 默认响应数据保留 30 天

## 3. 请求参数

### 3.1 必需参数
| 参数 | 类型 | 描述 |
|------|------|------|
| `model` | string | 模型 ID，如 `gpt-4o`, `o3`, `gpt-4.1` |
| `input` | string/array | 输入文本或消息数组 |

### 3.2 可选参数
| 参数 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `instructions` | string | - | 系统/开发者指令（替代 system message） |
| `tools` | array | - | 工具配置数组 |
| `tool_choice` | string/object | "auto" | 工具选择策略 |
| `parallel_tool_calls` | boolean | true | 是否允许并行工具调用 |
| `max_output_tokens` | integer | - | 最大输出 token 数 |
| `temperature` | number | 1 | 采样温度 (0-2) |
| `top_p` | number | 1 | 核采样概率 |
| `truncation` | string | "disabled" | 截断策略：`auto` 或 `disabled` |
| `store` | boolean | true | 是否存储响应（默认保留 30 天） |
| `metadata` | object | - | 16 个键值对的元数据 |
| `stream` | boolean | false | 是否启用流式响应 |
| `background` | boolean | false | 是否在后台运行 |
| `include` | array | - | 包含额外字段，如 `["reasoning.encrypted_content"]` |
| `previous_response_id` | string | - | 前一个响应 ID，用于会话链接 |
| `conversation_id` | string | - | 会话 ID |

### 3.3 推理模型参数
| 参数 | 类型 | 描述 |
|------|------|------|
| `reasoning.effort` | string | 推理努力程度：`low`, `medium`, `high` |
| `reasoning.summary` | string | 推理摘要配置 |

### 3.4 结构化输出参数
```json
{
  "text": {
    "format": {
      "type": "json_schema",
      "name": "schema_name",
      "strict": true,
      "schema": { ... }
    }
  }
}
```

## 4. 输入格式

### 4.1 简单文本输入
```json
{
  "model": "gpt-4o",
  "input": "What is the capital of France?"
}
```

### 4.2 消息数组输入
```json
{
  "model": "gpt-4o",
  "input": [
    {
      "role": "user",
      "content": [
        {
          "type": "input_text",
          "text": "Describe this image"
        },
        {
          "type": "input_image",
          "image_url": "https://example.com/image.png"
        }
      ]
    }
  ]
}
```

### 4.3 支持的内容类型
| 类型 | 描述 |
|------|------|
| `input_text` | 文本输入 |
| `input_image` | 图像输入（URL 或 base64） |
| `input_file` | 文件输入 |
| `input_audio` | 音频输入（base64 编码） |

### 4.4 消息角色
- `user`: 用户消息
- `assistant`: 助手消息（用于提供历史上下文）

## 5. 响应结构

### 5.1 响应对象
```json
{
  "id": "resp_xxx",
  "object": "response",
  "created_at": 1234567890,
  "status": "completed",
  "model": "gpt-4o",
  "output": [...],
  "output_text": "Combined text output",
  "usage": {
    "input_tokens": 100,
    "output_tokens": 200,
    "total_tokens": 300,
    "input_tokens_details": { "cached_tokens": 0 },
    "output_tokens_details": { "reasoning_tokens": 50 }
  }
}
```

### 5.2 Output 数组项类型
| 类型 | 描述 |
|------|------|
| `message` | 文本消息输出 |
| `function_call` | 函数调用 |
| `function_call_output` | 函数调用结果 |
| `web_search_call` | Web 搜索调用 |
| `file_search_call` | 文件搜索调用 |
| `code_interpreter_call` | 代码解释器调用 |
| `computer_call` | 计算机操作调用 |
| `image_generation_call` | 图像生成调用 |
| `mcp_call` | MCP 服务器调用 |
| `reasoning` | 推理项 |

### 5.3 内容类型
| 类型 | 描述 |
|------|------|
| `output_text` | 文本输出 |
| `refusal` | 拒绝响应 |

## 6. 内置工具 (Built-in Tools)

### 6.1 Web 搜索 (web_search_preview)
```json
{
  "type": "web_search_preview",
  "user_location": {
    "type": "approximate",
    "country": "US",
    "region": "California",
    "city": "San Francisco",
    "timezone": "America/Los_Angeles"
  },
  "search_context_size": "medium"
}
```

**参数：**
- `search_context_size`: `low` | `medium` | `high`
- `user_location`: 位置对象，包含 country, region, city, timezone

**限制：**
- 上下文窗口限制为 128000 tokens

### 6.2 文件搜索 (file_search)
```json
{
  "type": "file_search",
  "vector_store_ids": ["vs_xxx"],
  "max_num_results": 10,
  "ranking_options": {
    "ranker": "auto",
    "score_threshold": 0.5,
    "hybrid_search": {
      "embedding_weight": 0.5,
      "text_weight": 0.5
    }
  }
}
```

**参数：**
- `vector_store_ids`: 向量存储 ID 数组
- `max_num_results`: 最大结果数（默认 10，最大 50）
- `ranking_options`: 排序选项
  - `ranker`: 排序器类型
  - `score_threshold`: 分数阈值 (0.0-1.0)
  - `hybrid_search`: 混合搜索权重配置

### 6.3 代码解释器 (code_interpreter)
```json
{
  "type": "code_interpreter",
  "container": {
    "type": "auto",
    "memory_limit": "4g",
    "file_ids": ["file-xxx"]
  }
}
```

**参数：**
- `container.type`: `auto`（自动创建）或使用显式容器 ID
- `container.memory_limit`: 内存限制，如 `"4g"`
- `container.file_ids`: 要上传到容器的文件 ID

**容器管理：**
- 容器 20 分钟不使用会过期
- 使用 `POST /v1/containers` 显式创建容器
- 使用 `POST /v1/containers/{container_id}/files` 管理容器文件

### 6.4 计算机使用 (computer_use_preview)
```json
{
  "type": "computer_use_preview",
  "display_width": 1024,
  "display_height": 768,
  "environment": "browser"
}
```

**参数：**
- `display_width`: 显示宽度（像素）
- `display_height`: 显示高度（像素）
- `environment`: 环境类型
  - `browser`
  - `mac`
  - `windows`
  - `ubuntu`

**注意：** 使用 computer_use_preview 时必须设置 `truncation: "auto"`

### 6.5 图像生成 (image_generation)
```json
{
  "type": "image_generation",
  "action": "auto"
}
```

**参数：**
- `action`:
  - `auto`: 模型决定生成还是编辑
  - `generate`: 始终生成新图像
  - `edit`: 强制编辑（需要上下文中有图像）

**支持模型：**
- gpt-image-1.5
- gpt-image-1
- gpt-image-1-mini

### 6.6 远程 MCP 服务器 (mcp)
```json
{
  "type": "mcp",
  "server_label": "my-mcp-server",
  "server_url": "https://example.com/mcp",
  "server_description": "My MCP server description",
  "require_approval": "never",
  "headers": {
    "Authorization": "Bearer xxx"
  }
}
```

**参数：**
- `server_label`: 服务器唯一标识
- `server_url`: MCP 服务器 URL
- `server_description`: 服务器描述（可选）
- `require_approval`: 审批要求
  - `always`: 始终需要审批
  - `never`: 不需要审批
- `headers`: HTTP 请求头（用于认证）

**传输协议：**
- Streamable HTTP
- HTTP/SSE

## 7. 函数调用 (Function Calling)

### 7.1 定义函数
```json
{
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_weather",
        "description": "Get current weather",
        "parameters": {
          "type": "object",
          "properties": {
            "location": { "type": "string" }
          },
          "required": ["location"]
        }
      }
    }
  ]
}
```

### 7.2 tool_choice 选项
| 值 | 描述 |
|----|------|
| `"auto"` | 模型自动决定是否调用函数（默认） |
| `"required"` | 强制调用某个函数 |
| `"none"` | 禁止调用函数 |
| `{"type": "function", "function": {"name": "xxx"}}` | 强制调用指定函数 |

## 8. 流式响应 (Streaming)

### 8.1 启用流式
```json
{
  "model": "gpt-4o",
  "input": "Hello",
  "stream": true
}
```

### 8.2 主要事件类型
| 事件 | 描述 |
|------|------|
| `response.created` | 响应创建 |
| `response.in_progress` | 响应进行中 |
| `response.completed` | 响应完成 |
| `response.failed` | 响应失败 |
| `response.cancelled` | 响应取消 |
| `response.output_item.added` | 输出项添加 |
| `response.output_item.done` | 输出项完成 |
| `response.content_part.added` | 内容部分添加 |
| `response.content_part.done` | 内容部分完成 |
| `response.output_text.delta` | 文本增量 |
| `response.output_text.done` | 文本完成 |
| `response.reasoning_text.delta` | 推理文本增量 |
| `response.reasoning_text.done` | 推理文本完成 |
| `response.reasoning_summary_text.delta` | 推理摘要增量 |
| `response.reasoning_summary_text.done` | 推理摘要完成 |
| `error` | 错误 |

### 8.3 Delta 事件结构
```json
{
  "type": "response.output_text.delta",
  "item_id": "item_xxx",
  "output_index": 0,
  "content_index": 0,
  "delta": "Hello",
  "sequence_number": 1
}
```

**注意：** 使用 `sequence_number` 保证事件顺序

## 9. 后台模式 (Background Mode)

### 9.1 启用后台模式
```json
{
  "model": "o3",
  "input": "Complex task...",
  "background": true,
  "stream": true
}
```

### 9.2 使用 Webhooks
1. 在 Dashboard 设置 webhook endpoint
2. 订阅 `response.completed` 事件
3. 发起后台模式请求
4. 通过 webhook 接收完成通知

## 10. 会话状态管理

### 10.1 方法一：使用 previous_response_id
```json
{
  "model": "gpt-4o",
  "input": "Follow up question",
  "previous_response_id": "resp_xxx"
}
```

### 10.2 方法二：使用 Conversations API
```bash
# 创建会话
POST /v1/conversations

# 在会话中创建响应
POST /v1/responses
{
  "model": "gpt-4o",
  "input": "Hello",
  "conversation_id": "conv_xxx"
}
```

### 10.3 方法三：手动管理
自行维护消息历史，每次请求传入完整上下文

## 11. 结构化输出 (Structured Outputs)

### 11.1 JSON Schema 模式
```json
{
  "model": "gpt-4o",
  "input": "Extract the data",
  "text": {
    "format": {
      "type": "json_schema",
      "name": "extraction_result",
      "strict": true,
      "schema": {
        "type": "object",
        "properties": {
          "name": { "type": "string" },
          "age": { "type": "integer" }
        },
        "required": ["name", "age"],
        "additionalProperties": false
      }
    }
  }
}
```

### 11.2 使用 SDK 的 parse 方法
```python
from pydantic import BaseModel

class Result(BaseModel):
    name: str
    age: int

response = client.responses.parse(
    model="gpt-4o",
    input="Extract: John is 30 years old",
    text_format=Result
)
```

## 12. 与 Chat Completions API 的主要区别

| 特性 | Chat Completions | Responses API |
|------|------------------|---------------|
| 系统消息 | messages 数组中的 role: system | 顶层 instructions 参数 |
| 结构化输出 | response_format | text.format |
| 内置工具 | 不支持 | 支持 web_search, file_search 等 |
| 会话管理 | 手动 | previous_response_id / Conversations |
| 后台模式 | 不支持 | 支持 |
| 输出结构 | choices[0].message | output 数组 + output_text |

## 13. SDK 使用示例

### 13.1 Python SDK
```python
from openai import OpenAI

client = OpenAI()

# 基本用法
response = client.responses.create(
    model="gpt-4o",
    input="Hello, world!"
)
print(response.output_text)

# 带工具的请求
response = client.responses.create(
    model="gpt-4o",
    input="What's the weather in Tokyo?",
    tools=[{"type": "web_search_preview"}]
)

# 流式响应
for event in client.responses.create(
    model="gpt-4o",
    input="Tell me a story",
    stream=True
):
    if event.type == "response.output_text.delta":
        print(event.delta, end="")
```

### 13.2 Node.js SDK
```javascript
import OpenAI from 'openai';

const client = new OpenAI();

// 基本用法
const response = await client.responses.create({
    model: "gpt-4o",
    input: "Hello, world!"
});
console.log(response.output_text);

// 流式响应
const stream = await client.responses.create({
    model: "gpt-4o",
    input: "Tell me a story",
    stream: true
});

for await (const event of stream) {
    if (event.type === "response.output_text.delta") {
        process.stdout.write(event.delta);
    }
}
```

## 信息来源

- [OpenAI API Reference - Responses](https://platform.openai.com/docs/api-reference/responses)
- [Migrate to the Responses API](https://platform.openai.com/docs/guides/migrate-to-responses)
- [Streaming Events Reference](https://platform.openai.com/docs/api-reference/responses-streaming)
- [Using Tools](https://platform.openai.com/docs/guides/tools)
- [Web Search Tool](https://platform.openai.com/docs/guides/tools-web-search)
- [File Search Tool](https://platform.openai.com/docs/guides/tools-file-search)
- [Code Interpreter Tool](https://platform.openai.com/docs/guides/tools-code-interpreter)
- [Computer Use Tool](https://platform.openai.com/docs/guides/tools-computer-use)
- [Connectors and MCP Servers](https://platform.openai.com/docs/guides/tools-connectors-mcp)
- [Background Mode](https://platform.openai.com/docs/guides/background)
- [Conversation State](https://platform.openai.com/docs/guides/conversation-state)
- [Structured Outputs](https://platform.openai.com/docs/guides/structured-outputs)
