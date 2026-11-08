# 上门喂宠区域竞价平台（Go + 微信小程序）

本项目依据 `上门喂宠竞价平台_PRD_V1.0.md` 和 `pet-feeding-miniapp/dist/index.html#/owner` 实现，包含原生微信小程序与 Go 模块化单体 API。业务数据持久化到 SQLite，不再使用 JSON mock 文件、种子用户或演示任务。

## 已实现

**宠物主人端**

- 微信登录与 Bearer Token 会话，登录身份由后端统一确定。
- 主动设置头像与昵称；微信登录仅返回 OpenID，头像昵称通过微信头像选择器和昵称输入组件授权。
- 真实宠物档案创建；新账号默认无宠物、无任务、无订单。
- 发布任务：宠物、服务、日期/时段、竞价或一口价、预算、截止时间、小区、地图选点、详细门牌、安全确认与协议。
- 报价详情：学生报价、核验标识、选人锁定、待付款倒计时、支付、撤单和重新竞价。
- 订单详情：付款后才向中选学生开放门牌地址，支持签到、凭证、验收与评价。
- 邻里社区 Bar：首页展示动态总数与最新动态，可进入社区查看、发文字和图片动态。

**学生接单端**

- 任务大厅、保密报价、我的任务、上门签到、照片凭证、完工与收益中心。
- 学生开启定位后只展示 5 公里内的可报价任务，后端在提交报价时再次校验距离；成交前只返回距离，不返回门牌坐标。
- 学籍认证提交与审核状态展示；审核通过前可以浏览任务，但不能报价。
- 学生同样可以进入社区查看和发布动态。

**社区与上传**

- 动态列表支持游标分页和下拉刷新。
- 每条动态支持最多 500 字、9 张图片。
- 图片通过平台上传接口落盘，只允许在动态中引用平台 `/uploads/...` 资源。
- 支持 JPG、PNG、WebP、GIF，单张最大 8MB。

**运营管理平台**

- 浏览器入口 `/admin/`，提供运营概览、5km 派单对比、学籍审核、退款审核、用户管理、订单管理和审计日志。
- 派单对比使用真实待派任务与已认证学生位置，并排展示贪心算法和 KM（匈牙利算法）的匹配明细；当前为只读预演，不会自动成交。
- 支持停用/恢复平台账号；停用会立即清除该用户的全部会话并阻止后续登录。
- 学籍审核、退款审核和账号处置会写入后台审计日志。
- 当前使用 `ADMIN_TOKEN` 单管理员入口，完整生产边界见 [`docs/admin-operations.md`](docs/admin-operations.md)。

## 技术结构

```text
cmd/api/                  Go API 入口
internal/domain/          领域模型与接口视图
internal/store/           SQLite 持久化、事务、状态机与权限裁剪
internal/geo/             Haversine 距离计算与 5km 地理围栏常量
internal/dispatch/        贪心与 KM（匈牙利算法）批量匹配
internal/payment/         微信支付 v3 签名、回调验签、退款与补偿任务
internal/httpapi/         登录、业务路由、上传、CORS 与错误处理
admin/                    运营管理平台 Web 控制台
miniprogram/              原生微信小程序（WXML/WXSS/JS）
pet-feeding-miniapp/      原始 Vite 原型，保留用于视觉对照
```

后端金额全部使用整数分。选人、支付幂等、签到、验收等关键操作使用事务和条件更新。数据库默认位于 `data/zoo.db`，上传文件默认位于 `data/uploads/`。

## 本地启动

要求 Go 1.25+。

推荐把本机凭证放在项目根目录的 `.env.local`（已加入 `.gitignore`，权限应为 `600`），然后使用启动脚本：

```bash
./scripts/run-local.sh
```

脚本会读取 `.env.local`、校验 AppSecret 长度，再启动 Go API。可参考 `.env.example` 创建自己的配置，不要把真实密钥提交到仓库。

也可以直接通过环境变量启动开发模式：

```bash
DEV_AUTH=true ADMIN_TOKEN=dev-admin \
  go run ./cmd/api -addr :8080 -data data/zoo.db -uploads data/uploads
```

健康检查：

```bash
curl http://127.0.0.1:8080/v1/healthz
```

运营后台：

```text
http://127.0.0.1:8080/admin/
```

使用启动命令中配置的 `ADMIN_TOKEN` 登录。后台静态文件由 `ADMIN_DIR` 指定，默认是项目根目录的 `admin/`。

首次启动只写入三个可配置服务区，不会创建用户、宠物、任务、订单或社区动态。新账号的首页为空，需要先添加宠物档案再发单。

清空本地数据库并重新开始：

```bash
go run ./cmd/api -addr :8080 -data data/zoo.db -reset
```

