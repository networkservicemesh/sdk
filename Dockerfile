FROM golang:alpine as build

WORKDIR /go/src/github.com/networkservicemesh/sdk
COPY go.mod .

RUN go mod download

COPY . .
RUN go install -v github.com/networkservicemesh/sdk/cmd/icmp-server
RUN go install -v github.com/networkservicemesh/sdk/cmd/healthcheck

FROM alpine as runtime

COPY --from=build /go/bin/icmp-server /bin/icmp-server
COPY --from=build /go/bin/healthcheck /bin/healthcheck

CMD /bin/icmp-server