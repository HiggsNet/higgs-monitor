FROM golang:alpine as build
ADD . monitor
WORKDIR monitor
RUN apk add make git
RUN make

FROM alpine:edge
COPY --from=build /go/monitor/build/client /usr/local/bin/client

ENTRYPOINT ["/usr/local/bin/client"]
