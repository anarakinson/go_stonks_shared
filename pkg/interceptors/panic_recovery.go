package interceptors

import (
	"context"
	"log/slog"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryPanicRecovery - перехватывает паники и преобразует в gRPC ошибку
func UnaryPanicRecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (_ interface{}, err error) {

		defer func() {
			if r := recover(); r != nil {
				slog.Info("panic recovered", "recovery info", r, "stack", string(debug.Stack()))
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)

	}
}
