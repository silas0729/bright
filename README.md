# Brights

这是一个以英语高频单词学习为主、同时预留多学科扩展能力的 `Golang + React` 项目起步版。

当前这版已经帮你做好了两件最重要的事情：

1. 后端可以直接读取你目录里的词库文件并提供查询接口。
2. 前端已经有一个可运行的学习站首页，支持搜索、分类筛选和套餐展示。

## 先说结论

你这份 `brights_202605020108.csv` 文件虽然扩展名是 `.csv`，但实际内容是 Excel 工作簿格式。后端已经做了自动识别，启动时会优先把它当成 Excel 数据导入。

另外，这份词库里的 `classification` 字段目前是混合数据：

- 一部分是真正的场景分类，例如 `驾驶`
- 一部分是词库来源名，例如 `BBC较常用的英文单词 (13.5万单词量)`、`英语词频表COCA60000`

所以正式做产品时，建议你把分类拆成两套：

- `topic`：前台展示用的场景标签，例如 `驾驶`、`点餐`、`商务会议`
- `source`：后台维护用的来源标签，例如 `COCA60000`、`雅思高频`

当前代码已经把过长、像词库来源名的分类自动归并到 `综合词库`，避免前台分类页太乱。

## 推荐技术方案

### 1. 前端

- `React + TypeScript + Vite`
- 首页以英语高频词为主
- 核心页面建议：
  - 首页/品牌页
  - 单词列表页
  - 场景专题页
  - 会员/买断套餐页
  - 登录页
  - 后台管理页

### 2. 后端

- `Go` 标准库先起步，后续可切 `Gin` 或 `Chi`
- 当前已提供：
  - 词库查询接口
  - 分类统计接口
  - 套餐接口
  - 后台新增内容接口骨架
  - 本地词库导入接口

### 3. 数据层

首版演示先用内存存储，方便你快速跑通页面和接口。

正式上线建议换成：

- `MySQL 8`：主业务数据
- `Redis`：验证码、登录态、热点缓存
- `OSS/COS`：音频、图片、课程封面

### 4. 支付方案

建议分两类：

- 一次性买断：
  - PC 网站优先接 `微信支付 Native`
  - 微信内网页/小程序可接 `JSAPI`
- 月会员自动续费：
  - 首次开通先完成普通支付
  - 续费走微信支付 `委托代扣/周期扣费`

注意：月会员自动续费不是普通下单接口直接循环调用就行，它需要单独的产品能力申请、签约流程和扣费前通知机制。

## 推荐的产品分阶段

### Phase 1: 最小可行产品

- 英语单词首页
- 搜索、分类、分页
- 免费词和会员词标记
- 套餐展示
- 后台导入词库

### Phase 2: 可运营版本

- 用户注册/登录
- 学习记录
- 收藏、生词本
- 会员权限
- 微信支付一次性买断

### Phase 3: 商业化版本

- 月会员自动续费
- 后台课程/专题/学科管理
- 音频发音
- AI 例句、联想记忆
- 数据分析和转化漏斗

## 目录结构

```text
brights/
├─ api/   # Go 后端
├─ web/   # React 前端
└─ brights_202605020108.csv  # 实际是 Excel 格式的数据文件
```

## 已实现的接口

后端启动后默认监听 `http://localhost:8080`

- `GET /api/v1/health`
- `GET /api/v1/subjects`
- `GET /api/v1/stats`
- `GET /api/v1/classifications?subject=english`
- `GET /api/v1/words?subject=english&page=1&page_size=20&q=car`
- `GET /api/v1/plans`
- `POST /api/v1/admin/import/local`
- `POST /api/v1/admin/subjects`
- `POST /api/v1/admin/words`
- `POST /api/v1/admin/plans`
- `GET /mcp/info`
- `WS /mcp`

## MCP 接入

Brights 现在提供了一个基于 WebSocket 的 MCP 入口，地址是：

- `GET /mcp/info`
- `WS /mcp`

### 鉴权方式

MCP 连接需要学员登录态，并且该学员对指定学科有有效会员。

