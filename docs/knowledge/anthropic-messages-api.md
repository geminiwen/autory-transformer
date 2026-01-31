# Anthropic Messages API 知识文档

## 1. 概述

### 1.1 平台介绍
- **Anthropic Claude** 是 Anthropic 公司开发的先进 AI 助手
- 提供安全、可靠、可控的 AI 能力
- API 端点：`https://api.anthropic.com`
- 核心 API 是 Messages API，用于对话交互

### 1.2 主要特性
- 强大的推理和分析能力
- 扩展思考模式（Extended Thinking）
- 多模态支持（文本、图像、PDF）
- 工具使用（Tool Use）
- 提示缓存（Prompt Caching）
- 结构化输出（Structured Outputs）
- 批量处理（Batch API）

### 1.3 API 版本
- 当前 API 版本：`2023-06-01`
- 通过 `anthropic-version` header 指定
- Beta 功能需要额外的 `anthropic-beta` header

## 2. 核心端点

### 2.1 Messages API
```
POST https://api.anthropic.com/v1/messages
```

### 2.2 Message Batches API
```
POST https://api.anthropic.com/v1/messages/batches
GET  https://api.anthropic.com/v1/messages/batches/{batch_id}
GET  https://api.anthropic.com/v1/messages/batches/{batch_id}/results
```

## 3. 认证方式

