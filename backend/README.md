# Zapi-Go

OpenAI API 网关，Go 重写版。高性能、解耦架构、倒排索引渠道池。

## 架构

```
Handler → Core → Model 三层架构

cmd/server/main.go          # 入口：路由注册 + 启动 + 优雅关闭
internal/
├── config/config.go         # YAML配置加载 (env变量覆盖)
├── core/
│   ├── auth.go              # JWT(HMAC校验+常量时间比较)/bcrypt/登录限速/密码验证
│   ├── cache.go             # Token/User/Group 内存缓存 (RWMutex+TTL)
│   ├── cache_invalidate.go  # 缓存失效集中管理
│   ├── captcha.go           # 验证码生成/校验 (纯Go PNG)
│   ├── db.go                # InitDB/SeedDefaults
│   ├── errorlog.go          # 错误日志环形缓冲 + 文件持久化 + 按天切割
│   ├── heartbeat.go         # 轻量级心跳检测(GET /v1/models) + 故障/恢复通知 + AutoDisable控制
│   ├── helper.go            # 时区/安全整数/日期解析/模型映射规范化
│   ├── logwriter.go         # 批量日志写入 (channel 65536)
│   ├── quota.go             # 异步配额扣减 (channel 65536)
│   ├── tokencount.go        # tiktoken-go 精确token计数
│   └── routing/
│       ├── engine.go        # 模型映射/URL构建
│       ├── policy.go        # 路由策略枚举
│       ├── health.go        # 熔断器（三态：Closed/Open/Half-Open）+ 渠道健康追踪
│       ├── upstream.go      # 上游组索引 + 别名映射 + GetMaxFailsForChannel
│       └── pool.go          # 倒排索引渠道池 (O(1)查找+预排序+IsAvailable熔断过滤)
├── handler/
│   ├── auth.go              # 登录/注册/验证码/改密码/获取用户信息
│   ├── channel.go           # 渠道CRUD+测试 (含AutoDisable检查+测试成功恢复启用)
│   ├── export.go            # CSV/XLSX导出
│   ├── group.go             # 分组CRUD + rate_mode/RPM/TPM/模型策略
│   ├── helpers.go           # 版号/我的面板/我的用量
│   ├── log.go               # 日志查询
│   ├── notification.go      # 通知CRUD+批量发送+广播
│   ├── proxy.go             # OpenAI兼容代理转发 (函数化架构: 认证→限流→路由→转发→计费, 超时不算故障)
│   ├── settings.go          # 系统设置 (超管专属, 含心跳设置+AutoDisable开关恢复)
│   ├── stats.go             # 统计/用量/仪表盘
│   ├── token.go             # 令牌CRUD+充值 (含分页)
│   └── user.go              # 用户CRUD+充值/扣除 + rate_mode/RPM/TPM/模型策略 (含分页)
├── middleware/middleware.go  # CORS/认证/权限/分页/SSE错误格式
├── migrate/migrate.go       # 版本化数据库迁移
├── model/model.go           # 数据模型+全局DB
└── ratelimit/
    ├── ratelimit.go         # API限流 (64分片+FNV32)
    └── resolve.go           # 统一速率限制解析 (唯一真相源)
static/                      # 前端SPA (Vue 3 + Vite + Naive UI)
```

## 快速开始

```bash
# 编译
go build -o zapi-go ./cmd/server/

# 运行 (默认端口 65000)
./zapi-go

# 自定义配置
cp config.yaml myconfig.yaml
# 编辑 myconfig.yaml
```

## 配置 (config.yaml)

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `server.port` | 65000 | 服务端口 |
| `server.timezone_offset` | 8 | 时区偏移 (DB存UTC，显示+偏移) |
| `database.url` | - | PostgreSQL连接串 (自动降级SQLite) |
| `database.pool_size` | 20 | 连接池大小 |
| `security.admin_token` | - | 超级管理员API令牌 |
| `security.jwt_expire_hours` | 1 | JWT过期时间 |
| `proxy.timeout` | 120 | 上游超时(秒) |
| `proxy.retry_count` | 1 | 重试次数 |
| `proxy.max_fails` | 5 | 熔断阈值（连续失败N次触发熔断，0=禁用） |
| `proxy.fail_timeout` | 30 | 熔断恢复时间(秒) |
| `rate_limit.rpm` | 60 | 每API Key每分钟请求数 |
| `rate_limit.ip_rpm` | 120 | 每IP每分钟请求数 |
| `log.error_max_entries` | 10000 | 错误日志内存缓冲条数 |
| `log.error_max_days` | 30 | 错误日志文件保留天数 |
| `log.retention_days` | 90 | 请求日志数据库保留天数 |
| `heartbeat.enabled` | true | 心跳检测开关 |
| `heartbeat.interval` | 60 | 检测间隔(秒) |
| `heartbeat.timeout` | 10 | 检测超时(秒) |

