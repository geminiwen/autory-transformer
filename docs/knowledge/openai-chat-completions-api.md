# OpenAI Chat Completions API 知识文档

## 1. 概述
- Chat Completions API 是 OpenAI 的核心对话 API
- 支持多轮对话和单次问答
- 端点：`POST https://api.openai.com/v1/chat/completions`
- 支持文本、图像、音频等多模态输入

## 2. 核心端点

### 2.1 创建聊天完成 (Create Chat Completion)
```
POST /v1/chat/completions
```

## 3. 请求参数

### 3.1 必需参数
| 参数 | 类型 | 描述 |
|------|------|------|
| `model` | string | 模型 ID，如 `gpt-4o`, `gpt-4-turbo`, `gpt-3.5-turbo` |
| `messages` | array | 消息数组，描述对话历史 |

### 3.2 常用可选参数
| 参数 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `temperature` | number | 1 | 采样温度 (0-2)，越高越随机 |
| `top_p` | number | 1 | 核采样概率 (0-1) |
| `n` | integer | 1 | 为每个输入生成的完成数量 |
| `max_tokens` | integer | inf | 最大生成 token 数（已弃用，推荐使用 max_completion_tokens） |
| `max_completion_tokens` | integer | - | 完成部分的最大 token 数 |
| `stop` | string/array | null | 停止序列，最多 4 个 |
| `stream` | boolean | false | 是否启用流式响应 |
| `presence_penalty` | number | 0 | 存在惩罚 (-2.0 到 2.0) |
| `frequency_penalty` | number | 0 | 频率惩罚 (-2.0 到 2.0) |
| `logit_bias` | map | null | token 偏置映射 |
| `user` | string | - | 用户唯一标识符 |

### 3.3 函数调用参数
| 参数 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `tools` | array | - | 工具定义数组 |
| `tool_choice` | string/object | "auto" | 工具选择策略 |
| `parallel_tool_calls` | boolean | true | 是否允许并行工具调用 |

### 3.4 结构化输出参数
| 参数 | 类型 | 描述 |
|------|------|------|
| `response_format` | object | 响应格式配置 |

### 3.5 其他参数
| 参数 | 类型 | 描述 |
|------|------|------|
| `seed` | integer | 随机种子，用于确定性采样 |
| `logprobs` | boolean | 是否返回 token 概率 |
| `top_logprobs` | integer | 返回最可能的 N 个 token (0-20) |
| `service_tier` | string | 服务层级：`auto`, `default` |
| `store` | boolean | 是否存储输出用于模型改进 |
| `metadata` | object | 元数据对象 |

## 4. Messages 数组格式

### 4.1 消息角色类型
| 角色 | 描述 |
|------|------|
| `system` | 系统消息，定义助手行为 |
| `user` | 用户消息 |
| `assistant` | 助手消息 |
| `tool` | 工具调用结果消息 |

### 4.2 System 消息
```json
{
  "role": "system",
  "content": "You are a helpful assistant."
}
```

**特点：**
- 可选，但推荐使用
- 通常放在 messages 数组的第一位
- 用于设定助手的行为、角色、风格等

### 4.3 User 消息

#### 纯文本消息
```json
{
  "role": "user",
  "content": "What is the capital of France?"
}
```

#### 多模态消息（文本 + 图像）
```json
{
  "role": "user",
  "content": [
    {
      "type": "text",
      "text": "What's in this image?"
    },
    {
      "type": "image_url",
      "image_url": {
        "url": "https://example.com/image.png",
        "detail": "high"
      }
    }
  ]
}
```

**图像参数：**
- `url`: 图像 URL 或 base64 编码的数据 URI
- `detail`: 图像细节级别
  - `low`: 低分辨率（512x512）
  - `high`: 高分辨率
  - `auto`: 自动选择（默认）

#### 音频输入消息
```json
{
  "role": "user",
  "content": [
    {
      "type": "input_audio",
      "input_audio": {
        "data": "base64_encoded_audio",
        "format": "wav"
      }
    }
  ]
}
```

**音频参数：**
- `data`: base64 编码的音频数据
- `format`: 音频格式（`wav`, `mp3`）

### 4.4 Assistant 消息