### 开发登录

本地设置 `DEV_AUTH=true` 后，小程序会持久化一个设备标识并换取稳定开发账号：

```json
{
  "code": "wx.login 返回的 code",
  "deviceId": "本地持久化设备标识"
}
```

正式环境应配置微信小程序凭证并关闭 `DEV_AUTH`：

```bash
WECHAT_APP_ID=wx... \
WECHAT_APP_SECRET=... \
ADMIN_TOKEN=强随机值 \
go run ./cmd/api -addr :8080 -data data/zoo.db -uploads data/uploads
```

`WECHAT_APP_ID` 与 `WECHAT_APP_SECRET` 在微信公众平台的“开发管理 → 开发设置”中查看/重置。AppSecret 只应放在 `.env.local` 或生产环境变量中，不能写进小程序代码。若密钥曾经出现在聊天、截图或日志里，应立即在公众平台重置。

### 微信支付配置

真实扣款还需要微信支付商户号，仅有小程序 AppID/AppSecret 不够。完整配置如下：

```bash
WECHAT_MCH_ID=1900000000
WECHAT_PAY_API_V3_KEY=32位的APIv3密钥
WECHAT_PAY_SERIAL_NO=商户API证书序列号
WECHAT_PAY_PRIVATE_KEY_PATH=/absolute/path/apiclient_key.pem
WECHAT_PAY_PLATFORM_PUBLIC_KEY_PATH=/absolute/path/wechatpay_public_key.pem
WECHAT_PAY_PLATFORM_SERIAL_NO=微信支付公钥ID或平台证书序列号
WECHAT_PAY_NOTIFY_URL=https://api.example.com/v1/payments/wechat/notify
WECHAT_REFUND_NOTIFY_URL=https://api.example.com/v1/refunds/wechat/notify
```

获取位置：

- `WECHAT_MCH_ID`：微信支付商户平台首页的商户号。
- `WECHAT_PAY_API_V3_KEY`：商户平台“账户中心 → API安全 → APIv3密钥”。必须是 32 字节，设置后请自行安全保存。
- `WECHAT_PAY_SERIAL_NO` 和 `apiclient_key.pem`：商户平台“账户中心 → API安全 → API证书”申请并下载证书工具生成。序列号在商户平台证书详情中查看，私钥文件通常名为 `apiclient_key.pem`。
- `WECHAT_PAY_PLATFORM_PUBLIC_KEY_PATH` 和 `WECHAT_PAY_PLATFORM_SERIAL_NO`：商户平台“账户中心 → API安全 → 微信支付公钥/平台证书”下载。若使用平台证书，公钥文件可以是 PEM 证书；序列号填证书序列号。若使用新版微信支付公钥，则填公钥 ID。
- 两个回调地址必须是公网可访问的 HTTPS URL，不能是 `127.0.0.1`。本地联调可使用反向代理或内网穿透，并把最终 HTTPS 地址写入环境变量。

同时需要在商户平台确认：小程序 AppID 已与商户号绑定、已开通 JSAPI 支付、商户主体与小程序主体符合微信支付要求。开发登录生成的 `dev:` OpenID 会被后端明确拒绝发起支付，因此用 `DEV_AUTH=true` 只能验证业务流程，不能完成真实扣款。

未配置任何 `WECHAT_MCH_ID/WECHAT_PAY_*` 参数时，服务仍可启动，但支付接口返回 `503 PAYMENT_NOT_CONFIGURED`，不会退回假支付成功。

### 订单状态机

完整状态迁移、唯一约束、支付回调幂等和退款补偿见 [`docs/order-state-machine.md`](docs/order-state-machine.md)。核心链路为：

```text
选人锁定 -> 10 分钟待支付 -> 微信回调确认 -> 已预约
                                      -> 签到 -> 待验收 -> 完成
超时未支付 -> 关闭微信单 -> 释放学生时段 -> 重新竞价/过期
超时后支付成功 -> PAID_LATE -> 自动退款申请，不恢复订单
```

支付成功只能由微信回调或服务端主动查单确认，客户端没有任何“直接标记已支付”的接口。

### 5 公里地理围栏与派单对比

主人发布任务时必须通过微信地图选择服务位置，后端保存位置名称与经纬度。学生进入任务大厅时通过 `wx.getLocation({type: "gcj02"})` 上报当前位置，后端使用 Haversine 公式计算直线距离，只返回 5 公里内的竞价任务；直接打开任务详情和提交报价时都会再次校验距离，不能通过绕过前端跨区查看或报价。

平台派单半径上限为 5km，小程序固定使用 5km；后端允许学生上报 1–5km 的接单半径。学生端不会通过 JSON 获得主人精确经纬度：成交前只展示小区与距离，付款后才按订单权限开放详细门牌。历史上没有定位坐标的旧任务不会进入附近任务大厅，需要重新发布并选择位置。

