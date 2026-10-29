# Build Stage
FROM node:20-alpine AS builder
WORKDIR /app

# 优先复制依赖描述文件利用构建缓存
COPY package*.json ./
RUN npm ci

# 复制源码并执行 TypeScript 编译与 Vite 打包
COPY . .
RUN npm run build

# Runtime Stage
FROM nginx:1.27-alpine
WORKDIR /usr/share/nginx/html

# 清理默认 HTML 页面并载入构建产物与 Nginx 路由配置
RUN rm -rf ./*
COPY --from=builder /app/dist .
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
