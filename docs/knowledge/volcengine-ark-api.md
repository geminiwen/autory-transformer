# 火山引擎方舟 API 知识文档

## 1. 概述

### 1.1 平台介绍
- **火山方舟**是字节跳动火山引擎推出的一站式大模型服务平台
- 提供模型推理、评估、精调全流程服务
- 支持**豆包（Doubao）**系列模型和多种主流大模型
- API 高度兼容 OpenAI SDK，方便快速迁移

### 1.2 主要特性
- 高并发支持（TPM 可达 500 万）
- 多模态能力（文本、图像、视频理解）
- 丰富的内置工具（联网搜索、知识库、函数调用等）
- 支持 Chat API 和 Responses API 两种接口
- 上下文缓存功能降低成本

### 1.3 SDK 版本说明
- **SDK V3**：当前推荐版本，完全兼容 OpenAI 协议
- **SDK V1/V2**：已于 2024 年 11 月 30 日下线
- 建议使用 V3 接口或直接使用 OpenAI SDK

## 2. 核心端点

### 2.1 Chat API（对话接口）
```
POST https://ark.cn-beijing.volces.com/api/v3/chat/completions
```

### 2.2 Responses API（响应接口）
```
POST https://ark.cn-beijing.volces.com/api/v3/responses
GET  https://ark.cn-beijing.volces.com/api/v3/responses/{response_id}
DELETE https://ark.cn-beijing.volces.com/api/v3/responses/{response_id}
```

### 2.3 Batch API（批量接口）
```
POST https://ark.cn-beijing.volces.com/api/v3/batch
```

## 3. 认证方式

### 3.1 API Key 获取
1. 登录火山引擎控制台
2. 进入火山方舟页面
3. 在"API Key 管理"菜单中创建 API Key

### 3.2 认证方式
使用 HTTP Header 进行认证：
```
Authorization: Bearer YOUR_API_KEY
```

### 3.3 环境变量设置
```bash
export ARK_API_KEY="your-api-key-here"
```

## 4. Chat API 详解

### 4.1 请求参数

#### 必需参数
| 参数 | 类型 | 描述 |
|------|------|------|
| `model` | string | 模型 Endpoint ID（格式：ep-xxxxxxxxxx-xxxxx） |
| `messages` | array | 消息数组 |

#### 可选参数
| 参数 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `temperature` | number | 1.0 | 采样温度 (0-2) |
| `top_p` | number | 1.0 | 核采样概率 (0-1) |
| `max_tokens` | integer | - | 最大输出 token 数 |
| `stream` | boolean | false | 是否启用流式响应 |
| `stop` | string/array | - | 停止序列 |
| `presence_penalty` | number | 0 | 存在惩罚 (-2.0 到 2.0) |
| `frequency_penalty` | number | 0 | 频率惩罚 (-2.0 到 2.0) |
| `n` | integer | 1 | 生成的完成数量 |
| `user` | string | - | 用户标识符 |
| `tools` | array | - | 工具定义数组 |
| `tool_choice` | string/object | "auto" | 工具选择策略 |

### 4.2 Messages 格式

#### 系统消息
```json
{
  "role": "system",
  "content": "你是豆包，是由字节跳动开发的 AI 人工智能助手"
}
```

#### 用户消息（纯文本）
```json
{
  "role": "user",
  "content": "你好"
}
```

#### 用户消息（多模态）
```json
{
  "role": "user",
  "content": [
    {
      "type": "text",
      "text": "这张图片里有什么？"
    },
    {
      "type": "image_url",
      "image_url": {
        "url": "https://example.com/image.png"
      }
    }
  ]
}
```

#### 助手消息
```json
{
  "role": "assistant",
  "content": "你好！我是豆包，很高兴为你服务。"
}
```

### 4.3 响应格式

#### 非流式响应
```json
{
  "id": "021730896918756a0f9b9ad2029****",
  "object": "chat.completion",
  "created": 1730896926,
  "model": "doubao-pro-32k",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "你好！我是豆包，很高兴为你服务。"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 15,
    "total_tokens": 25,
    "prompt_tokens_details": {
      "cached_tokens": 0
    }
  },
  "service_tier": "default"
}
```

