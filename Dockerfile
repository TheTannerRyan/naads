# Gopherfile
# github.com/TheTannerRyan/Gopherfile

FROM golang:alpine as build
ENV GOPATH /go

RUN adduser -D -g '' gopher
COPY . /go/src/github.com/thetannerryan/naads
WORKDIR /go/src/github.com/thetannerryan/naads

RUN apk update
RUN apk --no-cache add ca-certificates tzdata

RUN apk add git
RUN go get -u github.com/golang/dep/cmd/dep
RUN dep ensure --vendor-only

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o /go/bin/exec /go/src/github.com/thetannerryan/naads/example/client.go

FROM scratch

COPY --from=build /go/src/github.com/thetannerryan/naads/status.html /go/bin/
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /go/bin/exec /go/bin/exec

ENV TZ America/Toronto
USER gopher
WORKDIR /go/bin

ENTRYPOINT ["/go/bin/exec"]
EXPOSE 6060