#### 普通助手回复
```json
{
  "role": "assistant",
  "content": "The capital of France is Paris."
}
```

#### 带工具调用的助手消息
```json
{
  "role": "assistant",
  "content": null,
  "tool_calls": [
    {
      "id": "call_xxx",
      "type": "function",
      "function": {
        "name": "get_weather",
        "arguments": "{\"location\": \"Paris\"}"
      }
    }
  ]
}
```

### 4.5 Tool 消息
```json
{
  "role": "tool",
  "tool_call_id": "call_xxx",
  "content": "{\"temperature\": 20, \"condition\": \"sunny\"}"
}
```

## 5. 响应格式

### 5.1 非流式响应
```json
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "gpt-4o-2024-05-13",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you today?"
      },
      "finish_reason": "stop",
      "logprobs": null
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 20,
    "total_tokens": 30,
    "prompt_tokens_details": {
      "cached_tokens": 0
    },
    "completion_tokens_details": {
      "reasoning_tokens": 0
    }
  },
  "system_fingerprint": "fp_xxx"
}
```

### 5.2 finish_reason 类型
| 值 | 描述 |
|----|------|
| `stop` | 自然结束或遇到停止序列 |
| `length` | 达到 max_tokens 限制 |
| `tool_calls` | 模型调用了工具 |
| `content_filter` | 内容被过滤 |
| `function_call` | 模型调用了函数（已弃用） |

### 5.3 使用统计 (Usage)
```json
{
  "usage": {
    "prompt_tokens": 100,
    "completion_tokens": 200,
    "total_tokens": 300,
    "prompt_tokens_details": {
      "cached_tokens": 50
    },
    "completion_tokens_details": {
      "reasoning_tokens": 150
    }
  }
}
```

**字段说明：**
- `prompt_tokens`: 输入 token 数
- `completion_tokens`: 输出 token 数
- `total_tokens`: 总 token 数
- `cached_tokens`: 缓存的 token 数（提示缓存）
- `reasoning_tokens`: 推理 token 数（o1/o3 模型）

## 6. 流式响应 (Streaming)

### 6.1 启用流式
```json
{
  "model": "gpt-4o",
  "messages": [...],
  "stream": true
}
```

### 6.2 流式响应格式
```
data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}

data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":null}]}

data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

### 6.3 流式 Delta 对象
```json
{
  "delta": {
    "role": "assistant",
    "content": "text chunk",
    "tool_calls": [...]
  }
}
```

### 6.4 流式选项
```json
{
  "stream": true,
  "stream_options": {
    "include_usage": true
  }
}
```

**stream_options 参数：**
- `include_usage`: 是否在最后一个 chunk 中包含使用统计

## 7. 函数调用 (Function Calling)

### 7.1 定义工具
```json
{
  "model": "gpt-4o",
  "messages": [...],
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_current_weather",
        "description": "Get the current weather in a given location",
        "parameters": {
          "type": "object",
          "properties": {
            "location": {
              "type": "string",
              "description": "The city and state, e.g. San Francisco, CA"
            },
            "unit": {
              "type": "string",
              "enum": ["celsius", "fahrenheit"],
              "description": "The temperature unit"
            }
          },
          "required": ["location"],
          "additionalProperties": false
        },
        "strict": true
      }
    }
  ]
}
```

**函数参数说明：**
- `name`: 函数名称（a-z, A-Z, 0-9, 下划线和连字符，最多 64 字符）
- `description`: 函数描述（帮助模型决定何时调用）
- `parameters`: JSON Schema 格式的参数定义
- `strict`: 是否启用严格模式（确保参数符合 schema）

### 7.2 tool_choice 选项
```json
// 自动选择（默认）
{"tool_choice": "auto"}

// 强制调用任意工具
{"tool_choice": "required"}

// 不调用工具
{"tool_choice": "none"}