#### finish_reason 类型
| 值 | 描述 |
|----|------|
| `stop` | 自然结束 |
| `length` | 达到 max_tokens 限制 |
| `tool_calls` | 调用了工具 |
| `content_filter` | 内容被过滤 |

#### 流式响应格式
```
data: {"id":"xxx","object":"chat.completion.chunk","created":1234567890,"model":"doubao-pro-32k","choices":[{"index":0,"delta":{"role":"assistant","content":"你"},"finish_reason":null}]}

data: {"id":"xxx","object":"chat.completion.chunk","created":1234567890,"model":"doubao-pro-32k","choices":[{"index":0,"delta":{"content":"好"},"finish_reason":null}]}

data: [DONE]
```

## 5. Responses API 详解

### 5.1 概述
Responses API 是火山方舟提供的新一代 API，支持更丰富的功能：
- 内置工具调用（联网搜索、知识库、图像处理等）
- 会话管理
- 后台模式
- 更灵活的响应格式

### 5.2 创建响应

#### 基本请求
```json
{
  "model": "ep-xxxxxxxxxx-xxxxx",
  "input": "你好",
  "instructions": "你是豆包，是由字节跳动开发的 AI 人工智能助手"
}
```

#### 带工具的请求
```json
{
  "model": "ep-xxxxxxxxxx-xxxxx",
  "input": "今天北京的天气怎么样？",
  "tools": [
    {
      "type": "web_search"
    }
  ]
}
```

### 5.3 响应格式
```json
{
  "id": "resp_xxx",
  "object": "response",
  "created_at": 1234567890,
  "status": "completed",
  "model": "doubao-pro-32k",
  "output": [
    {
      "type": "message",
      "content": [
        {
          "type": "output_text",
          "text": "今天北京天气晴朗..."
        }
      ]
    }
  ],
  "output_text": "今天北京天气晴朗...",
  "usage": {
    "input_tokens": 20,
    "output_tokens": 50,
    "total_tokens": 70
  }
}
```

## 6. 豆包模型列表

### 6.1 Doubao 主要模型

#### Doubao-pro 系列（高性能）
| 模型 | 上下文窗口 | 描述 |
|------|-----------|------|
| doubao-pro-4k | 4K | 高性能模型，短上下文 |
| doubao-pro-8k | 8K | 高性能模型，中等上下文 |
| doubao-pro-32k | 32K | 高性能模型，长上下文 |
| doubao-pro-128k | 128K | 高性能模型，超长上下文 |
| doubao-pro-256k | 256K | 高性能模型，极长上下文 |

#### Doubao-lite 系列（经济型）
| 模型 | 上下文窗口 | 描述 |
|------|-----------|------|
| doubao-lite-4k | 4K | 经济型模型，短上下文 |
| doubao-lite-8k | 8K | 经济型模型，中等上下文 |
| doubao-lite-32k | 32K | 经济型模型，长上下文 |
| doubao-lite-128k | 128K | 经济型模型，超长上下文 |

#### Doubao-vision 系列（视觉理解）
| 模型 | 上下文窗口 | 描述 |
|------|-----------|------|
| doubao-1.5-vision-pro-32k | 32K | 视觉理解，支持图像输入 |
| doubao-vision-pro-32k | 32K | 视觉理解，支持图像输入 |

#### 其他模型
| 模型 | 描述 |
|------|------|
| doubao-seed-code | 代码生成专用模型 |
| deepseek-r1 | DeepSeek R1 推理模型 |

### 6.2 Endpoint ID 说明
- 模型不能直接使用，需要先创建 Endpoint（推理接入点）
- Endpoint ID 格式：`ep-xxxxxxxxxx-xxxxx`
- 在控制台的"模型推理"页面创建 Endpoint
- 创建后在列表中可以看到 Endpoint ID

## 7. 函数调用 (Function Calling)

