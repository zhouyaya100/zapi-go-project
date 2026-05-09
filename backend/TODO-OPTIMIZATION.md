# Zapi-Go 待优化清单

> 审查时间：2026-05-02 (v4.3.0)，最后更新：v4.5.8，已完成的标记 ✅，其余待逐步修改

---

## 已完成

- ✅ JWT签名算法未校验 → auth.go keyFunc加了HMAC校验
- ✅ Admin Token时序攻击 → 改用crypto/subtle.ConstantTimeCompare
- ✅ proxy.go 275行巨型函数 → 拆成9个函数
- ✅ 列表无分页 → user/token/channel加limit/offset参数
- ✅ HandleTestChannel漏了AutoDisable全局检查 → 已加
- ✅ HandleTestChannel测试成功没设Enabled=true → 已加
- ✅ 流式响应缺少TPM速率限制记账 → processStreamResponse加了AccountTokens
- ✅ main.go忽略config.Load错误 → 加了log.Fatal
- ✅ heartbeat.go无效代码uid:=uid → 已删除
- ✅ 日志文件只能生成两天 → errorlog.go重写修复4处bug
- ✅ 版本号硬编码 → 统一更新到v4.0.1（仍4处硬编码，待ldflags方案）
- ✅ 前端日期参数用toISOString生成UTC日期差8小时 → 新增localDate()函数生成本地日期
- ✅ 心跳改轻量级：POST /v1/chat/completions → GET /v1/models，GPU满负荷不误判
- ✅ 代理超时不算故障：超时不加FailCount、不触发熔断、不自动禁用
- ✅ 心跳JSON注入风险消除：不再用json.Marshal构建请求体，改为GET请求无body
- ✅ HTTP Client已复用：proxy.go使用sharedClient全局实例，不再每次请求创建

---

## 安全性

### [严重] 默认凭据硬编码 — config.yaml + internal/core/db.go
- config.yaml里数据库URL含明文密码、admin_token、secret_key都是默认值
- db.go第44行 `HashPassword("Admin@123")` 默认管理员密码硬编码
- 建议：启动时检测默认密钥拒绝运行，secret_key改为自动生成并持久化到数据库

### [高] CORS允许任意源 — internal/middleware/middleware.go:16
- `Access-Control-Allow-Origin: *` 配合 `Credentials: true` 违反浏览器安全策略
- 建议：从配置读取允许的Origin列表，动态设置

### [高] 设置API泄露数据库URL — internal/handler/settings.go:105
- GET /api/settings 返回含明文密码的database_url
- 建议：返回时脱敏，如 `postgresql://***:***@host:5432/dbname`

### [中] 登录暴力破解防护不足 — internal/core/auth.go:64-78
- 失败5次锁定5分钟，但存储在内存map中重启即清空，且无渐进延迟
- 建议：失败次数持久化到DB，连续失败后指数退避

### [中] 验证码复用 — internal/handler/auth.go:90-91
- 登录和注册共用同一个captchaId/captchaCode
- 建议：登录和注册各自独立请求验证码

### [中] XSS风险 — static/app.js:134
- 错误消息等用户输入数据直接展示，需确认所有位置都未用v-html
- 建议：全局搜索确保无v-html渲染用户输入，添加CSP header

### [低] ~~模型名JSON注入~~ — internal/handler/channel.go:138 ~~internal/core/heartbeat.go:62~~
- ~~fmt.Sprintf直接拼接模型名到JSON body，模型名含`"`可注入~~
- ✅ 心跳已改为GET /v1/models无请求体，不再有注入风险
- channel.go手动测试仍有此风险，建议改用json.Marshal

---

## 性能

### [高] 导出无限制 — internal/handler/export.go:57
- `Limit(100000).Find(&logs)` 最多加载10万条到内存，可能OOM
- 建议：流式写入（逐批查询+写入），或限制更小批次

