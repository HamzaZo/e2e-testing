FROM golang:1.19-alpine3.17 As builder

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY main.go main.go
COPY cmd/ cmd/
COPY internal/  internal/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o bin/e2e main.go


FROM alpine:3.17.0

RUN addgroup nonroot && \
    adduser -S -G nonroot nonroot

USER nonroot

WORKDIR /

COPY --chown=nonroot:nonroot --from=builder /workspace/bin/e2e /e2e

ENTRYPOINT ["/e2e"]