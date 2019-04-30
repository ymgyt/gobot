FROM golang:1.12.4-alpine3.9 as build

WORKDIR /go/src/github.com/ymgyt/gobot

ENV GO111MODULE=off

RUN apk --no-cache add ca-certificates

COPY . ./

ARG VERSION

RUN echo $VERSION

RUN CGO_ENABLED=0 go build -o /gobot -ldflags "-X \"github.com/ymgyt/gobot/app.Version=$VERSION\""


FROM alpine:3.9

WORKDIR /root

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /gobot .

EXPOSE 80
EXPOSE 443

ENTRYPOINT ["./gobot"]