### 3.1 API Key 获取
1. 访问 [Anthropic Console](https://console.anthropic.com/)
2. 创建 API Key
3. 妥善保管密钥

### 3.2 认证方式
使用 HTTP Header 进行认证：
```
x-api-key: YOUR_API_KEY
```

### 3.3 环境变量设置
```bash
export ANTHROPIC_API_KEY="your-api-key-here"
```

## 4. Messages API 详解

### 4.1 请求参数

#### 必需参数
| 参数 | 类型 | 描述 |
|------|------|------|
| `model` | string | 模型标识符 |
| `max_tokens` | integer | 最大生成 token 数 |
| `messages` | array | 消息数组 |

#### 可选参数
| 参数 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `system` | string/array | - | 系统提示 |
| `temperature` | number | 1.0 | 采样温度 (0-1) |
| `top_p` | number | - | 核采样概率 |
| `top_k` | integer | - | Top-K 采样 |
| `metadata` | object | - | 元数据对象 |
| `stop_sequences` | array | - | 停止序列（最多 4 个）|
| `stream` | boolean | false | 是否启用流式响应 |
| `tools` | array | - | 工具定义数组 |
| `tool_choice` | object | - | 工具选择策略 |

#### 高级参数
| 参数 | 类型 | 描述 |
|------|------|------|
| `thinking.type` | string | 启用扩展思考模式 |
| `thinking.budget_tokens` | integer | 思考预算 token 数 |
| `output_config.format` | object | 结构化输出配置 |

### 4.2 Messages 格式

#### User 消息（纯文本）
```json
{
  "role": "user",
  "content": "Hello, Claude!"
}
```

#### User 消息（多模态）
```json
{
  "role": "user",
  "content": [
    {
      "type": "text",
      "text": "What's in this image?"
    },
    {
      "type": "image",
      "source": {
        "type": "base64",
        "media_type": "image/png",
        "data": "iVBORw0KGgoAAAANS..."
      }
    }
  ]
}
```

#### User 消息（图像 URL）
```json
{
  "type": "image",
  "source": {
    "type": "url",
    "url": "https://example.com/image.png"
  }
}
```

#### Assistant 消息
```json
{
  "role": "assistant",
  "content": "Hello! How can I help you today?"
}
```

#### Assistant 消息（带工具调用）
```json
{
  "role": "assistant",
  "content": [
    {
      "type": "text",
      "text": "Let me check the weather for you."
    },
    {
      "type": "tool_use",
      "id": "toolu_xxx",
      "name": "get_weather",
      "input": {
        "location": "San Francisco"
      }
    }
  ]
}
```

#### User 消息（工具结果）
```json
{
  "role": "user",
  "content": [
    {
      "type": "tool_result",
      "tool_use_id": "toolu_xxx",
      "content": "{\"temperature\": 20, \"condition\": \"sunny\"}"
    }
  ]
}
```

### 4.3 System Prompt

#### 简单 System Prompt
```json
{
  "model": "claude-sonnet-4-5-20250929",
  "max_tokens": 1024,
  "system": "You are a helpful AI assistant.",
  "messages": [...]
}
```

#### 复杂 System Prompt（带缓存）
```json
{
  "system": [
    {
      "type": "text",
      "text": "You are an expert programmer.",
      "cache_control": {"type": "ephemeral"}
    }
  ],
  "messages": [...]
}
```

### 4.4 响应格式

#### 非流式响应
```json
{
  "id": "msg_xxx",
  "type": "message",
  "role": "assistant",
  "content": [
    {
      "type": "text",
      "text": "Hello! How can I help you today?"
    }
  ],
  "model": "claude-sonnet-4-5-20250929",
  "stop_reason": "end_turn",
  "stop_sequence": null,
  "usage": {
    "input_tokens": 10,
    "output_tokens": 15,
    "cache_creation_input_tokens": 0,
    "cache_read_input_tokens": 0
  }
}
```

#### stop_reason 类型
| 值 | 描述 |
|----|------|
| `end_turn` | 自然结束 |
| `max_tokens` | 达到 max_tokens 限制 |
| `stop_sequence` | 遇到停止序列 |
| `tool_use` | 模型调用了工具 |

### 4.5 流式响应

#### 启用流式
```json
{
  "model": "claude-sonnet-4-5-20250929",
  "max_tokens": 1024,
  "messages": [...],
  "stream": true
}
```

#### 流式事件类型
| 事件类型 | 描述 |
|---------|------|
| `message_start` | 消息开始 |
| `content_block_start` | 内容块开始 |
| `content_block_delta` | 内容块增量 |
| `content_block_stop` | 内容块结束 |
| `message_delta` | 消息元数据更新 |
| `message_stop` | 消息结束 |
| `ping` | 保持连接的心跳 |
| `error` | 错误事件 |

#### 流式响应示例
```
event: message_start
data: {"type":"message_start","message":{"id":"msg_xxx","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-5-20250929","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"!"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":15}}

event: message_stop
data: {"type":"message_stop"}
```

## 5. Claude 模型列表

### 5.1 Claude 4.5 系列（最新）

#### Claude Opus 4.5
- **模型 ID**: `claude-opus-4-5-20251101`
- **发布日期**: 2025 年 11 月
- **特点**: 最强大的模型，卓越的推理和分析能力
- **上下文窗口**: 200K tokens
- **最大输出**: 16K tokens

#### Claude Sonnet 4.5
- **模型 ID**: `claude-sonnet-4-5-20250929`
- **发布日期**: 2025 年 9 月
- **特点**: 性能与成本的最佳平衡
- **上下文窗口**: 200K tokens
- **最大输出**: 8K tokens
- **推荐用途**: 大多数应用场景

#### Claude Haiku 4.5
- **模型 ID**: `claude-haiku-4-5-20250925`
- **发布日期**: 2025 年 9 月
- **特点**: 最快速、最经济的模型
- **上下文窗口**: 200K tokens
- **最大输出**: 8K tokens

### 5.2 Claude 4 系列

#### Claude Opus 4
- **模型 ID**: `claude-opus-4-20250514`
- **上下文窗口**: 128K tokens

#### Claude Sonnet 4
- **模型 ID**: `claude-sonnet-4-20250514`
- **上下文窗口**: 128K tokens

### 5.3 Claude 3.5 系列（传统）

#### Claude 3.5 Sonnet
- **模型 ID**: `claude-3-5-sonnet-20241022`
- **上下文窗口**: 200K tokens
- **最大输出**: 8K tokens

#### Claude 3.5 Haiku
- **模型 ID**: `claude-3-5-haiku-20241022`
- **上下文窗口**: 200K tokens
- **最大输出**: 8K tokens

### 5.4 Claude 3 系列（传统）

| 模型 | 模型 ID | 上下文 | 特点 |
|------|---------|--------|------|
| Claude 3 Opus | claude-3-opus-20240229 | 200K | 最强 v3 模型 |
| Claude 3 Sonnet | claude-3-sonnet-20240229 | 200K | 平衡 v3 模型 |
| Claude 3 Haiku | claude-3-haiku-20240307 | 200K | 快速 v3 模型 |

## 6. 工具使用 (Tool Use)

### 6.1 定义工具
```json
{
  "model": "claude-sonnet-4-5-20250929",
  "max_tokens": 1024,
  "tools": [
    {
      "name": "get_weather",
      "description": "Get the current weather in a given location",
      "input_schema": {
        "type": "object",
        "properties": {
          "location": {
            "type": "string",
            "description": "The city and state, e.g. San Francisco, CA"
          },
          "unit": {
            "type": "string",
            "enum": ["celsius", "fahrenheit"],
            "description": "The unit of temperature"
          }
        },
        "required": ["location"]
      }
    }
  ],
  "messages": [
    {"role": "user", "content": "What's the weather in SF?"}
  ]
}
```

### 6.2 工具选择策略

#### 自动选择（默认）
```json
{
  "tool_choice": {"type": "auto"}
}
```

#### 强制使用任意工具
```json
{
  "tool_choice": {"type": "any"}
}
```

#### 强制使用指定工具
```json
{
  "tool_choice": {
    "type": "tool",
    "name": "get_weather"
  }
}
```

### 6.3 处理工具调用

#### 模型响应（包含工具调用）
```json
{
  "id": "msg_xxx",
  "content": [
    {
      "type": "text",
      "text": "Let me check the weather for you."
    },
    {
      "type": "tool_use",
      "id": "toolu_xxx",
      "name": "get_weather",
      "input": {
        "location": "San Francisco, CA",
        "unit": "fahrenheit"
      }
    }
  ],
  "stop_reason": "tool_use"
}
```

#### 提供工具结果
```json
{
  "model": "claude-sonnet-4-5-20250929",
  "max_tokens": 1024,
  "messages": [
    {"role": "user", "content": "What's the weather in SF?"},
    {
      "role": "assistant",
      "content": [
        {"type": "text", "text": "Let me check the weather for you."},
        {
          "type": "tool_use",
          "id": "toolu_xxx",
          "name": "get_weather",
          "input": {"location": "San Francisco, CA", "unit": "fahrenheit"}
        }
      ]
    },
    {
      "role": "user",
      "content": [
        {
          "type": "tool_result",
          "tool_use_id": "toolu_xxx",
          "content": "65°F and sunny"
        }
      ]
    }
  ]
}
```

### 6.4 严格工具使用（Strict Tool Use）

启用严格模式确保工具输入严格遵循 schema：

```json
{
  "tools": [
    {
      "name": "get_weather",
      "description": "Get weather",
      "input_schema": {
        "type": "object",
        "properties": {
          "location": {"type": "string"}
        },
        "required": ["location"],
        "additionalProperties": false
      }
    }
  ]
}
```

**要求：**
- 使用 beta header: `anthropic-beta: strict-tool-use-2025-11-13`
- 所有对象必须设置 `additionalProperties: false`
- 所有字段必须在 `required` 中

## 7. 视觉理解 (Vision)

### 7.1 图像输入

#### Base64 编码
```json
{
  "role": "user",
  "content": [
    {
      "type": "image",
      "source": {
        "type": "base64",
        "media_type": "image/jpeg",
        "data": "/9j/4AAQSkZJRg..."
      }
    },
    {
      "type": "text",
      "text": "Describe this image"
    }
  ]
}
```

#### URL 引用
```json
{
  "type": "image",
  "source": {
    "type": "url",
    "url": "https://example.com/image.png"
  }
}
```

### 7.2 支持的图像格式
- `image/jpeg`
- `image/png`
- `image/gif`
- `image/webp`

### 7.3 图像限制
- 单次请求最多 20 张图像（claude.ai）
- API 请求最多 100 张图像
- 建议将图像放在文本之前以获得最佳性能

### 7.4 PDF 支持

Claude 可以理解 PDF 文档（最多 100 页）：
- 转换每一页为图像
- 提取文本内容
- 分析图表、表格等视觉元素

```json
{
  "role": "user",
  "content": [
    {
      "type": "document",
      "source": {
        "type": "base64",
        "media_type": "application/pdf",
        "data": "JVBERi0xLjQKJ..."
      }
    },
    {
      "type": "text",
      "text": "Summarize this PDF"
    }
  ]
}
```

## 8. 扩展思考 (Extended Thinking)

### 8.1 概述
扩展思考允许模型在生成响应前进行深度思考，提高复杂任务的准确性。

### 8.2 启用扩展思考
```json
{
  "model": "claude-sonnet-4-5-20250929",
  "max_tokens": 4096,
  "thinking": {
    "type": "enabled",
    "budget_tokens": 2000
  },
  "messages": [
    {"role": "user", "content": "Solve this complex math problem..."}
  ]
}
```

### 8.3 参数说明
| 参数 | 类型 | 描述 |
|------|------|------|
| `thinking.type` | string | `enabled` 启用扩展思考 |
| `thinking.budget_tokens` | integer | 思考预算（最多 10K tokens） |

### 8.4 响应格式
```json
{
  "content": [
    {
      "type": "thinking",
      "thinking": "Let me break down this problem step by step..."
    },
    {
      "type": "text",
      "text": "The answer is 42."
    }
  ]
}
```

### 8.5 最佳实践
- 从通用指令开始，根据思考输出迭代优化
- 使用 few-shot 示例引导思考模式
- 不要将思考内容回传给模型
- 不要预填充（prefill）思考内容

## 9. 提示缓存 (Prompt Caching)

### 9.1 概述
- 缓存重复使用的大段上下文
- 降低成本和延迟
- 缓存有效期：5 分钟（默认）或 1 小时（扩展）

### 9.2 启用缓存

#### Beta Header
```
anthropic-beta: prompt-caching-2024-07-31
```

#### 扩展缓存（1 小时）
```
anthropic-beta: extended-cache-ttl-2025-04-11
```

### 9.3 标记缓存点
```json
{
  "model": "claude-sonnet-4-5-20250929",
  "max_tokens": 1024,
  "system": [
    {
      "type": "text",
      "text": "You are an AI assistant for Acme Inc...",
      "cache_control": {"type": "ephemeral"}
    }
  ],
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "Here is the full text of a complex document...",
          "cache_control": {"type": "ephemeral"}
        },
        {
          "type": "text",
          "text": "What does it say about budget?"
        }
      ]
    }
  ]
}
```

### 9.4 最小缓存要求
| 模型 | 最小 tokens |
|------|-----------|
| Claude Sonnet 4.5 / Opus 4.5 | 1024 |
| Claude Haiku 4.5 | 2048 |
| Claude 3.5 Sonnet / Opus | 1024 |
| Claude 3.5 Haiku | 2048 |

### 9.5 缓存统计
```json
{
  "usage": {
    "input_tokens": 100,
    "output_tokens": 50,
    "cache_creation_input_tokens": 2000,
    "cache_read_input_tokens": 1950
  }
}
```

**字段说明：**
- `cache_creation_input_tokens`: 创建缓存的 tokens（全价）
- `cache_read_input_tokens`: 命中缓存的 tokens（90% 折扣）

## 10. 结构化输出 (Structured Outputs)

### 10.1 概述
结构化输出确保 Claude 的响应严格遵循指定的 JSON schema，使用约束解码技术。

### 10.2 支持的模型
- Claude Sonnet 4.5
- Claude Opus 4.5
- Claude Haiku 4.5

### 10.3 配置结构化输出

#### 新版 API（推荐）
```json
{
  "model": "claude-sonnet-4-5-20250929",
  "max_tokens": 1024,
  "output_config": {
    "format": {
      "type": "json_schema",
      "json_schema": {
        "name": "user_info",
        "strict": true,
        "schema": {
          "type": "object",
          "properties": {
            "name": {"type": "string"},
            "age": {"type": "integer"},
            "email": {"type": "string"}
          },
          "required": ["name", "age", "email"],
          "additionalProperties": false
        }
      }
    }
  },
  "messages": [
    {"role": "user", "content": "Extract user info: John is 30, email john@example.com"}
  ]
}
```

#### 旧版 API（仍支持）
使用 beta header: `anthropic-beta: structured-outputs-2025-11-13`

```json
{
  "output_format": {
    "type": "json_schema",
    "json_schema": {...}
  }
}
```

### 10.4 Schema 要求
- 必须设置 `"strict": true`
- 所有对象必须包含 `"additionalProperties": false`
- 所有属性必须在 `required` 数组中声明
- 支持的类型：string, number, boolean, integer, object, array, enum, anyOf

### 10.5 限制
- 不兼容 Citations
- 不兼容 Message Prefilling
- 数值约束（minimum, maximum）不会被强制执行

## 11. 批量处理 (Batch API)

### 11.1 概述
- 异步处理大量请求
- 成本降低 50%
- 适合非实时任务
- 大多数批次在 1 小时内完成

### 11.2 创建批次

#### 准备请求文件（JSONL）
```jsonl
{"custom_id": "request-1", "params": {"model": "claude-sonnet-4-5-20250929", "max_tokens": 1024, "messages": [{"role": "user", "content": "Hello"}]}}
{"custom_id": "request-2", "params": {"model": "claude-sonnet-4-5-20250929", "max_tokens": 1024, "messages": [{"role": "user", "content": "Hi"}]}}
```

#### 创建批次请求
```bash
curl https://api.anthropic.com/v1/messages/batches \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d '{
    "requests": [
      {
        "custom_id": "request-1",
        "params": {
          "model": "claude-sonnet-4-5-20250929",
          "max_tokens": 1024,
          "messages": [{"role": "user", "content": "Hello"}]
        }
      }
    ]
  }'
```

### 11.3 批次限制
- 最多 100,000 个请求
- 文件大小最多 256 MB
- 批次处理时间最多 24 小时
- 结果保留 29 天

### 11.4 查询批次状态
```bash
curl https://api.anthropic.com/v1/messages/batches/{batch_id} \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01"
```

### 11.5 获取批次结果
```bash
curl https://api.anthropic.com/v1/messages/batches/{batch_id}/results \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01"
```

### 11.6 常见用例
- 大规模评估
- 内容审核
- 数据分类
- 批量翻译
- 文档处理

## 12. SDK 使用示例

### 12.1 Python SDK

#### 安装
```bash
pip install anthropic
```

#### 基本用法
```python
import anthropic

client = anthropic.Anthropic(
    api_key="your-api-key"
)

message = client.messages.create(
    model="claude-sonnet-4-5-20250929",
    max_tokens=1024,
    messages=[
        {"role": "user", "content": "Hello, Claude!"}
    ]
)

print(message.content[0].text)
```

#### 流式响应
```python
with client.messages.stream(
    model="claude-sonnet-4-5-20250929",
    max_tokens=1024,
    messages=[
        {"role": "user", "content": "Tell me a story"}
    ]
) as stream:
    for text in stream.text_stream:
        print(text, end="", flush=True)
```

#### 使用工具
```python
message = client.messages.create(
    model="claude-sonnet-4-5-20250929",
    max_tokens=1024,
    tools=[
        {
            "name": "get_weather",
            "description": "Get weather info",
            "input_schema": {
                "type": "object",
                "properties": {
                    "location": {"type": "string"}
                },
                "required": ["location"]
            }
        }
    ],
    messages=[
        {"role": "user", "content": "What's the weather in Tokyo?"}
    ]
)

# 处理工具调用
if message.stop_reason == "tool_use":
    tool_use = next(block for block in message.content if block.type == "tool_use")
    print(f"Tool: {tool_use.name}")
    print(f"Input: {tool_use.input}")
```

#### 视觉理解
```python
import base64

with open("image.jpg", "rb") as f:
    image_data = base64.b64encode(f.read()).decode("utf-8")

message = client.messages.create(
    model="claude-sonnet-4-5-20250929",
    max_tokens=1024,
    messages=[
        {
            "role": "user",
            "content": [
                {
                    "type": "image",
                    "source": {
                        "type": "base64",
                        "media_type": "image/jpeg",
                        "data": image_data
                    }
                },
                {
                    "type": "text",
                    "text": "Describe this image"
                }
            ]
        }
    ]
)
```

### 12.2 TypeScript SDK

#### 安装
```bash
npm install @anthropic-ai/sdk
```

#### 基本用法
```typescript
import Anthropic from '@anthropic-ai/sdk';

const client = new Anthropic({
  apiKey: process.env.ANTHROPIC_API_KEY
});

const message = await client.messages.create({
  model: 'claude-sonnet-4-5-20250929',
  max_tokens: 1024,
  messages: [
    { role: 'user', content: 'Hello, Claude!' }
  ]
});

console.log(message.content[0].text);
```

#### 流式响应
```typescript
const stream = await client.messages.stream({
  model: 'claude-sonnet-4-5-20250929',
  max_tokens: 1024,
  messages: [
    { role: 'user', content: 'Tell me a story' }
  ]
});

for await (const event of stream) {
  if (event.type === 'content_block_delta' && event.delta.type === 'text_delta') {
    process.stdout.write(event.delta.text);
  }
}
```

#### 使用工具
```typescript
const message = await client.messages.create({
  model: 'claude-sonnet-4-5-20250929',
  max_tokens: 1024,
  tools: [
    {
      name: 'get_weather',
      description: 'Get weather info',
      input_schema: {
        type: 'object',
        properties: {
          location: { type: 'string' }
        },
        required: ['location']
      }
    }
  ],
  messages: [
    { role: 'user', content: "What's the weather in Tokyo?" }
  ]
});

if (message.stop_reason === 'tool_use') {
  const toolUse = message.content.find(block => block.type === 'tool_use');
  console.log(`Tool: ${toolUse.name}`);
  console.log(`Input:`, toolUse.input);
}
```

### 12.3 cURL 示例

#### 基本请求
```bash
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d '{
    "model": "claude-sonnet-4-5-20250929",
    "max_tokens": 1024,
    "messages": [
      {"role": "user", "content": "Hello, Claude!"}
    ]
  }'
```

#### 流式请求
```bash
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d '{
    "model": "claude-sonnet-4-5-20250929",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'
```

## 13. 价格说明

### 13.1 Claude 4.5 系列定价

#### 标准上下文（≤200K tokens）
| 模型 | 输入价格 | 输出价格 | 缓存写入 | 缓存读取 |
|------|---------|---------|---------|---------|
| Claude Opus 4.5 | $5.00/MTok | $25.00/MTok | $6.25/MTok | $0.50/MTok |
| Claude Sonnet 4.5 | $3.00/MTok | $15.00/MTok | $3.75/MTok | $0.30/MTok |
| Claude Haiku 4.5 | $1.00/MTok | $5.00/MTok | $1.25/MTok | $0.10/MTok |

#### 长上下文（>200K tokens）
| 模型 | 输入价格 | 输出价格 |
|------|---------|---------|
| Claude Sonnet 4.5 | $6.00/MTok | $22.50/MTok |

### 13.2 批量处理价格
- 所有批量请求享受 **50% 折扣**
- 可与提示缓存折扣叠加

### 13.3 提示缓存折扣
- 缓存读取：**90% 折扣**
- 缓存写入：25% 额外成本
- 缓存命中时总成本可降低 80-90%

### 13.4 成本优化建议
- 使用 Haiku 处理简单任务
- 使用 Sonnet 处理大多数任务
- 仅在需要最高性能时使用 Opus
- 利用提示缓存降低重复请求成本
- 非实时任务使用批量 API

## 14. 速率限制

### 14.1 限制类型
- **RPM** (Requests Per Minute): 每分钟请求数
- **ITPM** (Input Tokens Per Minute): 每分钟输入 tokens
- **OTPM** (Output Tokens Per Minute): 每分钟输出 tokens

### 14.2 使用层级

| 层级 | 要求 | RPM | TPM |
|------|------|-----|-----|
| Tier 1 | 新账户 | 50 | 50K |
| Tier 2 | $25 消费 | 1,000 | 80K |
| Tier 3 | $200 消费 | 2,000 | 160K |
| Tier 4 | $1,000 消费 | 4,000 | 400K |

**注意：** 具体限制因模型而异，会随着使用量自动提升。

### 14.3 速率限制响应
```json
{
  "error": {
    "type": "rate_limit_error",
    "message": "Rate limit exceeded"
  }
}
```

**HTTP Headers:**
```
retry-after: 10
```

## 15. 错误处理

### 15.1 错误类型

| 错误码 | 错误类型 | 描述 | 处理方式 |
|--------|---------|------|---------|
| 400 | invalid_request_error | 请求参数错误 | 检查请求格式 |
| 401 | authentication_error | 认证失败 | 检查 API Key |
| 403 | permission_error | 权限不足 | 检查账户权限 |
| 404 | not_found_error | 资源不存在 | 检查资源 ID |
| 429 | rate_limit_error | 超出速率限制 | 等待并重试 |
| 500 | api_error | API 内部错误 | 稍后重试 |
| 529 | overloaded_error | API 过载 | 稍后重试 |

### 15.2 重试策略

#### Python 示例
```python
import time
import anthropic
from anthropic import RateLimitError

client = anthropic.Anthropic()

def create_message_with_retry(messages, max_retries=3):
    for attempt in range(max_retries):
        try:
            return client.messages.create(
                model="claude-sonnet-4-5-20250929",
                max_tokens=1024,
                messages=messages
            )
        except RateLimitError as e:
            if attempt < max_retries - 1:
                # 检查 retry-after header
                retry_after = getattr(e.response, 'headers', {}).get('retry-after')
                wait_time = int(retry_after) if retry_after else 2 ** attempt
                print(f"Rate limited. Waiting {wait_time} seconds...")
                time.sleep(wait_time)
            else:
                raise
        except anthropic.APIError as e:
            if e.status_code in [500, 529]:
                if attempt < max_retries - 1:
                    wait_time = 2 ** attempt
                    time.sleep(wait_time)
                else:
                    raise
            else:
                raise
```

### 15.3 最佳实践
1. **始终检查 `retry-after` header**
2. 使用指数退避策略
3. 实现请求队列管理
4. 监控速率限制使用情况
5. 优雅处理加速限制（acceleration limits）

## 16. 最佳实践

### 16.1 提示工程
- 清晰、具体的指令
- 使用 XML 标签组织内容
- 提供 few-shot 示例
- 使用 Chain-of-Thought 提示
- 将复杂任务分解为子任务

### 16.2 性能优化
- 使用流式响应提升用户体验
- 利用提示缓存降低成本和延迟
- 根据任务选择合适的模型
- 合理设置 `max_tokens`

### 16.3 成本控制
- 简单任务使用 Haiku
- 批量任务使用 Batch API
- 充分利用提示缓存
- 监控 token 使用情况

### 16.4 安全建议
- 使用环境变量存储 API Key
- 实现速率限制
- 验证和清理用户输入
- 不要在客户端暴露 API Key
- 使用 `metadata.user_id` 追踪滥用

## 17. 与 OpenAI API 的主要区别

| 特性 | OpenAI | Anthropic |
|------|--------|-----------|
| 认证 Header | Authorization: Bearer | x-api-key |
| 版本 Header | 无 | anthropic-version |
| max_tokens | 可选 | **必需** |
| System Prompt | messages 中 | 顶层 `system` 参数 |
| 工具定义 | tools.function | tools（直接定义） |
| 图像格式 | image_url | image.source |
| 流式事件 | data: {delta} | event: type, data: {...} |
| 响应格式 | choices[0].message | content[0] |
| 扩展思考 | o1 模型内置 | thinking 参数 |

## 18. 常见问题 FAQ

### 18.1 为什么 max_tokens 是必需的？
Anthropic 要求明确指定 `max_tokens` 以防止意外的高成本和确保可预测的行为。

### 18.2 如何减少延迟？
- 使用流式响应
- 启用提示缓存
- 使用 Haiku 模型
- 减少上下文长度

### 18.3 支持哪些编程语言？
- 官方 SDK：Python、TypeScript
- 社区 SDK：Java、Go、Ruby 等
- REST API：所有能发起 HTTP 请求的语言

### 18.4 如何处理长文档？
- Claude 支持 200K token 上下文窗口
- 使用 PDF 支持功能（最多 100 页）
- 使用提示缓存减少重复文档的成本
- 考虑文档分块策略

### 18.5 批量 API 多久完成？
- 大多数批次在 1 小时内完成
- 最多 24 小时
- 可通过 API 查询状态

## 信息来源

- [Claude API Documentation](https://docs.anthropic.com/en/api/overview)
- [Messages API Reference](https://docs.anthropic.com/en/api/messages)
- [Using the Messages API](https://platform.claude.com/docs/en/build-with-claude/working-with-messages)
- [Streaming Messages](https://platform.claude.com/docs/en/build-with-claude/streaming)
- [Tool Use](https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters-anthropic-claude-messages-tool-use.html)
- [Vision Documentation](https://platform.claude.com/docs/en/build-with-claude/vision)
- [PDF Support](https://platform.claude.com/docs/en/build-with-claude/pdf-support)
- [Extended Thinking](https://platform.claude.com/docs/en/build-with-claude/extended-thinking)
- [Prompt Caching](https://platform.claude.com/docs/en/build-with-claude/prompt-caching)
- [Structured Outputs](https://platform.claude.com/docs/en/build-with-claude/structured-outputs)
- [Batch Processing](https://platform.claude.com/docs/en/build-with-claude/batch-processing)
- [Models Overview](https://platform.claude.com/docs/en/about-claude/models/overview)
- [Pricing](https://claude.com/pricing)
- [Rate Limits](https://docs.anthropic.com/en/api/rate-limits)
- [Errors](https://docs.anthropic.com/en/api/errors)
- [Python SDK](https://github.com/anthropics/anthropic-sdk-python)
- [TypeScript SDK](https://github.com/anthropics/anthropic-sdk-typescript)