### 7.1 定义函数
```json
{
  "model": "ep-xxxxxxxxxx-xxxxx",
  "messages": [
    {"role": "user", "content": "北京今天天气怎么样？"}
  ],
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_weather",
        "description": "获取指定城市的天气信息",
        "parameters": {
          "type": "object",
          "properties": {
            "location": {
              "type": "string",
              "description": "城市名称，例如：北京、上海"
            }
          },
          "required": ["location"]
        }
      }
    }
  ]
}
```

### 7.2 tool_choice 选项
```json
// 自动选择（默认）
{"tool_choice": "auto"}

// 强制调用工具
{"tool_choice": "required"}

// 不调用工具
{"tool_choice": "none"}

// 强制调用指定工具
{
  "tool_choice": {
    "type": "function",
    "function": {"name": "get_weather"}
  }
}
```

### 7.3 处理函数调用响应
```json
{
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": null,
        "tool_calls": [
          {
            "id": "call_xxx",
            "type": "function",
            "function": {
              "name": "get_weather",
              "arguments": "{\"location\": \"北京\"}"
            }
          }
        ]
      },
      "finish_reason": "tool_calls"
    }
  ]
}
```

### 7.4 提供函数结果
```json
{
  "model": "ep-xxxxxxxxxx-xxxxx",
  "messages": [
    {"role": "user", "content": "北京今天天气怎么样？"},
    {
      "role": "assistant",
      "content": null,
      "tool_calls": [
        {
          "id": "call_xxx",
          "type": "function",
          "function": {
            "name": "get_weather",
            "arguments": "{\"location\": \"北京\"}"
          }
        }
      ]
    },
    {
      "role": "tool",
      "tool_call_id": "call_xxx",
      "content": "{\"temperature\": 15, \"condition\": \"晴\"}"
    }
  ]
}
```

## 8. 内置工具（Responses API）

### 8.1 联网搜索 (Web Search)
```json
{
  "model": "ep-xxxxxxxxxx-xxxxx",
  "input": "2026年最新的AI发展趋势",
  "tools": [
    {
      "type": "web_search"
    }
  ]
}
```

**特点：**
- 实时获取互联网信息
- 自动引用来源
- 支持时效性查询

### 8.2 知识库搜索 (Knowledge Search)
```json
{
  "model": "ep-xxxxxxxxxx-xxxxx",
  "input": "公司的年假政策是什么？",
  "tools": [
    {
      "type": "knowledge_search",
      "knowledge_base_id": "kb_xxx"
    }
  ]
}
```

**特点：**
- 检索私域知识库
- 支持文档上传和管理
- 提供引用溯源

### 8.3 图像处理 (Image Process)
```json
{
  "model": "ep-xxxxxxxxxx-xxxxx",
  "input": "帮我生成一张夕阳下的海滩图片",
  "tools": [
    {
      "type": "image_process"
    }
  ]
}
```

**功能：**
- 图像生成
- 图像编辑
- 图像理解

### 8.4 豆包助手
```json
{
  "model": "ep-xxxxxxxxxx-xxxxx",
  "input": "帮我安排一下明天的日程",
  "tools": [
    {
      "type": "doubao_assistant"
    }
  ]
}
```

### 8.5 MCP 服务器
```json
{
  "model": "ep-xxxxxxxxxx-xxxxx",
  "input": "查询数据库中的用户信息",
  "tools": [
    {
      "type": "mcp",
      "server_url": "https://your-mcp-server.com"
    }
  ]
}
```

## 9. 视觉理解 (Vision)

### 9.1 图像理解
```json
{
  "model": "ep-xxxxxxxxxx-xxxxx",
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "描述这张图片的内容"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://example.com/image.png"
          }
        }
      ]
    }
  ]
}
```

### 9.2 支持的图像格式
- URL 引用（https://）
- Base64 编码（data:image/png;base64,...）

### 9.3 视觉模型能力
- 物体识别和检测
- 场景理解
- 文字识别（OCR）
- 图表分析
- 详细的视觉描述