// 强制调用指定工具
{
  "tool_choice": {
    "type": "function",
    "function": {"name": "get_current_weather"}
  }
}
```

### 7.3 处理工具调用响应
```json
{
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": null,
        "tool_calls": [
          {
            "id": "call_abc123",
            "type": "function",
            "function": {
              "name": "get_current_weather",
              "arguments": "{\"location\": \"San Francisco, CA\", \"unit\": \"fahrenheit\"}"
            }
          }
        ]
      },
      "finish_reason": "tool_calls"
    }
  ]
}
```

### 7.4 提供工具结果
```json
{
  "model": "gpt-4o",
  "messages": [
    {"role": "user", "content": "What's the weather in SF?"},
    {
      "role": "assistant",
      "content": null,
      "tool_calls": [
        {
          "id": "call_abc123",
          "type": "function",
          "function": {
            "name": "get_current_weather",
            "arguments": "{\"location\": \"San Francisco, CA\"}"
          }
        }
      ]
    },
    {
      "role": "tool",
      "tool_call_id": "call_abc123",
      "content": "{\"temperature\": 22, \"condition\": \"sunny\"}"
    }
  ]
}
```

### 7.5 并行工具调用
```json
{
  "parallel_tool_calls": true
}
```

当设置为 `true` 时，模型可以在一次响应中调用多个工具：
```json
{
  "tool_calls": [
    {
      "id": "call_1",
      "type": "function",
      "function": {"name": "get_weather", "arguments": "{\"location\": \"Paris\"}"}
    },
    {
      "id": "call_2",
      "type": "function",
      "function": {"name": "get_weather", "arguments": "{\"location\": \"Tokyo\"}"}
    }
  ]
}
```

## 8. 结构化输出 (Structured Outputs)

### 8.1 JSON 模式
```json
{
  "model": "gpt-4o",
  "messages": [...],
  "response_format": {
    "type": "json_object"
  }
}
```

**注意：** 使用 `json_object` 时，必须在消息中明确要求模型生成 JSON。

### 8.2 JSON Schema 模式（严格）
```json
{
  "model": "gpt-4o",
  "messages": [...],
  "response_format": {
    "type": "json_schema",
    "json_schema": {
      "name": "math_response",
      "strict": true,
      "schema": {
        "type": "object",
        "properties": {
          "steps": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "explanation": {"type": "string"},
                "output": {"type": "string"}
              },
              "required": ["explanation", "output"],
              "additionalProperties": false
            }
          },
          "final_answer": {"type": "string"}
        },
        "required": ["steps", "final_answer"],
        "additionalProperties": false
      }
    }
  }
}
```

**strict 模式特点：**
- 保证输出严格符合 schema
- 所有对象必须设置 `"additionalProperties": false`
- 所有字段必须在 `required` 中声明
- 支持的类型：string, number, boolean, integer, object, array, enum, anyOf

### 8.3 文本格式
```json
{
  "response_format": {
    "type": "text"
  }
}
```

## 9. 提示缓存 (Prompt Caching)

### 9.1 自动缓存
- OpenAI 自动缓存最近使用的提示前缀
- 缓存有效期：5-10 分钟
- 缓存命中时 `cached_tokens` > 0
- 缓存的 token 价格为正常价格的 50%

### 9.2 查看缓存统计
```json
{
  "usage": {
    "prompt_tokens": 1000,
    "completion_tokens": 100,
    "total_tokens": 1100,
    "prompt_tokens_details": {
      "cached_tokens": 800
    }
  }
}
```

### 9.3 优化缓存使用
- 将不变的上下文放在消息数组前面
- 保持提示前缀稳定
- 在 5-10 分钟内复用相同的提示前缀

## 10. 推理模型 (o1, o3 系列)

### 10.1 支持的模型
- `o1-preview`
- `o1-mini`
- `o3-mini`

### 10.2 特殊参数
```json
{
  "model": "o3-mini",
  "messages": [...],
  "reasoning_effort": "medium"
}
```

**reasoning_effort 选项：**
- `low`: 低推理努力（快速，便宜）
- `medium`: 中等推理努力（平衡，默认）
- `high`: 高推理努力（深度思考，昂贵）

### 10.3 限制
推理模型不支持以下参数：
- `temperature`
- `top_p`
- `presence_penalty`
- `frequency_penalty`
- `logprobs`
- `tools` (o1 系列)
- `stream`（仅 o1-preview 和 o1-mini，o3-mini 支持）

### 10.4 推理 Token
```json
{
  "usage": {
    "completion_tokens_details": {
      "reasoning_tokens": 512
    }
  }
}
```

**注意：** 推理 token 计入总使用量，但通常不在响应内容中显示。

## 11. 音频输出 (Audio Output)

### 11.1 启用音频输出
```json
{
  "model": "gpt-4o-audio-preview",
  "modalities": ["text", "audio"],
  "audio": {
    "voice": "alloy",
    "format": "wav"
  },
  "messages": [...]
}
```

### 11.2 音频参数
| 参数 | 类型 | 选项 | 描述 |
|------|------|------|------|
| `voice` | string | alloy, echo, fable, onyx, nova, shimmer | 语音选择 |
| `format` | string | wav, mp3, flac, opus, pcm16 | 音频格式 |

### 11.3 音频响应格式
```json
{
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": "Hello!",
        "audio": {
          "id": "audio_xxx",
          "data": "base64_encoded_audio",
          "expires_at": 1234567890,
          "transcript": "Hello!"
        }
      }
    }
  ]
}
```

## 12. 模型列表

### 12.1 GPT-4o 系列
| 模型 | 描述 | 上下文窗口 | 最大输出 |
|------|------|-----------|---------|
| `gpt-4o` | 最新的高智能旗舰模型 | 128K | 16K |
| `gpt-4o-2024-11-20` | 特定快照版本 | 128K | 16K |
| `gpt-4o-2024-08-06` | 支持结构化输出 | 128K | 16K |
| `gpt-4o-2024-05-13` | 第一个版本 | 128K | 4K |
| `gpt-4o-mini` | 经济实惠的小型模型 | 128K | 16K |
| `gpt-4o-mini-2024-07-18` | 特定快照版本 | 128K | 16K |
| `gpt-4o-audio-preview` | 支持音频输入输出 | 128K | 16K |

### 12.2 GPT-4 Turbo 系列
| 模型 | 描述 | 上下文窗口 | 最大输出 |
|------|------|-----------|---------|
| `gpt-4-turbo` | 最新 GPT-4 Turbo | 128K | 4K |
| `gpt-4-turbo-2024-04-09` | 特定快照版本 | 128K | 4K |
| `gpt-4-turbo-preview` | 预览版本 | 128K | 4K |

### 12.3 GPT-4 系列
| 模型 | 描述 | 上下文窗口 | 最大输出 |
|------|------|-----------|---------|
| `gpt-4` | 标准 GPT-4 | 8K | 8K |
| `gpt-4-0613` | 特定快照版本 | 8K | 8K |
| `gpt-4-32k` | 更大上下文版本 | 32K | 32K |

### 12.4 GPT-3.5 Turbo 系列
| 模型 | 描述 | 上下文窗口 | 最大输出 |
|------|------|-----------|---------|
| `gpt-3.5-turbo` | 经济的快速模型 | 16K | 4K |
| `gpt-3.5-turbo-0125` | 特定快照版本 | 16K | 4K |
| `gpt-3.5-turbo-1106` | 支持并行函数调用 | 16K | 4K |

### 12.5 推理模型
| 模型 | 描述 | 上下文窗口 | 最大输出 |
|------|------|-----------|---------|
| `o1-preview` | 推理预览模型 | 128K | 32K |
| `o1-mini` | 更快的推理模型 | 128K | 64K |
| `o3-mini` | 最新的推理模型 | 200K | 100K |

## 13. SDK 使用示例

### 13.1 Python SDK

#### 基本用法
```python
from openai import OpenAI

