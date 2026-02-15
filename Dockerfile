# Stage 1: Build the Vue.js frontend
FROM node:lts-alpine as builder_frontend
WORKDIR /app/beango-web
RUN npm install -g pnpm # Install pnpm
COPY beango-web/package.json beango-web/pnpm-lock.yaml ./
RUN rm -rf node_modules # Clean up node_modules before installing
RUN pnpm install --frozen-lockfile # Use pnpm install
COPY beango-web/index.html ./index.html
COPY beango-web/vite.config.ts ./vite.config.ts
COPY beango-web/tsconfig.json ./tsconfig.json
COPY beango-web/tsconfig.app.json ./tsconfig.app.json
COPY beango-web/tsconfig.node.json ./tsconfig.node.json
COPY beango-web/src ./src
COPY beango-web/public ./public
RUN pnpm run build

# Stage 2: Build the Go backend
FROM golang:alpine as builder_backend
WORKDIR /app
# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download
# Install build-base for CGO and sqlite development headers
RUN apk add --no-cache build-base sqlite-dev
# Copy the rest of the backend source code
COPY . .
# Build the backend application with CGO_ENABLED=1 and gcc
RUN CGO_ENABLED=1 GOOS=linux CC=gcc go build -o /app/beango .

# Stage 3: Final image
FROM alpine/git as runner
WORKDIR /app
# Install sqlite runtime libraries
RUN apk add --no-cache sqlite-libs

# Copy built backend from builder_backend stage
COPY --from=builder_backend /app/beango ./

# Copy built frontend assets from builder_frontend stage
COPY --from=builder_frontend /app/beango-web/dist ./web/dist

# Expose the port the Gin server listens on
EXPOSE 10777

# Declare /out as a volume so it can be mounted by the host
VOLUME /out

# Command to run the application
ENTRYPOINT ["./beango"]
