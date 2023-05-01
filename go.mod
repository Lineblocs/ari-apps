module lineblocs.com/processor

go 1.17

require (
	cloud.google.com/go/speech v1.0.0
	cloud.google.com/go/texttospeech v1.0.0
	github.com/CyCoreSystems/ari-proxy/v5 v5.2.2
	github.com/CyCoreSystems/ari/v5 v5.2.0
	github.com/Lineblocs/go-helpers v0.0.4-0.20230424211213-b9c5f29df796
	github.com/aws/aws-sdk-go v1.44.161
	github.com/go-redis/redis/v8 v8.11.4
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.4.2
	github.com/joho/godotenv v1.4.0
	github.com/rotisserie/eris v0.5.1
	github.com/sirupsen/logrus v1.9.0
	github.com/stretchr/testify v1.7.0
	github.com/u2takey/ffmpeg-go v0.3.0
	golang.org/x/net v0.1.0
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
	google.golang.org/api v0.60.0
	google.golang.org/genproto v0.0.0-20211021150943-2b146023228c
	google.golang.org/grpc v1.41.0
	google.golang.org/protobuf v1.27.1
)

require (
	cloud.google.com/go v0.97.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.17.6 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.18.18 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.13.17 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.24 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.31 // indirect
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.20.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.24 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.12.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.14.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.18.6 // indirect
	github.com/aws/smithy-go v1.13.5 // indirect
	github.com/bshuster-repo/logrus-logstash-hook v1.1.0 // indirect
	github.com/census-instrumentation/opencensus-proto v0.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/clockworksoul/smudge v1.0.1 // indirect
	github.com/cncf/udpa/go v0.0.0-20201120205902-5459f2c99403 // indirect
	github.com/cncf/xds/go v0.0.0-20210805033703-aa0b78936158 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/envoyproxy/go-control-plane v0.9.10-0.20210907150352-cf90f659a021 // indirect
	github.com/envoyproxy/protoc-gen-validate v0.1.0 // indirect
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.8 // indirect
	github.com/googleapis/gax-go/v2 v2.1.1 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/inconshreveable/log15 v0.0.0-20201112154412-8562bdadbbac // indirect
	github.com/innix/logrus-cloudwatch v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/mailgun/mailgun-go/v4 v4.8.1 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/nats-io/nats.go v1.8.1 // indirect
	github.com/nats-io/nkeys v0.0.2 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/stripe/stripe-go/v71 v71.48.0 // indirect
	github.com/t-tomalak/logrus-easy-formatter v0.0.0-20190827215021-c074f06c5816 // indirect
	github.com/ttacon/builder v0.0.0-20170518171403-c099f663e1c2 // indirect
	github.com/ttacon/libphonenumber v1.2.1 // indirect
	github.com/u2takey/go-utils v0.0.0-20210821132353-e90f7c6bacb5 // indirect
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519 // indirect
	golang.org/x/sys v0.1.0 // indirect
	golang.org/x/text v0.4.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace github.com/CyCoreSystems/ari/v5 => github.com/nadirhamid/ari/v5 v5.2.5

replace github.com/CyCoreSystems/ari-proxy/v5 => github.com/nadirhamid/ari-proxy/v5 v5.2.4
