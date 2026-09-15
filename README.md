# 刮刮乐游戏平台

支持 PC 与移动端浏览器的 H5 小游戏平台。首款游戏为刮刮乐，账号、游戏配置、金币账本和奖励能力作为公共平台模块建设。

## 已确定的产品边界

- PC 与移动端 H5 网页
- 用户需要注册后参与
- 仅使用虚拟金币
- 不充值、不提现、不交易、不兑换现金
- 第一款《零钱小票》的主题、图案、奖项和概率已经确认

## 当前可运行闭环

- 用户注册、登录、注销与 30 天会话
- 新用户初始 1000 虚拟金币
- 八款卡片价格和动态余额门槛展示
- 购买《零钱小票》并在服务端提前锁定结果
- Canvas 刮层、三格结果揭示和中奖兑奖
- MySQL 金币事务、幂等购卡和金币流水
- 未配置 MySQL 时，本地开发自动使用内存存储；数据会在 API 重启后清空

## 技术栈

- Web：React、TypeScript、Vite、Canvas 2D
- API：Go
- 数据：MySQL、Redis
- 部署：Docker Compose、Nginx

## 目录

```text
apps/
  api/        Go API 服务
  web/        H5 前端
deploy/
  nginx/      Web 与 API 反向代理配置
compose.yaml  本地和服务器统一容器编排
```

## 本地开发

前端：

```bash
npm install
npm run dev
```

后端：

```bash
cd apps/api
go run ./cmd/server
```

运行检查：

```bash
npm run typecheck
npm run build
cd apps/api && go test ./...
```

## Docker 部署

复制环境变量模板并替换所有密码：

```bash
cp .env.example .env
docker compose up -d --build
```

启动后，Web 通过 80 端口访问，API 存活检查为 `/api/v1/health`，数据库就绪检查为 `/api/v1/ready`。API 启动时自动执行尚未应用的数据库迁移。

测试阶段使用裸 IP 和 HTTP 时保持 `COOKIE_SECURE=false`；配置 HTTPS 后必须改成 `COOKIE_SECURE=true`。

## 服务器维护

服务器项目目录为 `/opt/guaguale`。更新已部署版本：

```bash
./deploy/scripts/update-server.sh
```

MySQL 每日备份由 systemd 定时器执行，默认在每天凌晨 03:20 后随机延迟最多 15 分钟运行，备份保存在 `backups/mysql`，保留 14 天。正式上线前需要将备份同步到独立的对象存储。
