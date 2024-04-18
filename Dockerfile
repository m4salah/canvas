FROM golang:1-bullseye AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download -x


COPY . ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X 'main.release=`git rev-parse --short=8 HEAD`'" -o /bin/server ./cmd/server

FROM gcr.io/distroless/static-debian11
WORKDIR /app

COPY --from=builder /bin/server ./

CMD ["./server"]

