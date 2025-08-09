package interceptors

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type RedisCacheInterceptor struct {
	redis     *redis.Client
	pubSub    *redis.PubSub
	maxMemory int64
}

var (
	errTooBigData  = errors.New("data is too big for caching")
	errInvalidType = errors.New("invalid type")
)

// создает новый интерсептор на основе клиента редис
func NewRedisCacheInterceptor(redis *redis.Client) *RedisCacheInterceptor {
	var maxMemory int64 = 0
	config, err := redis.ConfigGet(context.Background(), "maxmemory").Result()
	if err != nil {
		slog.Error("Failed to get Redis config", "error", err)
	} else {
		maxMemoryStr := config["maxmemory"]
		maxMemory, err = strconv.ParseInt(maxMemoryStr, 10, 64)
		if err != nil {
			slog.Error("Failed to parse maxMemory value", "error", err)
			maxMemory = 0
		}
	}

	return &RedisCacheInterceptor{
		redis:     redis,
		maxMemory: maxMemory,
	}
}

// Subscribe - подписывается на событие инвалидации
// принимает ключ, который будет инвалидироваться, и ключ, по которому срабатывает инвалидация
func (i *RedisCacheInterceptor) Subscribe(cacheKey, invalidationKey string) error {

	// Подписываемся на события инвалидации
	i.pubSub = i.redis.Subscribe(context.Background(), invalidationKey)
	if _, err := i.pubSub.Receive(context.Background()); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	go i.listenForInvalidations(cacheKey)

	return nil

}

// listenForInvalidations - слушает события инвалидации
func (i *RedisCacheInterceptor) listenForInvalidations(cacheKey string) {
	ch := i.pubSub.Channel()
	for msg := range ch {
		if msg.Payload == cacheKey {
			i.redis.Del(context.Background(), cacheKey)
			slog.Info("Cache invalidated", "key", cacheKey)
		}
	}
}

// Unary - создает непосредственно интерсептор, кеширующий данные, возвращаемые запросом
// метод принимает:
// cacheKey - ключ редиса, который кешируется,
// methodName - метод(запрос), на котором срабатывает,
// ttl - время жизни кеша
func (i *RedisCacheInterceptor) Unary(
	cacheKey string,
	methodName string,
	ttl time.Duration,
) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Кешируем только указанный метод
		if method != methodName {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		// Пробуем получить из кеша
		cachedData, err := i.redis.Get(ctx, cacheKey).Bytes()
		if err == nil {
			// пытаемся десериализовать
			if err = proto.Unmarshal(cachedData, reply.(proto.Message)); err != nil {
				// удаляем ключ, который невозможно десериализовать и продолжаем
				slog.Info("Error unmarshaling cached data", "cache key", cacheKey)
				i.redis.Del(ctx, cacheKey)
			} else {
				// возвращаем кеш
				slog.Info("Returning cached data", "cache key", cacheKey)
				return nil
			}
		}
		if err != redis.Nil {
			// если ошибка не в отсутствии ключа - логируем проблему
			slog.Error("Redis get error", "error", err, "key", cacheKey)
		}

		// Вызываем оригинальный метод
		if err := invoker(ctx, method, req, reply, cc, opts...); err != nil {
			return err
		}

		// проверяем, возможно ли преобразовать ответ к нужному типу данных
		msg, ok := reply.(proto.Message)
		if !ok {
			slog.Error("Failed to serialize data to type", "error", errInvalidType)
			return errInvalidType
		}
		// Сохраняем в кеш
		if data, err := proto.Marshal(msg); err == nil {
			// проверяем, поместятся ли полученные данные в кеш
			if len(data) > int(i.maxMemory) {
				slog.Error("Failed to save data to cache", "error", errTooBigData)
				return errTooBigData
			}
			// кешируем
			if err := i.redis.Set(ctx, cacheKey, data, ttl).Err(); err != nil {
				slog.Error("Failed to cache data", "error", err)
			} else {
				slog.Info("Successfully cached data", "cache key", cacheKey)
			}
		}

		return nil
	}
}

// ------------------------------------------ //
// ---------------- example ----------------- //

// ---------- on subscriber site: ----------- //

/*

// создаем клиент редиса
	// 1. Инициализация Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})

	// 2. Создаем интерсептор
	cacheInterceptor := interceptors.NewRedisCacheInterceptor(rdb)

	// 3. Подписываемся на инвалидацию кеша
	if err := cacheInterceptor.Subscribe(
		"markets:list",
		"markets:invalidated",
	); err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	// 4. Создаем gRPC соединение с клиентским интерсептором
	conn, err := grpc.Dial(
		"spot-service:50051",
		grpc.WithUnaryInterceptor(
			cacheInterceptor.UnaryClientInterceptor(
				"markets:list",
				spot_pb.SpotInstrumentService_ViewMarkets_FullMethodName,
				5*time.Minute,
			),
		),
	)

// ---------- on publisher site: ----------- //

// создаем клиент редиса
redisClient := redis.NewClient(
	&redis.Options{
		Addr:     os.Getenv("REDIS_ADDRESS"),
    	Password: os.Getenv("REDIS_PASSWORD"),
	    DB:       os.Getenv("REDIS_DB"),
	},
)

func UpdateMarkets(redisClient ) {
	// ... логика обновления ...

	// Публикуем событие об обновлении
	redisClient.Publish(ctx, "markets:invalidated", "markets:list")

}

*/
