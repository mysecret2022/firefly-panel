# Stage 1: Build Frontend (使用原生宿主机平台极速构建，产物是与CPU架构无关的纯HTML/JS)
FROM --platform=$BUILDPLATFORM node:18-alpine AS web_image

WORKDIR /build

# 固定使用与项目兼容的 pnpm 8 版本
RUN npm install -g pnpm@8.15.9

COPY ./package.json ./pnpm-lock.yaml ./

RUN pnpm install

COPY . .

RUN pnpm run build-only

# Stage 2: Build Backend (针对不同平台自动编译 CGO SQLite)
FROM golang:1.21-alpine3.18 AS server_image

WORKDIR /build

COPY ./service .

RUN apk add --no-cache bash curl gcc git musl-dev

RUN go env -w GO111MODULE=on \
    && export PATH=$PATH:/go/bin \
    && go install -a -v github.com/go-bindata/go-bindata/...@latest \
    && go install -a -v github.com/elazarl/go-bindata-assetfs/...@latest \
    && go-bindata-assetfs -o=assets/bindata.go -pkg=assets assets/... \
    && go build -o sun-panel --ldflags="-X sun-panel/global.RUNCODE=release -X sun-panel/global.ISDOCKER=docker" main.go

# Stage 3: Runtime
FROM alpine:3.18

WORKDIR /app

COPY --from=web_image /build/dist /app/web
COPY --from=server_image /build/sun-panel /app/sun-panel

EXPOSE 3002

RUN apk add --no-cache bash ca-certificates su-exec tzdata \
    && chmod +x ./sun-panel \
    && ./sun-panel -config

CMD ["./sun-panel"]