### 9.4 多图理解
```json
{
  "role": "user",
  "content": [
    {"type": "text", "text": "比较这两张图片的异同"},
    {"type": "image_url", "image_url": {"url": "https://example.com/image1.png"}},
    {"type": "image_url", "image_url": {"url": "https://example.com/image2.png"}}
  ]
}
```

## 10. 上下文缓存

### 10.1 概述
- 自动缓存重复的提示前缀
- 缓存有效期：约 5-10 分钟
- 缓存命中时 `cached_tokens` > 0
- 降低计费成本

### 10.2 查看缓存统计
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

### 10.3 优化建议
- 将不变的上下文放在前面
- 保持提示前缀稳定
- 在缓存有效期内复用

## 11. SDK 使用示例

### 11.1 Python SDK

#### 安装
```bash
pip install 'volcengine-python-sdk[ark]'
```

#### 基本用法
```python
from volcenginesdkarkruntime import Ark

client = Ark(
    base_url="https://ark.cn-beijing.volces.com/api/v3",
    api_key="your-api-key"
)

# 非流式
completion = client.chat.completions.create(
    model="ep-xxxxxxxxxx-xxxxx",
    messages=[
        {"role": "system", "content": "你是豆包，由字节跳动开发的AI助手"},
        {"role": "user", "content": "你好"}
    ]
)

print(completion.choices[0].message.content)
```

#### 流式响应
```python
stream = client.chat.completions.create(
    model="ep-xxxxxxxxxx-xxxxx",
    messages=[
        {"role": "user", "content": "讲个故事"}
    ],
    stream=True
)

for chunk in stream:
    if chunk.choices[0].delta.content is not None:
        print(chunk.choices[0].delta.content, end="")
```

#### 使用 OpenAI SDK
```python
from openai import OpenAI

client = OpenAI(
    base_url="https://ark.cn-beijing.volces.com/api/v3",
    api_key="your-ark-api-key"
)

completion = client.chat.completions.create(
    model="ep-xxxxxxxxxx-xxxxx",
    messages=[
        {"role": "user", "content": "你好"}
    ]
)

print(completion.choices[0].message.content)
```

### 11.2 Java SDK

#### Maven 依赖
```xml
<dependency>
    <groupId>com.volcengine</groupId>
    <artifactId>volcengine-java-sdk-ark-runtime</artifactId>
    <version>LATEST</version>
</dependency>
```

#### 基本用法
```java
import com.volcengine.ark.runtime.model.completion.chat.ChatCompletionRequest;
import com.volcengine.ark.runtime.model.completion.chat.ChatMessage;
import com.volcengine.ark.runtime.model.completion.chat.ChatMessageRole;
import com.volcengine.ark.runtime.service.ArkService;

public class Main {
    public static void main(String[] args) {
        ArkService service = new ArkService();
        service.setApiKey("your-api-key");

        ChatCompletionRequest request = ChatCompletionRequest.builder()
            .model("ep-xxxxxxxxxx-xxxxx")
            .message(ChatMessage.builder()
                .role(ChatMessageRole.USER)
                .content("你好")
                .build())
            .build();

        service.createChatCompletion(request)
            .getChoices()
            .forEach(choice ->
                System.out.println(choice.getMessage().getContent())
            );
    }
}
```

### 11.3 Go SDK

#### 安装
```bash
go get github.com/volcengine/volcengine-go-sdk/service/arkruntime
```

#### 基本用法
```go
package main

import (
    "context"
    "fmt"
    "github.com/volcengine/volcengine-go-sdk/service/arkruntime"
    "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

func main() {
    client := arkruntime.NewClient(
        arkruntime.WithBaseUrl("https://ark.cn-beijing.volces.com/api/v3"),
        arkruntime.WithApiKey("your-api-key"),
    )

    ctx := context.Background()

    req := &model.ChatCompletionRequest{
        Model: "ep-xxxxxxxxxx-xxxxx",
        Messages: []*model.ChatCompletionMessage{
            {
                Role:    model.ChatMessageRoleUser,
                Content: "你好",
            },
        },
    }

    resp, err := client.CreateChatCompletion(ctx, req)
    if err != nil {
        panic(err)
    }

    fmt.Println(resp.Choices[0].Message.Content)
}
```