client = OpenAI()

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "Hello!"}
    ]
)

print(response.choices[0].message.content)
```

#### 流式响应
```python
stream = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Tell me a story"}],
    stream=True
)

for chunk in stream:
    if chunk.choices[0].delta.content is not None:
        print(chunk.choices[0].delta.content, end="")
```

#### 函数调用
```python
tools = [
    {
        "type": "function",
        "function": {
            "name": "get_weather",
            "description": "Get the current weather",
            "parameters": {
                "type": "object",
                "properties": {
                    "location": {"type": "string"}
                },
                "required": ["location"]
            }
        }
    }
]

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "What's the weather in Tokyo?"}],
    tools=tools,
    tool_choice="auto"
)

# 处理工具调用
if response.choices[0].message.tool_calls:
    tool_call = response.choices[0].message.tool_calls[0]
    print(f"Function: {tool_call.function.name}")
    print(f"Arguments: {tool_call.function.arguments}")
```

#### 结构化输出
```python
from pydantic import BaseModel

class Step(BaseModel):
    explanation: str
    output: str

class MathResponse(BaseModel):
    steps: list[Step]
    final_answer: str

completion = client.beta.chat.completions.parse(
    model="gpt-4o-2024-08-06",
    messages=[
        {"role": "system", "content": "You are a helpful math tutor."},
        {"role": "user", "content": "Solve 8x + 31 = 2"}
    ],
    response_format=MathResponse
)

