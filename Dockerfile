FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/worker ./cmd/worker

FROM alpine:3.22 AS runtime
RUN apk add --no-cache ca-certificates wget && addgroup -S agentscope && adduser -S -G agentscope agentscope
WORKDIR /app
COPY --from=build /out/api /app/api
COPY --from=build /out/worker /app/worker
USER agentscope
EXPOSE 8080
ENTRYPOINT ["/app/api"]