### 11.4 cURL 示例

#### 基本请求
```bash
curl https://ark.cn-beijing.volces.com/api/v3/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARK_API_KEY" \
  -d '{
    "model": "ep-xxxxxxxxxx-xxxxx",
    "messages": [
      {
        "role": "user",
        "content": "你好"
      }
    ]
  }'
```

#### 流式请求
```bash
curl https://ark.cn-beijing.volces.com/api/v3/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARK_API_KEY" \
  -d '{
    "model": "ep-xxxxxxxxxx-xxxxx",
    "messages": [{"role": "user", "content": "你好"}],
    "stream": true
  }'
```

## 12. 计费说明

### 12.1 计费方式
- 按 token 后付费
- 分为输入 token 和输出 token
- 缓存命中的 token 价格更低

### 12.2 计费组成
| 类型 | 对应字段 | 说明 |
|------|---------|------|
| 推理输入 | prompt_tokens | 输入内容转换的 token 数 |
| 推理输出 | completion_tokens | 输出内容转换的 token 数 |
| 缓存命中 | cached_tokens | 命中缓存的 token 数（价格更低） |

### 12.3 参考价格（以 Doubao 为例）

#### Doubao-pro 系列
| 模型 | 输入价格 | 输出价格 | 缓存价格 |
|------|---------|---------|---------|
| doubao-pro-4k | 0.8 元/百万 token | 2 元/百万 token | 约 0.4 元/百万 token |
| doubao-pro-32k | 0.8 元/百万 token | 2 元/百万 token | 约 0.4 元/百万 token |
| doubao-pro-128k | 5 元/百万 token | 9 元/百万 token | 约 2.5 元/百万 token |

#### Doubao-lite 系列
| 模型 | 输入价格 | 输出价格 | 缓存价格 |
|------|---------|---------|---------|
| doubao-lite-4k | 0.3 元/百万 token | 0.6 元/百万 token | 约 0.15 元/百万 token |
| doubao-lite-32k | 0.3 元/百万 token | 0.6 元/百万 token | 约 0.15 元/百万 token |
| doubao-lite-128k | 1 元/百万 token | 2 元/百万 token | 约 0.5 元/百万 token |

#### 代码模型
| 模型 | 输入价格 | 输出价格 |
|------|---------|---------|
| doubao-seed-code（16k输入） | 1.2 元/百万 token | 8 元/百万 token |

**注意：** 具体价格请以官网最新定价为准。

### 12.4 免费额度
- 新用户注册后会获得一定的免费额度
- 免费额度用于抵扣按 token 计费
- 在免费额度内，实时调用不收费

## 13. 速率限制

### 13.1 TPM（Tokens Per Minute）
- 每分钟可处理的 token 数量
- 不同套餐有不同的 TPM 限制
- 高级套餐可达 500 万 TPM

### 13.2 RPM（Requests Per Minute）
- 每分钟可发起的请求数
- 根据 TPM 和平均请求大小计算
- 例：500 万 TPM ≈ 500-1250 RPM（假设每请求 4k-10k tokens）

### 13.3 TPM 保障包
- 火山方舟提供 TPM 保障包服务
- 保障在高并发场景下的稳定性
- 适合有大规模调用需求的用户

## 14. 错误处理

### 14.1 常见错误码
| 错误码 | 描述 | 解决方案 |
|--------|------|---------|
| 401 | 无效的 API Key | 检查 API Key 是否正确 |
| 429 | 超出速率限制 | 降低请求频率或升级套餐 |
| 500 | 服务器错误 | 稍后重试 |
| 400 | 请求参数错误 | 检查请求参数格式 |

### 14.2 重试策略
```python
import time
from volcenginesdkarkruntime import Ark
from volcenginesdkarkruntime.exceptions import ArkAPIError

client = Ark(api_key="your-api-key")

def chat_with_retry(messages, max_retries=3):
    for attempt in range(max_retries):
        try:
            return client.chat.completions.create(
                model="ep-xxxxxxxxxx-xxxxx",
                messages=messages
            )
        except ArkAPIError as e:
            if e.status_code == 429:
                if attempt < max_retries - 1:
                    wait_time = 2 ** attempt
                    print(f"速率限制，等待 {wait_time} 秒...")
                    time.sleep(wait_time)
                else:
                    raise
            else:
                raise
```