## 轻量级心跳 (v4.3.0+)

心跳检测使用 `GET /v1/models` 而非 `POST /v1/chat/completions`，只测服务器连通性，不算力：

| 响应 | 判定 |
|------|------|
| 连接失败（DNS/连接拒绝） | 真正故障，FailCount+1，同步到熔断器，可能触发自动禁用 |
| 5xx | 服务器不健康，FailCount+1，同步到熔断器 |
| 其他（200/401/403/404/429等） | 服务器存活，FailCount清零，熔断器恢复 |

GPU满负荷时 /v1/models 秒回，不会被误判为故障。

### 心跳-熔断联动 (v4.4.0)

心跳检测到故障后调用 `SyncFromHeartbeat` 同步到熔断器：

| 熔断状态 | 条件 | 行为 |
|----------|------|------|
| Closed→Open | FailCount >= max_fails | 渠道被`IsAvailable()`过滤，路由时自动跳过 |
| Open→HalfOpen | 等待fail_timeout（默认30秒） | 放行一次探测请求 |
| HalfOpen→Closed | 探测成功 | FailCount清零，恢复正常路由 |
| HalfOpen→Open | 探测失败 | 回到熔断状态 |

- `SyncFromHeartbeat`直接设置FailCount（不+1），避免心跳和代理双重计数
- auto_ban=OFF时熔断仍生效（仅控制永久禁用，不控制临时跳过）
- max_fails/fail_timeout取渠道所属上游组中最严格的值，未配置时默认(5, 30)

## 代理超时处理 (v4.5.0)

代理请求超时（context deadline）的处理：

| 错误类型 | 行为 |
|----------|------|
| 超时 | 加FailCount，触发熔断，但不自动禁用渠道（慢≠坏） |
| 连接拒绝/DNS失败 | 加FailCount，触发熔断，AutoBan时自动禁用渠道 |
| 上游5xx | 加FailCount，触发熔断，AutoBan时自动禁用渠道 |
| 客户端断开(context canceled) | 不计FailCount，不触发熔断，返回499 |

GPU满负荷时：代理超时会触发熔断暂时摘除慢渠道，但不会永久禁用（AutoBan不触发）。熔断超时后自动半开探测恢复。

## 渠道故障自动禁用 (v4.0.x)

### 双重控制

渠道是否被自动禁用由两个开关共同决定：

| 条件 | 说明 |
|------|------|
| `heartbeat.auto_disable = true` | 全局开关，在系统设置-心跳检测中配置 |
| 渠道 `auto_ban = true` | 渠道级开关，编辑渠道时配置 |

两个条件**同时为 true** 时，渠道连续失败5次才会被自动禁用。

### 仅告警模式

将系统设置中的「自动禁用」改为「仅告警」时：
- 渠道连续失败不会自动禁用，仍正常计数 fail_count
- 失败次数 >= 2 时仍会发送告警通知
- 之前因自动禁用而关闭的渠道会**自动恢复**启用

### 恢复机制

| 触发方式 | 行为 |
|----------|------|
| 心跳检测成功 | 自动重新启用渠道，fail_count清零 |
| 手动测试渠道成功 | 自动重新启用渠道，fail_count清零 |
| 关闭AutoDisable开关 | 批量恢复所有因自动禁用关闭的渠道 |

## 速率限制系统 (v3.9.x)

### rate_mode 互斥模式

| 级别 | 模式 | 说明 |
|------|------|------|
| 分组 | `global` | 全局RPM/TPM，所有模型共用 |
| 分组 | `per_model` | 每模型独立RPM/TPM (JSON配置，通配符`*`为默认) |
| 用户 | `inherit` | 继承分组设置 (默认) |
| 用户 | `global` | 用户级全局RPM/TPM，覆盖分组 |
| 用户 | `per_model` | 用户级每模型独立限制，覆盖分组 |

