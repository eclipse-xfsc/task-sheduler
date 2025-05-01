FROM golang:1.24.2-alpine3.20 as builder

RUN apk add git

WORKDIR /go/src/github.com/eclipse-xfsc/task-sheduler

ARG APP_REPO_TAG

ADD . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X main.Version=$APP_REPO_TAG" -mod=vendor -o /tmp/task ./cmd/task/...

FROM alpine:3.19 as runner

COPY --from=builder /tmp/task /opt/task

WORKDIR /opt

CMD ["./task"]
