package interceptors

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// XRequestIDInterceptor - unary interceptor для добавления X-Request-ID
func XRequestIDClient() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Получаем метаданные из контекста или создаем новые
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		// Проверяем, есть ли уже X-Request-ID
		requestIDs := md.Get("x-request-id")
		if len(requestIDs) == 0 {
			// Генерируем новый UUID
			requestID := uuid.New().String()
			md.Set("x-request-id", requestID)
		}

		// Создаем новый контекст с обновленными метаданными
		newCtx := metadata.NewOutgoingContext(ctx, md.Copy())

		// Продолжаем выполнение запроса с новым контекстом
		return invoker(newCtx, method, req, reply, cc, opts...)
	}
}
