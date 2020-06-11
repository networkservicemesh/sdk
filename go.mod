module github.com/networkservicemesh/sdk

go 1.13

require (
	github.com/RoaringBitmap/roaring v0.4.23
	github.com/cheekybits/is v0.0.0-20150225183255-68e9c0620927 // indirect
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/edwarnicke/exechelper v1.0.1
	github.com/golang/protobuf v1.3.5
	github.com/google/uuid v1.1.1
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/matryer/try v0.0.0-20161228173917-9ac251b645a2
	github.com/nats-io/nats-streaming-server v0.17.0
	github.com/nats-io/stan.go v0.6.0
	github.com/networkservicemesh/api v0.0.0-20200525170518-89690ec70489
	github.com/open-policy-agent/opa v0.16.1
	github.com/opentracing/opentracing-go v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.3
	github.com/spiffe/go-spiffe/v2 v2.0.0-alpha.4.0.20200528145730-dc11d0c74e85
	github.com/stretchr/testify v1.5.1
	github.com/uber/jaeger-client-go v2.21.1+incompatible
	github.com/uber/jaeger-lib v2.2.0+incompatible // indirect
	go.uber.org/goleak v1.0.0
	golang.org/x/sys v0.0.0-20200501145240-bc7a7d42d5c3
	gonum.org/v1/gonum v0.6.2
	google.golang.org/grpc v1.27.1
)

replace (
    github.com/networkservicemesh/api => ../api
)