module github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar

go 1.16

require (
	github.com/golang/mock v1.6.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.11.3
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1
	github.com/heetch/confita v0.10.0
	github.com/jmoiron/sqlx v1.3.5
	github.com/kr/pretty v0.3.0 // indirect
	github.com/lib/pq v1.2.0
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/pkg/errors v0.9.1
	github.com/rabbitmq/amqp091-go v1.5.0
	github.com/rs/xid v1.4.0
	github.com/rs/zerolog v1.28.0
	github.com/stretchr/testify v1.8.0
	golang.org/x/net v0.0.0-20220907135653-1e95f45603a7 // indirect
	golang.org/x/sys v0.0.0-20220908150016-7ac13a9a928d // indirect
	google.golang.org/genproto v0.0.0-20220822174746-9e6da59bd2fc
	google.golang.org/grpc v1.48.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.2.0
	google.golang.org/protobuf v1.28.1
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace github.com/hihoak/otus-cource-hws/hw12_13_14_15_calendar/pkg => ./pkg
