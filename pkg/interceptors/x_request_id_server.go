package interceptors

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type contextKey string

const (
	requestIDKey contextKey = "x-request-id"
)

// XRequestIDServer - извлекает x-Request-id из входящих запросов и передавает его в контексте
func XRequestIDServer() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Извлекаем метаданные из входящего запроса
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		// Получаем или генерируем request id
		requestIDs := md.Get("x-request-id")
		var requestID string
		if len(requestIDs) == 0 {
			requestID = uuid.New().String()
		} else {
			requestID = requestIDs[0]
		}

		// Добавляем request id в контекст, сохраняя оригинальные значения
		ctx = context.WithValue(ctx, requestIDKey, requestID) // Для текущего сервиса

		// Продолжаем обработку запроса
		return handler(ctx, req)
	}
}