response = completion.choices[0].message.parsed
print(response.final_answer)
```

#### 视觉（图像输入）
```python
response = client.chat.completions.create(
    model="gpt-4o",
    messages=[
        {
            "role": "user",
            "content": [
                {"type": "text", "text": "What's in this image?"},
                {
                    "type": "image_url",
                    "image_url": {
                        "url": "https://example.com/image.png",
                        "detail": "high"
                    }
                }
            ]
        }
    ]
)
```

### 13.2 Node.js SDK

#### 基本用法
```javascript
import OpenAI from 'openai';

const client = new OpenAI();

const response = await client.chat.completions.create({
    model: "gpt-4o",
    messages: [
        { role: "system", content: "You are a helpful assistant." },
        { role: "user", content: "Hello!" }
    ]
});

console.log(response.choices[0].message.content);
```

#### 流式响应
```javascript
const stream = await client.chat.completions.create({
    model: "gpt-4o",
    messages: [{ role: "user", content: "Tell me a story" }],
    stream: true
});

for await (const chunk of stream) {
    process.stdout.write(chunk.choices[0]?.delta?.content || '');
}
```

#### 函数调用
```javascript
const tools = [
    {
        type: "function",
        function: {
            name: "get_weather",
            description: "Get the current weather",
            parameters: {
                type: "object",
                properties: {
                    location: { type: "string" }
                },
                required: ["location"]
            }
        }
    }
];

const response = await client.chat.completions.create({
    model: "gpt-4o",
    messages: [{ role: "user", content: "What's the weather in Tokyo?" }],
    tools: tools,
    tool_choice: "auto"
});

if (response.choices[0].message.tool_calls) {
    const toolCall = response.choices[0].message.tool_calls[0];
    console.log(`Function: ${toolCall.function.name}`);
    console.log(`Arguments: ${toolCall.function.arguments}`);
}
```

#### 结构化输出
```javascript
const completion = await client.beta.chat.completions.parse({
    model: "gpt-4o-2024-08-06",
    messages: [
        { role: "system", content: "You are a helpful math tutor." },
        { role: "user", content: "Solve 8x + 31 = 2" }
    ],
    response_format: {
        type: "json_schema",
        json_schema: {
            name: "math_response",
            strict: true,
            schema: {
                type: "object",
                properties: {
                    steps: {
                        type: "array",
                        items: {
                            type: "object",
                            properties: {
                                explanation: { type: "string" },
                                output: { type: "string" }
                            },
                            required: ["explanation", "output"],
                            additionalProperties: false
                        }
                    },
                    final_answer: { type: "string" }
                },
                required: ["steps", "final_answer"],
                additionalProperties: false
            }
        }
    }
});

const response = completion.choices[0].message.parsed;
console.log(response.final_answer);
```

### 13.3 cURL 示例

#### 基本请求
```bash
curl https://api.openai.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{
    "model": "gpt-4o",
    "messages": [
      {
        "role": "system",
        "content": "You are a helpful assistant."
      },
      {
        "role": "user",
        "content": "Hello!"
      }
    ]
  }'
```

#### 流式请求
```bash
curl https://api.openai.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Say hello"}],
    "stream": true
  }'