### 值含义

| 值 | 含义 |
|----|------|
| `0` | 无额度/禁止 |
| `-1` | 不限制 |
| `正数` | 实际限制值 |

### 解析优先级

用户设置始终覆盖分组。`ResolveRateLimits()` 是唯一真相源，proxy和仪表盘共用。

## 列表分页 (v4.0.x)

用户列表、令牌列表、渠道列表支持可选分页参数：

| 参数 | 说明 |
|------|------|
| `limit` | 每页条数，最大500。不传则返回全部（向后兼容） |
| `offset` | 偏移量，默认0 |

请求示例：
```
GET /api/users?limit=20&offset=0    → {"items": [...], "total": N}
GET /api/users                       → [...]（无分页，向后兼容）
```

## 错误日志系统 (v3.9.x)

### 覆盖场景

| 场景 | 日志示例 |
|------|----------|
| 令牌额度不足 | `令牌额度不足: 令牌[test] ID:6 额度:0/0` |
| 用户已禁用 | `用户已禁用: testuser ID:4` |
| 模型被限制 | `模型被限: 用户testuser 模型:gpt-4` |
| 速率限制拦截 | `速率限制: 请求频率超限 用户:testuser IP:1.2.3.4` |
| 用户额度不足 | `用户额度不足: testuser 已用:5000/额度:5000` |
| 无可用渠道 | `无可用渠道: 模型:gpt-4 用户:testuser` |
| 代理失败(上游超时) | `代理失败: 用户testuser 模型:gpt-4 渠道[xxx] 错误:上游服务超时` |
| 上游HTTP错误 | `上游HTTP错误: 用户testuser 模型:gpt-4 渠道[xxx] HTTP:500` |
| 心跳故障/恢复 | `心跳故障: 渠道[xxx] ID:5 失败次数:3` |
| 渠道测试失败 | `渠道测试失败: [xxx] ID:6 模型:gpt-4 错误:...` |

### 存储

- 内存环形缓冲 (max_entries，FIFO)
- 文件持久化: `logs/zapi-YYYY-MM-DD.log`，按天切割
- 自动清理: 超过 `error_max_days` 天的日志文件自动删除

## 代理流程

```
请求 → API Key验证 → Token缓存查询 → 配额检查 → User缓存查询
     → 速率限制解析(含TPM记账) → 模型权限 → 分组过滤 → 渠道池O(1)选择 → 模型映射
     → 上游转发 → token计数(tiktoken) → 配额扣减(异步) → 日志记录(异步)
     → 失败重试+故障转移 → 自动禁用(可选) → 错误日志记录
```

### 代理处理器架构 (v4.0.x)

`HandleProxy` 由以下函数协作完成：

| 函数 | 职责 |
|------|------|
| `authenticateRequest` | API Key验证 + Token/User缓存查询 |
| `checkQuotaAndRateLimit` | 令牌额度 + 用户状态 + 限流 + 模型权限 |
| `buildUpstreamRequest` | 模型映射 + stream_options注入 + 请求构建 |
| `handleChannelFail` | fail_count递增 + AutoDisable检查 + 路由池更新（仅真正故障调用） |
| `handleChannelTimeout` | 超时处理：FailCount+1，触发熔断但不自动禁用渠道（慢≠坏） |
| `handleChannelSuccess` | fail_count清零 + 路由池更新 |
| `sanitizeUpstreamError` | 清洗上游错误（隐藏Python traceback等） |
| `processStreamResponse` | SSE流式转发 + token计数 + TPM记账 + 配额扣减 |
| `processNonStreamResponse` | 非流式转发 + TTS/Chat/Images分类计费 |
| `proxyError` | 统一流式/非流式错误响应格式 |

## 安全增强 (v4.0.x)

| 增强 | 说明 |
|------|------|
| JWT签名算法校验 | keyFunc验证HMAC签名方法，拒绝none算法攻击 |
| Admin Token常量时间比较 | 使用crypto/subtle.ConstantTimeCompare防止时序攻击 |
| 配置加载错误检查 | config.yaml缺失或格式错误时拒绝启动 |

