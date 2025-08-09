package grpc_helpers

import (
	"crypto/tls"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

func NewGRPCClient(
	addr string,
	tlsConfig *tls.Config,
	interceptors ...grpc.UnaryClientInterceptor,
) (*grpc.ClientConn, error) {
	var creds credentials.TransportCredentials

	if tlsConfig != nil {
		creds = credentials.NewTLS(tlsConfig)
	} else {
		creds = insecure.NewCredentials()
	}

	opts := []grpc.DialOption{
		// security
		grpc.WithTransportCredentials(creds),
		// OpenTelemetry трассировщик
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		// добавляем интерсепторы
		grpc.WithChainUnaryInterceptor(interceptors...),
		// поддержка соединения
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		// балансировщик нагрузки
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
		// Параметры подключения
		grpc.WithConnectParams(grpc.ConnectParams{
			MinConnectTimeout: 5 * time.Second,
			Backoff:           backoff.DefaultConfig,
		}),
	}

	return grpc.NewClient(addr, opts...)
}