```

## 14. 最佳实践

### 14.1 提示工程
- 使用清晰、具体的指令
- 提供示例（few-shot learning）
- 指定输出格式
- 使用分隔符区分不同部分
- 要求模型逐步思考（chain-of-thought）

### 14.2 性能优化
- 使用提示缓存减少成本
- 选择合适的模型（不一定总用最大的）
- 使用 max_tokens 限制输出长度
- 考虑使用 gpt-4o-mini 处理简单任务

### 14.3 错误处理
- 处理速率限制（429 错误）
- 实现重试逻辑（指数退避）
- 验证输入以避免无效请求
- 处理内容过滤（content_filter finish_reason）

### 14.4 安全性
- 不要在提示中包含敏感信息
- 验证和清理用户输入
- 实现输出验证
- 使用 `user` 参数追踪滥用

## 15. 价格（2024 年参考）

### 15.1 GPT-4o 系列
| 模型 | 输入价格 | 输出价格 | 缓存价格 |
|------|---------|---------|---------|
| gpt-4o | $2.50/1M tokens | $10.00/1M tokens | $1.25/1M tokens |
| gpt-4o-mini | $0.15/1M tokens | $0.60/1M tokens | $0.075/1M tokens |

### 15.2 GPT-4 Turbo
| 模型 | 输入价格 | 输出价格 |
|------|---------|---------|
| gpt-4-turbo | $10.00/1M tokens | $30.00/1M tokens |

### 15.3 推理模型
| 模型 | 输入价格 | 输出价格 | 推理 token 价格 |
|------|---------|---------|----------------|
| o1-preview | $15.00/1M tokens | $60.00/1M tokens | $15.00/1M tokens |
| o1-mini | $3.00/1M tokens | $12.00/1M tokens | $3.00/1M tokens |
| o3-mini (low) | $1.10/1M tokens | $4.40/1M tokens | 包含在输出中 |
| o3-mini (medium) | $1.10/1M tokens | $4.40/1M tokens | 包含在输出中 |
| o3-mini (high) | $1.10/1M tokens | $4.40/1M tokens | 包含在输出中 |

**注意：** 价格可能变动，请查看官方文档获取最新价格。

## 16. 限制与配额

### 16.1 速率限制
- TPM (Tokens Per Minute): 每分钟 token 数
- RPM (Requests Per Minute): 每分钟请求数
- RPD (Requests Per Day): 每日请求数

限制因账户层级而异：
- Free tier: 较低限制
- Pay-as-you-go: 中等限制
- Enterprise: 可定制更高限制

### 16.2 上下文窗口限制
- 不同模型有不同的上下文窗口
- 输入 + 输出 token 总数不能超过上下文窗口
- 超出限制会返回 400 错误

### 16.3 其他限制
- 图像数量：每个请求最多处理多张图像（取决于模型）
- 工具数量：最多 128 个工具
- 停止序列：最多 4 个
- messages 数组：建议不超过数千条消息

## 17. 常见错误处理

### 17.1 错误代码
| 代码 | 描述 | 解决方案 |
|------|------|---------|
| 401 | 无效的 API 密钥 | 检查 API 密钥是否正确 |
| 429 | 速率限制或配额超限 | 实现重试逻辑，等待后重试 |
| 500 | 服务器错误 | 稍后重试 |
| 503 | 服务不可用 | 稍后重试 |
| 400 | 无效请求 | 检查请求参数 |

### 17.2 重试逻辑示例（Python）
```python
import time
from openai import OpenAI, RateLimitError

client = OpenAI()

def chat_with_retry(messages, max_retries=3):
    for attempt in range(max_retries):
        try:
            return client.chat.completions.create(
                model="gpt-4o",
                messages=messages
            )
        except RateLimitError:
            if attempt < max_retries - 1:
                wait_time = 2 ** attempt
                print(f"Rate limited. Waiting {wait_time} seconds...")
                time.sleep(wait_time)
            else:
                raise
```

## 信息来源

- [OpenAI API Reference - Chat Completions](https://platform.openai.com/docs/api-reference/chat)
- [Chat Completions Guide](https://platform.openai.com/docs/guides/chat-completions)
- [Function Calling](https://platform.openai.com/docs/guides/function-calling)
- [Structured Outputs](https://platform.openai.com/docs/guides/structured-outputs)
- [Vision](https://platform.openai.com/docs/guides/vision)
- [Audio](https://platform.openai.com/docs/guides/audio)
- [Prompt Caching](https://platform.openai.com/docs/guides/prompt-caching)
- [Models](https://platform.openai.com/docs/models)
- [Error Codes](https://platform.openai.com/docs/guides/error-codes)
- [Rate Limits](https://platform.openai.com/docs/guides/rate-limits)
