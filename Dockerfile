FROM golang:1-bullseye AS builder
WORKDIR /src

# general rule: you should generally do things that don't change very often earlier in the Dockerfile than things that do change more often.
# we copy the source code later.
# because our dependency doesn't change often, but our code does.
# install our deps
COPY go.mod go.sum ./
RUN go mod download -x

# then copy our source code
COPY . ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X 'main.release=`git rev-parse --short=8 HEAD`'" -o /bin/server ./cmd/server

FROM gcr.io/distroless/static-debian11
WORKDIR /app

COPY --from=builder /bin/server ./

CMD ["./server"]

