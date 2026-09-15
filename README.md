# 刮刮乐游戏平台

面向移动端浏览器的 H5 小游戏平台。首款游戏为刮刮乐，账号、游戏配置和奖励能力会作为公共平台模块建设。

## 已确定的产品边界

- H5 网页
- 用户需要注册后参与
- 仅使用虚拟金币
- 不充值、不提现、不交易、不兑换现金
- 游戏主题、图案、奖项和概率方案待产品讨论后确定

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

启动后，Web 通过 80 端口访问，API 健康检查地址为 `/api/v1/health`。

