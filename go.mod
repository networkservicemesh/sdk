module github.com/networkservicemesh/sdk

go 1.23

require (
	github.com/RoaringBitmap/roaring v0.9.4
	github.com/antonfisher/nested-logrus-formatter v1.3.1
	github.com/benbjohnson/clock v1.3.0
	github.com/edwarnicke/exechelper v1.0.2
	github.com/edwarnicke/genericsync v0.0.0-20220910010113-61a344f9bc29
	github.com/edwarnicke/grpcfd v1.1.4
	github.com/edwarnicke/serialize v1.0.7
	github.com/fsnotify/fsnotify v1.7.0
	github.com/ghodss/yaml v1.0.0
	github.com/golang-jwt/jwt/v4 v4.5.1
	github.com/golang/protobuf v1.5.4
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.6.0
	github.com/miekg/dns v1.1.57
	github.com/nats-io/nats-streaming-server v0.24.6
	github.com/nats-io/stan.go v0.10.3
	github.com/networkservicemesh/api v1.14.2-rc.1.0.20241209080353-bbb4cd5f8f00
	github.com/open-policy-agent/opa v0.68.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.20.2
	github.com/r3labs/diff v1.1.0
	github.com/sirupsen/logrus v1.9.3
	github.com/spiffe/go-spiffe/v2 v2.1.7
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.46.1
	go.opentelemetry.io/otel v1.28.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.43.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.28.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.28.0
	go.opentelemetry.io/otel/exporters/prometheus v0.43.0
	go.opentelemetry.io/otel/exporters/zipkin v1.20.0
	go.opentelemetry.io/otel/metric v1.28.0
	go.opentelemetry.io/otel/sdk v1.28.0
	go.opentelemetry.io/otel/sdk/metric v1.20.0
	go.opentelemetry.io/otel/trace v1.28.0
	go.uber.org/atomic v1.9.0
	go.uber.org/goleak v1.3.1-0.20241121203838-4ff5fa6529ee
	golang.org/x/net v0.28.0
	gonum.org/v1/gonum v0.6.2
	google.golang.org/grpc v1.66.0
	google.golang.org/protobuf v1.34.2
	gopkg.in/yaml.v2 v2.4.0
)

require (
	cloud.google.com/go/compute/metadata v0.3.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/OneOfOne/xxhash v1.2.8 // indirect
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.2.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fatih/color v1.14.1 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-jose/go-jose/v3 v3.0.3 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/flatbuffers v2.0.8+incompatible // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.20.0 // indirect
	github.com/hashicorp/go-hclog v1.5.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-msgpack v1.1.5 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/raft v1.3.9 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/minio/highwayhash v1.0.2 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nats-io/jwt/v2 v2.5.0 // indirect
	github.com/nats-io/nats-server/v2 v2.9.23 // indirect
	github.com/nats-io/nats.go v1.31.0 // indirect
	github.com/nats-io/nkeys v0.4.6 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/openzipkin/zipkin-go v0.4.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/tchap/go-patricia/v2 v2.3.1 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/yashtewari/glob-intersection v0.2.0 // indirect
	github.com/zeebo/errs v1.3.0 // indirect
	go.etcd.io/bbolt v1.3.10 // indirect
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	golang.org/x/crypto v0.26.0 // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.23.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	golang.org/x/time v0.6.0 // indirect
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240701130421-f6361c86f094 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240701130421-f6361c86f094 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)
