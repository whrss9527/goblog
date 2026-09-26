# 自托管：docker compose up -d --build
# 数据库（PGlite）和照片默认都放在 /app/.data，记得挂载成数据卷。

FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:22-alpine AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
# NEXT_PUBLIC_TIMEZONE is compiled into the browser code, so it is a build argument.
ARG NEXT_PUBLIC_TIMEZONE=Asia/Shanghai
ENV NEXT_PUBLIC_TIMEZONE=$NEXT_PUBLIC_TIMEZONE \
    NEXT_OUTPUT=standalone \
    NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM node:22-alpine
WORKDIR /app
ENV NODE_ENV=production \
    PORT=3000 \
    HOSTNAME=0.0.0.0 \
    NEXT_TELEMETRY_DISABLED=1
COPY --from=build /app/.next/standalone ./
COPY --from=build /app/.next/static ./.next/static
COPY --from=build /app/public ./public
RUN mkdir -p /app/.data && chown -R node:node /app/.data
USER node
VOLUME /app/.data
EXPOSE 3000
CMD ["node", "server.js"]