## 删除用户说明

删除用户时：
- ✅ 删除用户账号及其所有 API Key
- ⚠️ **保留**历史调用日志、统计数据（按 user_id 关联）
- 已删除用户的日志中用户名显示为 `-` 或 `user:<id>`
- 保留的日志会在 `log.retention_days`（默认 90 天）后自动清除

## 默认管理员

- 用户名: `admin`
- 密码: `Admin@123`
- 超管ID: 1

## Linux 部署

```bash
# 编译 Linux 版本
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dist-linux/zapi-go ./cmd/server

# 上传 dist-linux/ 目录到服务器后
cd /path/to/zapi-go
bash start.sh    # 启动
bash stop.sh     # 停止
tail -f zapi.log # 查看日志
```

### start.sh 功能

- 自动cd到脚本所在目录
- 检查zapi.pid防止重复启动
- nohup后台运行，PID写入zapi.pid
- 创建logs目录

### stop.sh 功能

- 优先优雅退出（SIGTERM + 5秒等待）
- 超时后SIGKILL强杀
- 无pid文件时按进程名查找
- 自动清理过期pid文件

## 版本

v4.5.9

## 变更日志

### v4.5.9 (2026-05-06)

**前端**
- 修复用户管理页面"最大Token数"翻译不一致：改为"最大令牌数"，与"令牌数上限"、"编辑令牌"等统一用语
- 此处Token指API Key数量限制（用户最多可创建几个令牌），非LLM的token用量

### v4.5.8 (2026-05-04)

**前端**
- 全新动态背景动画（ParticleNetwork.vue）：三层景深粒子系统 + 路由节点数据包转发动画 + 极光流动带 + 声纳脉冲 + 连线流光效果
- 有机噪声运动：粒子使用正弦组合噪声实现自然漂浮，非固定位置动画
- 视差滚动：背景层随页面滚动以不同速度偏移（极光0.1x / 远0.15x / 中0.25x / 近0.35x）
- 网格均匀节点分布 + 中心微力防聚堆，解决粒子集中左侧问题
- 浅绿色系配色（green/emerald/teal），替换原青蓝紫配色
- 内容区半透明背景 + backdrop-filter:blur，粒子动画透出但不遮挡UI
- Sidebar/Header半透明适配，暗色/亮色双模式完整支持
- NaiveUI全局主题边框增强：暗色borderColor rgba(255,255,255,0.12)，亮色rgba(0,0,0,0.14)
- Card/DataTable独立边框色、暗色卡片背景微调（#1e1f27）
- StatsCard暗色边框增强 + box-shadow ring，提升模块边缘清晰度
- 仪表盘模型访问表格和限额卡片增加阴影层次感

### v4.5.5 (2026-05-04)

**前端**
- 复制功能HTTP环境兼容：所有页面的复制操作增加 `document.execCommand('copy')` fallback，非HTTPS环境不再报"复制失败"
- 复制/查看按钮优化：`size="tiny"` → `size="small"`，emoji图标(📋🙈👁) → SVG矢量图标，圆形图标按钮更美观
- 使用指南API Key安全遮挡：代码块中的API Key显示为 `sk-xxxx••••••••xxxx`，点击复制时输出完整Key
- 涉及页面：我的令牌、管理员令牌管理、用户管理、使用指南

### v4.5.0 (2026-05-04)

**新功能**
- 渠道熔断机制（Circuit Breaker）：连续失败N次自动摘除渠道，fail_timeout秒后半开探测，成功则恢复
- 三态熔断器：Closed（正常）→ Open（熔断，跳过渠道）→ Half-Open（半开，放1个试探请求）
- 系统设置新增"熔断阈值"和"熔断恢复时间"配置项（proxy_max_fails / proxy_fail_timeout）
- selectFromPool加入IsAvailable检查：不走上游组时熔断也生效（之前只有上游组路径有检查）
- 超时计入熔断：代理超时触发FailCount递增+熔断判定（之前超时不算故障导致慢渠道永远摘不掉）
- context canceled不算故障：客户端主动断开返回499，不计FailCount

