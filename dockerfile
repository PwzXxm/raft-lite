# building
FROM golang:1 as builder
WORKDIR /src/
# get&cache dependancies first
COPY go.mod .
RUN go mod download
# copy & build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o raft-lite .

# actual image
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app/
COPY --from=builder /src/raft-lite .
CMD ["./raft-lite", "peer", "-c", "/data/config.json"]
