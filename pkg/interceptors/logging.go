package interceptors

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func UnaryLoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {

		// Засекаем время выполнения
		startTime := time.Now()

		// Логируем входящий запрос
		logger.Info(
			"gRPC request",
			zap.String("method", info.FullMethod),
			zap.Any("request", req),
		)

		// Вызываем следующий обработчик
		resp, err = handler(ctx, req)

		// Логируем ошибку (если есть)
		if err != nil {
			st, _ := status.FromError(err)
			logger.Error(
				"gRPC error",
				zap.String("method", info.FullMethod),
				zap.Error(err),
				zap.Any("status_code", st.Code()),
				zap.String("status_message", st.Message()),
				zap.Duration("duration", time.Since(startTime)),
			)
		} else {
			// Логируем успешный ответ
			logger.Info(
				"gRPC response",
				zap.String("method", info.FullMethod),
				zap.Any("response", resp),
				zap.Duration("duration", time.Since(startTime)),
			)
		}

		return resp, err

	}
}