**改进**
- AutoBan阈值与熔断阈值统一：AutoBan不再硬编码5次，改用MaxFails配置值
- 设置API新增proxy_max_fails/proxy_fail_timeout后端验证：max_fails>=0, fail_timeout>=1
- 上游组前端fail_timeout最小值从0改为1（0会导致熔断瞬间恢复无意义）
- 删除未使用的RecordTimeout死代码
- ResponseHeaderTimeout改为0（无限），完全靠proxy.timeout控制超时

**前端**
- 系统设置页面代理区域新增"熔断阈值"和"熔断恢复时间"输入框
- 中英文翻译新增proxyMaxFails/proxyFailTimeout

### v4.4.0 (2026-05-03)

**新功能**
- 心跳熔断联动：心跳检测发现故障后同步到熔断器（`SyncFromHeartbeat`），故障渠道被`IsAvailable()`过滤，路由时自动跳过
- 渠道故障可视化：渠道卡片显示⚠️故障标签+橙色脉冲边框+失败次数；上游组显示💓心跳延迟列和不健康状态
- `GetMaxFailsForChannel`：渠道属于多个上游组时取最严格的max_fails/fail_timeout；未配置或不在任何组中时默认(5, 30)

**改进**
- auto_ban=OFF时熔断仍生效：auto_ban仅控制永久禁用，不控制临时熔断跳过（30秒半开探测自动恢复）
- max_fails=0不再导致永不熔断：未配置时默认5次触发熔断
- `handleChannelFail`改用`GetMaxFailsForChannel`获取最严格参数，修复之前永远取`UpstreamGroupIDs[0]`的bug
- 心跳故障同步不重复计数：`SyncFromHeartbeat`直接设置FailCount（不+1），避免心跳和代理双重计数
- 上游组状态判定增强：`lb_status.go`检查`FailCount > 0`即为不健康，新增`response_time`和`heart_fail_count`字段

**前端**
- 暗色主题全量修复：20+组件硬编码色值→CSS变量(`var(--n-xxx)`)，删除fallback泄漏，Login/Register/Guide等页面暗色适配
- StatsCard/UsageChart暗色适配
- SideNav暗色适配

### v4.3.0 (2026-05-02)

**新功能**
- 轻量级心跳：从 `POST /v1/chat/completions` 改为 `GET /v1/models`，GPU满负荷时不会误判故障
- 新增 `handleChannelTimeout`：超时专用处理函数（v4.5.0起超时计入熔断故障）

**改进**
- 心跳判定扩展：非5xx响应均视为服务器存活（200/401/403/404/429等）

**前端**
- Vue 3 + Vite + Naive UI 重写（从单文件CDN迁移到组件化SPA）
- 渠道页心跳延迟💓显示修复

### v4.0.2 (2026-04-30)

**Bug修复**
- 修复Dashboard图表、用量分析、导出报表的日期范围差8小时（前端用toISOString生成UTC日期，改为localDate生成本地日期）

### v4.0.1 (2026-04-29)

**安全修复**
- JWT签名算法校验：keyFunc验证HMAC方法，拒绝none算法攻击
- Admin Token常量时间比较：防止时序攻击逐字节猜测
- 配置加载错误检查：config.yaml缺失或格式错误时拒绝启动

**Bug修复**
- 修复日志文件只能生成两天的问题（errorlog.go 4处bug：关闭后未置nil、失败未重置fileDay、无Sync()、错误静默吞掉）
- 修复渠道测试(HandleTestChannel)遗漏AutoDisable全局检查，仅告警模式下仍会禁用渠道
- 修复渠道测试成功后编辑保存渠道回到禁用状态（测试成功时未设Enabled=true到数据库）
- 修复流式响应缺少TPM速率限制记账（AccountTokens调用缺失）
- 删除heartbeat.go中无效的`uid := uid; _ = uid`代码

**新功能**
- 渠道故障自动禁用全局开关：心跳检测设置UI，可选「自动禁用」或「仅告警」
- 关闭AutoDisable时自动恢复之前被禁用的渠道
- 列表分页：用户/令牌/渠道列表支持limit/offset参数（向后兼容）
- Linux部署包：start.sh（pid管理+防重复启动）+ stop.sh（优雅退出+强杀+进程名查找）

**重构**
- proxy.go 275行巨型函数拆分为9个独立函数，提升可维护性
- 统一流式/非流式错误响应格式（proxyError函数，消除8处重复代码）