管理后台的“派单对比”会把当前待竞价任务与已认证学生组成候选边，分别运行贪心和 KM，对比匹配数、总得分、总距离及逐条分配。该功能只用于运营观察算法差异，不会选人、锁单、扣款或改变订单状态。算法、评分和真实自动派单所需边界见 [`docs/dispatch-algorithm.md`](docs/dispatch-algorithm.md)。

### 学籍审核

学生端需提交姓名、学校、学号、入学年份和学信网在线验证码。提交后进入 `PENDING`，运营人员核对学信网报告后通过或驳回；审核通过默认有效期一年。完整方案见 `docs/student-verification.md`。

管理员使用 `ADMIN_TOKEN` 查看待审申请并审核：

```bash
curl 'http://127.0.0.1:8080/v1/admin/student-verifications?status=PENDING' \
  -H 'X-Admin-Token: dev-admin'

curl -X POST http://127.0.0.1:8080/v1/admin/student-verifications/<申请ID>/approve \
  -H 'X-Admin-Token: dev-admin' \
  -H 'Content-Type: application/json' \
  -d '{"validUntil":"2027-07-01"}'
```

驳回时必须填写原因，学生可重新提交。旧接口 `/v1/admin/students/<用户ID>/approve` 仍保留兼容。

## 打开小程序

1. 启动 Go API，默认监听 `http://127.0.0.1:8080`。
2. 微信开发者工具选择“导入项目”，目录选择本仓库的 `miniprogram/`。
3. 项目使用测试号配置与 `urlCheck: false`；开发工具会调用 `wx.login` 并自动登录。
4. 进入“我的 → 编辑昵称与头像”，选择微信头像和昵称并保存；`wx.login` 不会自动提供这两项资料。
5. 真机调试时，将 `miniprogram/app.js` 中的 `apiBase` 改为电脑局域网 IP，并关闭开发工具的合法域名校验。
6. 首次发布任务或进入学生任务大厅时授权定位；`app.json` 已声明 `getLocation` 和 `chooseLocation`。
7. 正式发布需要 HTTPS API 域名、微信 AppID/Secret、合法业务域名，并在微信公众平台“用户隐私保护指引”中声明位置用途。

Docker：

```bash
docker build -t zoo-go-api .
docker run --rm -p 8080:8080 \
  -e DEV_AUTH=true -e ADMIN_TOKEN=dev-admin \
  -v zoo-go-data:/app/data \
  zoo-go-api
```

## 推荐验收链路

1. 主人端首页进入“添加宠物”，创建一只真实宠物。
2. 发布任务，在地图中选择服务位置，填写服务小区和详细门牌，提交后首页出现进行中的任务。
3. 切到学生端，在“我的”页面提交学籍资料与学信网在线验证码；进入任务大厅时授权定位，确认只能看到 5 公里内任务。
4. 打开 `/admin/` 审核该学生，并查看“派单对比”的贪心与 KM 结果；再打开小程序任务提交保密报价。
5. 切回主人端选人并确认支付，订单进入已预约。
6. 学生端签到并上传至少两张真实图片凭证，提交完工。
7. 主人端验收并评价，学生收益中心同步结算数据。
8. 任意身份进入社区，发布文字/图片动态，首页社区 Bar 与社区列表同步更新。

真实支付验收时还需要使用微信账号登录，并配置完整微信支付商户参数；`DEV_AUTH=true` 只能跑通到支付前。

## API

除健康检查、微信登录、管理员审核和 `/uploads/*` 文件读取外，`/v1/*` 接口都要求：