## 15. 最佳实践

### 15.1 提示词优化
- 使用清晰、具体的指令
- 提供示例以引导模型
- 适当使用 system message 设定角色
- 合理控制上下文长度

### 15.2 性能优化
- 利用上下文缓存降低成本
- 根据任务复杂度选择合适的模型
- 使用 max_tokens 限制输出长度
- 流式响应提升用户体验

### 15.3 成本控制
- 简单任务使用 lite 系列
- 复杂任务使用 pro 系列
- 充分利用缓存机制
- 监控 token 使用情况

### 15.4 安全建议
- 不要在客户端暴露 API Key
- 使用环境变量存储密钥
- 实现请求频率限制
- 验证和过滤用户输入

## 16. 与 OpenAI API 的差异

### 16.1 主要相似点
| 特性 | OpenAI | 火山方舟 |
|------|--------|---------|
| 基本接口结构 | chat.completions.create | 相同 |
| 消息格式 | messages 数组 | 相同 |
| 流式响应 | 支持 SSE | 相同 |
| 函数调用 | tools 参数 | 相同 |
| SDK 兼容性 | OpenAI SDK | 兼容 |

### 16.2 主要差异点
| 特性 | OpenAI | 火山方舟 |
|------|--------|---------|
| Base URL | api.openai.com | ark.cn-beijing.volces.com |
| 模型标识 | gpt-4o | Endpoint ID (ep-xxx) |
| 内置工具 | 部分支持 | 丰富的内置工具 |
| Responses API | 有 | 有（支持更多工具） |
| 国内访问 | 需要代理 | 直接访问 |

### 16.3 迁移建议
1. 替换 base_url 和 api_key
2. 将 OpenAI 模型名替换为火山方舟 Endpoint ID
3. 其他代码基本无需修改
4. 测试函数调用等高级功能

## 17. 常见问题 FAQ

### 17.1 如何获取 Endpoint ID？
1. 登录火山引擎控制台
2. 进入火山方舟 -> 模型推理
3. 创建推理接入点（Endpoint）
4. 创建成功后即可看到 Endpoint ID

### 17.2 是否支持自定义模型？
支持。可以通过模型精调功能训练自定义模型，训练完成后创建 Endpoint 即可使用。

### 17.3 如何提高响应速度？
- 使用流式响应
- 选择较小的模型（如 lite 系列）
- 减少上下文长度
- 利用缓存机制

### 17.4 支持哪些编程语言？
- 官方 SDK：Python、Java、Go
- OpenAI 兼容：所有支持 OpenAI SDK 的语言
- HTTP API：所有能发起 HTTP 请求的语言

### 17.5 如何监控使用情况？
在火山引擎控制台的"用量统计"页面可以查看：
- Token 使用量
- 请求次数
- 费用消耗
- 错误统计

## 信息来源

- [火山方舟大模型服务平台官方文档](https://www.volcengine.com/docs/82379)
- [对话(Chat) API](https://www.volcengine.com/docs/82379/1494384)
- [兼容 OpenAI SDK](https://www.volcengine.com/docs/82379/1330626)
- [迁移至 Responses API](https://www.volcengine.com/docs/82379/1585128)
- [函数调用 Function Calling](https://www.volcengine.com/docs/82379/1262342)
- [工具调用](https://www.volcengine.com/docs/82379/1958524)
- [图片理解](https://www.volcengine.com/docs/82379/1362931)
- [模型列表](https://www.volcengine.com/docs/82379/1330310)
- [SDK 常见使用示例](https://www.volcengine.com/docs/82379/1544136)
- [模型服务计费说明](https://www.volcengine.com/docs/82379/1544681)
- [错误码](https://www.volcengine.com/docs/82379/1099476)
- [豆包大模型产品页](https://www.volcengine.com/product/doubao)
