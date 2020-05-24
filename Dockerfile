FROM golang:alpine as build

WORKDIR /go/src/github.com/networkservicemesh/sdk

COPY go.mod go.sum ./
COPY ./pkg/imports/ ./pkg/imports/
RUN go build ./pkg/imports/

COPY . .
RUN go install -v github.com/networkservicemesh/sdk/cmd/icmp-server
RUN go install -v github.com/networkservicemesh/sdk/cmd/icmp-client
RUN go install -v github.com/networkservicemesh/sdk/cmd/healthcheck
RUN go install -v github.com/networkservicemesh/sdk/cmd/nsmgr
RUN go install -v github.com/networkservicemesh/sdk/cmd/registry

FROM alpine as runtime

RUN apk update
RUN apk add tmux

RUN mkdir -p /run/networkservicemesh/

COPY --from=build /go/bin/icmp-server /bin/icmp-server
COPY --from=build /go/bin/icmp-client /bin/icmp-client
COPY --from=build /go/bin/healthcheck /bin/healthcheck
COPY --from=build /go/bin/nsmgr /bin/nsmgr
COPY --from=build /go/bin/registry /bin/registry
COPY --from=build /go/src/github.com/networkservicemesh/sdk/run.sh /bin/run.sh
RUN chmod +x /bin/run.sh

CMD /bin/icmp-server
