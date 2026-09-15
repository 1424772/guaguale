FROM node:22-alpine AS builder

WORKDIR /workspace
COPY package.json package-lock.json ./
COPY apps/web/package.json apps/web/package.json
RUN npm ci

COPY apps/web ./apps/web
RUN npm run build --workspace @guaguale/web

FROM nginx:1.28-alpine

COPY deploy/nginx/default.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /workspace/apps/web/dist /usr/share/nginx/html

EXPOSE 80

