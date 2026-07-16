# Stage 1: Build the Vue.js frontend
FROM node:lts-alpine as builder_frontend
WORKDIR /app/beango-web
RUN npm install -g pnpm
COPY beango-web/package.json beango-web/pnpm-lock.yaml ./
RUN rm -rf node_modules
RUN pnpm install --frozen-lockfile
COPY beango-web/index.html ./index.html
COPY beango-web/vite.config.ts ./vite.config.ts
COPY beango-web/tsconfig.json ./tsconfig.json
COPY beango-web/tsconfig.app.json ./tsconfig.app.json
COPY beango-web/tsconfig.node.json ./tsconfig.node.json
COPY beango-web/src ./src
COPY beango-web/public ./public
RUN pnpm run build

# Stage 2: Build the Go backend (CGO disabled, pure Go)
FROM golang:alpine as builder_backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/beango .

# Stage 3: Final image (minimal)
FROM alpine:latest as runner
WORKDIR /app

COPY --from=builder_backend /app/beango ./
COPY --from=builder_frontend /app/beango-web/dist ./web/dist
COPY config/ ./config/

EXPOSE 10777
VOLUME /out

ENTRYPOINT ["./beango"]