```http
Authorization: Bearer <token>
```

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/v1/auth/wechat/login` | 微信或开发设备登录 |
| GET | `/v1/auth/me` | 当前登录用户 |
| POST | `/v1/auth/logout` | 退出并删除会话 |
| POST | `/v1/profile` | 更新当前用户昵称与头像 |
| GET | `/v1/meta` | 发单配置、日期与费用参数 |
| GET | `/v1/owner/home` | 主人首页、任务、宠物、地址与社区摘要 |
| GET | `/v1/owner/pets` | 当前用户宠物档案 |
| POST | `/v1/pets` | 创建宠物档案 |
| GET | `/v1/owner/tasks?filter=` | 主人订单列表 |
| POST | `/v1/tasks` | 创建任务 |
| GET | `/v1/tasks/{id}?role=` | 任务详情（主人/学生权限裁剪） |
| POST | `/v1/tasks/{id}/bids` | 学生报价/更新报价 |
| POST | `/v1/tasks/{id}/award` | 主人选人并锁定 |
| POST | `/v1/tasks/{id}/reopen` | 待付款重新竞价 |
| POST | `/v1/tasks/{id}/cancel` | 未支付撤单 |
| GET | `/v1/orders/{id}?role=` | 订单详情（地址按权限开放） |
| POST | `/v1/orders/{id}/payments` | 创建微信 JSAPI 支付参数 |
| GET | `/v1/orders/{id}/payments/{paymentId}` | 主动查单并同步支付结果 |
| POST | `/v1/orders/{id}/refunds` | 主人提交退款申请 |
| POST | `/v1/orders/{id}/checkin` | 学生签到 |
| POST | `/v1/orders/{id}/complete` | 提交完工与平台图片凭证 |
| POST | `/v1/orders/{id}/confirm` | 主人验收 |
| POST | `/v1/orders/{id}/rate` | 主人评价 |
| GET | `/v1/student/tasks/available` | 学生任务大厅 |
| POST | `/v1/student/location` | 上报学生当前位置与接单半径 |
| GET | `/v1/student/tasks` | 学生已报价/已中选任务 |
| GET | `/v1/student/earnings` | 学生收益 |
| GET | `/v1/student/profile` | 学生认证资料 |
| POST | `/v1/student/verification` | 提交学籍认证 |
| GET | `/v1/community/posts` | 社区动态分页 |
| POST | `/v1/community/posts` | 发布社区动态 |
| POST | `/v1/uploads` | 上传图片，multipart 字段为 `file` |
| GET | `/v1/admin/overview` | 运营概览 |
| GET | `/v1/admin/dispatch/compare?limit=50` | 贪心与 KM 只读派单对比 |
| GET | `/v1/admin/users?q=&status=ALL` | 管理员查询用户 |
| POST | `/v1/admin/users/{id}/status` | 管理员停用或恢复用户 |
| GET | `/v1/admin/orders?status=ALL` | 管理员查询订单 |
| GET | `/v1/admin/audit-logs?limit=100` | 管理员查询审计日志 |
| GET | `/v1/admin/refunds?status=REQUESTED` | 管理员查看退款申请 |
| GET | `/v1/admin/student-verifications?status=PENDING` | 管理员查看学籍待审申请 |
| POST | `/v1/admin/student-verifications/{id}/approve` | 通过学籍认证并设置有效期 |
| POST | `/v1/admin/student-verifications/{id}/reject` | 驳回学籍认证并填写原因 |
| POST | `/v1/admin/refunds/{id}/approve` | 管理员审核并调用微信退款 |
| POST | `/v1/admin/refunds/{id}/reject` | 管理员驳回普通退款申请 |
| POST | `/v1/payments/wechat/notify` | 微信支付回调（验签，无需 Bearer Token） |
| POST | `/v1/refunds/wechat/notify` | 微信退款回调（验签，无需 Bearer Token） |

响应统一为 `{"data": ...}`，错误为 `{"error":{"code":"...","message":"..."}}`。

## 校验

```bash
gofmt -w cmd internal
go test -race ./...
go vet ./...
go build ./cmd/api
for f in $(find miniprogram -name '*.js'); do node --check "$f"; done
node scripts/validate-miniprogram.mjs
```

## 生产边界

- 微信支付 v3 已接入 JSAPI 预下单、主动查单、支付/退款回调验签解密、关单和退款查询；上线前必须用真实商户参数完成 1 分钱验收。
- 退款当前为人工审核：主人提交申请，管理员调用审核接口后才发起原路退款。
- 学籍认证当前为“学信网验证码 + 人工审核”；实名与 18 岁校验仍需接入持牌服务商。
- 运营后台当前使用静态 `ADMIN_TOKEN`，正式环境必须接入管理员账号、RBAC、MFA、登录限流和独立审计存储。
- 微信支付公钥/平台证书轮换时，需要同步更换 `WECHAT_PAY_PLATFORM_PUBLIC_KEY_PATH` 和 `WECHAT_PAY_PLATFORM_SERIAL_NO`。
- 虚拟号、订阅消息、对象存储、客服工单和退款分账仍需接入对应服务商。
- 当前距离为直线距离，不包含道路、交通和天气；上线前应按业务需要评估路线距离、定位时效和学生可接单状态。
- 当前不会自动派单。若要自动成交，需要补充任务批次快照、带超时的唯一领取、位置/学籍/时段复核、通知重试和失败回退，不能只用算法结果直接改写订单。
- SQLite 适合单实例部署。多实例或高并发生产环境应迁移到 MySQL/PostgreSQL，并保留事务、唯一索引和权限裁剪。
- 门牌、证件等敏感信息在生产环境必须加密、最小权限访问并记录审计日志。