### [高] 统计查询全表扫描 — internal/handler/stats.go:14-25
- 5个独立COUNT/SUM查询，全表扫描
- 建议：合并查询，对高频统计使用缓存

### [中] 每次请求解析model_rate_limits JSON — internal/ratelimit/resolve.go:58-84
- 每次代理请求都json.Unmarshal用户/分组的model_rate_limits
- 建议：缓存解析结果，或在CachedLookupUser/CachedLookupGroup时预解析

### [中] quota扣减批量刷新清空全部token缓存 — internal/core/quota.go:38-39
- InvalidateAllTokenCache()每5秒清空所有token缓存，大量token时缓存几乎无效
- 建议：维护tokenID→apiKey反向索引，精准失效单个token

### [中] ~~代理处理器每次创建新HTTP Client~~ — ~~internal/handler/proxy.go:29-36~~
- ~~getHTTPClient()每次请求创建http.Client，可复用~~
- ✅ 已改为sharedClient全局实例（v4.2.x），仅配置变更时需重建

### [低] 本地缓存驱逐策略 — internal/core/cache.go:66-74
- 缓存满时随机删除10%，无法保证淘汰最不活跃的条目
- 建议：实现LRU或使用成熟缓存库

### [低] BodyLimit仅检查Content-Length — internal/middleware/middleware.go:26-29
- 只检查ContentLength header，攻击者可伪造小Content-Length实际发送大body
- 建议：使用io.LimitReader或gin-contrib/size中间件

---

## 代码架构

### [高] 重复的查询过滤逻辑 — handler/log.go, handler/helpers.go, handler/export.go
- 日期过滤、模型过滤、用户过滤等代码重复
- 建议：提取buildLogQuery(c *gin.Context) *gorm.DB公共函数

### [中] 分页API不一致 — channel/user/token用limit/offset, log/stats用page/pageSize
- 不同列表接口使用不同分页参数格式
- 建议：统一为limit/offset或page/pageSize

### [中] 全局变量DB暴露 — internal/model/model.go:113
- var DB *gorm.DB 任何包可直接操作数据库，无抽象层
- 建议：通过接口/repository模式封装数据访问

### [中] 配置写回文件覆盖注释 — internal/handler/settings.go:226-228
- 修改设置时整个Config写回config.yaml，手动编辑的注释和格式丢失
- 且写文件无原子性，可能写到一半崩溃导致配置损坏
- 建议：使用写临时文件+原子重命名，或仅更新变更的key值

### [中] 缺少事务 — handler/user.go, handler/group.go
- 删用户先删token再删用户，删分组先清用户再删分组，均无事务包裹
- 中途失败导致数据不一致
- 建议：用model.DB.Transaction包裹

### [中] model_rate_limits无验证 — handler/user.go:99, handler/group.go:92
- 直接接受前端传入的字符串存入数据库，无JSON格式验证
- 建议：写入前验证JSON格式和字段类型

### [中] heartbeat_enabled运行时修改不启停goroutine — settings.go
- 修改心跳开关后需重启才生效
- 建议：检查变更值，false→true时启动goroutine

### [低] 不一致的错误处理 — 多处handler
- 有些地方检查ShouldBindJSON错误，有些忽略
- 建议：统一错误处理方式

### [低] 重复的SplitComma函数 — internal/core/helper.go:23 和 internal/core/routing/pool.go:263
- 两个包各实现了相同逻辑
- 建议：统一到core包

---

## 运维

### [高] 优雅关闭不完整 — cmd/server/main.go:189-196
- 收到信号后直接os.Exit(0)，不会等待Gin服务器完成在途请求
- 建议：使用http.Server + srv.Shutdown(ctx)替代r.Run()

### [高] 无结构化日志 — 全项目
- 所有日志使用fmt.Printf/log.Println + core.ErrLog.Error，无级别无请求ID
- 建议：引入slog（Go 1.21+）或zap/zerolog，统一结构化日志