- 推荐方式：`Authorization: Bearer <learner_access_token>`
- 兼容方式：`ws://localhost:8080/mcp?subject=english&token=<learner_access_token>`

其中：

- `subject` 或 `subject_key` 必填
- token 可以来自 `POST /api/v1/auth/login` 或 `POST /api/v1/auth/register` 返回的 `access_token`

### 工具命名风格

对外暴露的工具名已经调整为更短、更接近 XiaozhiMCP 常见风格的 `snake_case` 名称：

- `list_subjects`
- `list_categories`
- `list_grades`
- `search_words`
- `list_classification_stats`
- `list_membership_plans`
- `get_catalog_stats`

为了兼容旧调用，下面这些名字仍然可以继续调用，但不会在 `tools/list` 中重复暴露：

- `brights_list_subjects`
- `brights_list_categories`
- `brights_list_grades`
- `brights_search_words`
- `brights_list_classification_stats`
- `brights_list_membership_plans`
- `brights_get_catalog_stats`
- `list_words`
- `list_plans`

### 返回结构

`tools/call` 成功时会返回：

```json
{
  "content": [
    {
      "type": "text",
      "text": "{\n  \"success\": true,\n  \"tool\": \"search_words\",\n  \"result\": { ... }\n}"
    }
  ],
  "structuredContent": {
    "success": true,
    "tool": "search_words",
    "result": {}
  }
}
```

业务执行失败时会返回 `isError: true`，并保持同样的 envelope：

```json
{
  "isError": true,
  "structuredContent": {
    "success": false,
    "tool": "search_words",
    "error": "active membership is required for subject english"
  }
}
```

### 本地示例客户端

仓库里已经带了一个可以直接跑的 PowerShell WebSocket 示例：

```powershell
cd D:\ProjectCode\brights
Copy-Item api\examples\mcp-client.config.example.json api\examples\mcp-client.config.json
```

把 `api/examples/mcp-client.config.json` 里的这几个字段改掉：

- `access_token`：填学员登录接口拿到的 token
- `subject_key`：填你要访问的学科，例如 `english`
- `tool_name`：默认是 `search_words`

然后运行：

```powershell
cd D:\ProjectCode\brights
powershell -ExecutionPolicy Bypass -File .\api\examples\mcp-client.ps1 -Config .\api\examples\mcp-client.config.json
```

如果你只想先看工具列表：

```powershell
powershell -ExecutionPolicy Bypass -File .\api\examples\mcp-client.ps1 -Config .\api\examples\mcp-client.config.json -ListToolsOnly
```

### 建议调用顺序

客户端连接后，建议按这个顺序走：

1. 发送 `initialize`
2. 发送 `notifications/initialized`
3. 发送 `tools/list`
4. 发送 `tools/call`

## 本地运行

### 1. 启动后端

```powershell
cd D:\ProjectCode\brights\api
go run ./cmd/server
```

如果你想手动指定数据文件：

```powershell
$env:BRIGHTS_DATA_FILE="D:\ProjectCode\brights\brights_202605020108.csv"
go run ./cmd/server
```

### 2. 启动前端

```powershell
cd D:\ProjectCode\brights\web
npm install
npm run dev
```

前端开发地址默认是：

- `http://localhost:5173`

## 下一步最值得做的 5 件事

1. 把 `classification` 正式拆成 `topic` 和 `source`
2. 接入 `MySQL` 并做真实后台登录
3. 给单词增加 `level / is_vip / example / audio_url`
4. 接微信支付 Native 和 JSAPI
5. 再做会员自动续费

## 未来推荐的数据表

- `subjects`
- `topics`
- `words`
- `word_examples`
- `users`
- `memberships`
- `payment_orders`
- `payment_callbacks`
- `admin_users`

## 关于微信支付的实施建议

网站场景通常这样选：

- PC 打开网站扫码购买：`Native`
- 微信内打开网页购买：`JSAPI`
- 月会员续费：`委托代扣/周期扣费`

等你准备继续往下做时，下一步最合适的是：

1. 先把后台登录和 MySQL 版本做出来
2. 然后把 `words / topics / plans / orders / memberships` 这些表建起来
3. 最后再接微信支付真实商户参数
