FROM golang:1.25-alpine AS builder
WORKDIR /app
RUN adduser -D -u 10001 scratchuser
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o music-recomendations .
RUN mkdir -p /app/data && chown -R scratchuser:scratchuser /app/data

FROM scratch
COPY --from=builder /app/music-recomendations /music-recomendations
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder --chown=10001:10001 /app/data /app/data
USER scratchuser
EXPOSE 8080
ENTRYPOINT ["/music-recomendations"]
