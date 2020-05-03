FROM golang:alpine as build

WORKDIR /go/src/github.com/networkservicemesh/sdk
COPY go.mod .

RUN go mod download

COPY . .
RUN go install -v github.com/networkservicemesh/sdk/cmd/icmp-server
RUN go install -v github.com/networkservicemesh/sdk/cmd/healthcheck
RUN go install -v github.com/networkservicemesh/sdk/cmd/nsmgr
RUN go install -v github.com/networkservicemesh/sdk/cmd/register

FROM alpine as runtime

RUN apk update
RUN apk add tmux

RUN mkdir -p /run/networkservicemesh/

COPY --from=build /go/bin/icmp-server /bin/icmp-server
COPY --from=build /go/bin/healthcheck /bin/healthcheck
COPY --from=build /go/bin/nsmgr /bin/nsmgr
COPY --from=build /go/bin/register /bin/register

CMD /bin/icmp-server