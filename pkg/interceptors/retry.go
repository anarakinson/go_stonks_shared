package interceptors

import (
	"context"
	"math"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RetryInterceptor(maxRetries int) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		var lastErr error

		for i := 0; i < maxRetries; i++ {
			// Пробуем выполнить запрос
			lastErr = invoker(ctx, method, req, reply, cc, opts...)

			// Если успешно - возвращаем результат
			if lastErr == nil {
				return nil
			}

			// Проверяем, нужно ли повторять для этой ошибки
			if !isRetriableError(lastErr) {
				return lastErr
			}

			// Ждём перед повторной попыткой
			select {
			case <-time.After(ExponentialBackoff(i + 1)): // экспоненциальный backoff
			case <-ctx.Done():
				return ctx.Err() // если контекст отменили
			}
		}

		return lastErr
	}
}

// Проверяем, стоит ли повторять запрос
func isRetriableError(err error) bool {
	if err == nil {
		return false
	}

	// Преобразуем в gRPC статус
	status, ok := status.FromError(err)
	if !ok {
		return false // не gRPC ошибка
	}

	// Ретраим только для этих кодов:
	switch status.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}

// Backoff стратегия (экспоненциальная)
func ExponentialBackoff(attempt int) time.Duration {
	return time.Duration(math.Pow(2, float64(attempt))) * time.Second
}