### [中] 无健康检查端点 — cmd/server/main.go
- 缺/health或/ready端点，K8s/Docker无法探活
- 建议：添加GET /health返回数据库连通性、版本等

### [中] 无Prometheus指标
- 无请求延迟、错误率、token消耗等metrics导出
- 建议：添加/metrics端点

### [中] 配置热更新不完整 — internal/handler/settings.go
- 修改代理超时、连接池大小等配置后，已创建的sharedTransport和http.Client不会更新
- 建议：配置变更时重建Transport/Client，或文档说明哪些配置需重启生效

### [低] 版本号硬编码多处 — cmd/server/main.go, handler/helpers.go
- ✅ 前端已从/api/version动态获取版本号（v4.5.x Vue3重写）
- Go后端仍有2处硬编码（main.go启动日志、helpers.go版本API），可考虑ldflags方案

---

## 前端

### [高] ~~app.js单文件过大~~ — ~~static/app.js (491行, 56KB+)~~
- ~~整个前端逻辑在一个JS文件中，无模块化~~
- ✅ 已迁移到Vue 3 + Vite + Naive UI组件化SPA（/tmp/zapi-go-vue3/）

### [高] ~~index.html单文件过大~~ — ~~static/index.html (1810行, 119KB)~~
- ~~所有页面模板堆叠在一个HTML文件中~~
- ✅ 已迁移到Vue 3组件化架构

### [中] 前端未处理JWT过期 — ✅ 已修复
- ✅ 401时自动清除token并跳转登录页（v4.5.x Vue3重写时已实现）

### [中] API Key明文返回前端 — internal/handler/token.go:41
- HandleListMyTokens返回完整key，若前端被XSS所有token泄露
- 建议：只在创建时返回完整key，列表接口返回脱敏key

### [中] XSS潜在风险 — ✅ 已修复（Vue3自动转义）
- ✅ Vue 3模板默认HTML转义，无v-html渲染用户输入
- v4.5.8暗色主题全量修复，所有硬编码色值改为CSS变量

### [低] ~~前端未压缩~~ — ~~static/index.html, static/app.js~~
- ~~HTML和JS未压缩，app.js 56KB可minify到约30KB~~
- ✅ Vite构建自动minify + tree-shake + code-split

### [低] localStorage存储JWT — static/app.js:100
- 容易被XSS窃取
- 建议：改用HttpOnly cookie
- 注：v4.5.x已迁移到Vue3，仍用localStorage存储JWT

### v4.5.8 新增前端优化
- ✅ 全新动态背景动画（ParticleNetwork.vue）：三层景深粒子+路由节点数据包+极光带+声纳脉冲+连线流光
- ✅ 视差滚动效果：背景层随滚动不同速度偏移
- ✅ 浅绿色系配色替换青蓝紫
- ✅ 半透明UI + backdrop-filter:blur，动画透出但不遮挡
- ✅ NaiveUI全局主题边框增强（暗色0.12/亮色0.14）
- ✅ StatsCard/DataTable shadow增强，模块边缘更清晰

---

## 其他

### [中] SQLite兼容性问题 — internal/core/helper.go:48-49, handler/log.go:23
- TzDateExpr()使用PostgreSQL语法INTERVAL，ILIKE也是PG特有，用SQLite会报错
- 建议：根据数据库类型选择SQL方言，或文档明确仅支持PostgreSQL

### [中] 心跳检测串行执行 — internal/core/heartbeat.go
- checkAllChannels串行检测所有渠道，渠道多时一轮检测可能超过心跳间隔
- 建议：使用goroutine并发检测，或限制并发数

### [低] 随机数源不安全 — internal/core/captcha.go:22-24, internal/core/routing/pool.go:143
- 使用math/rand而非crypto/rand生成验证码ID和路由权重选择
- 建议：captcha ID改用crypto/rand，路由选择可用math/rand/v2
