package interceptors

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

type RedisCacheInterceptor struct {
	redis  *redis.Client
	pubSub *redis.PubSub
}

// создает новый интерсептор на основе клиента редис
func NewInterceptor(redis *redis.Client) *RedisCacheInterceptor {
	return &RedisCacheInterceptor{
		redis: redis,
	}
}

// Subscribe - подписывается на событие инвалидации
// принимает ключ, который будет инвалидироваться, и ключ, по которому срабатывает инвалидация
func (i *RedisCacheInterceptor) Subscribe(cacheKey, invalidationKey string) {

	// Подписываемся на события инвалидации
	i.pubSub = i.redis.Subscribe(context.Background(), invalidationKey)
	go i.listenForInvalidations(cacheKey)

}

// listenForInvalidations - слушает события инвалидации
func (i *RedisCacheInterceptor) listenForInvalidations(cacheKey string) {
	ch := i.pubSub.Channel()
	for msg := range ch {
		if msg.Payload == cacheKey {
			i.redis.Del(context.Background(), cacheKey)
		}
	}
}

// Unary - создает непосредственно интерсептор, кеширующий данные, возвращаемые запросом
// метод принимает ключ редиса, который кешируется, метод(запрос), на котором срабатывает, и время жизни кеша
func (i *RedisCacheInterceptor) Unary(cacheKey, methodName string, ttl time.Duration) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// кешируем только указанный метод
		if info.FullMethod != methodName {
			return handler(ctx, req)
		}

		// Пытаемся получить из кеша
		if cached, err := i.redis.Get(ctx, cacheKey).Bytes(); err == nil {
			var response interface{}
			if err := json.Unmarshal(cached, &response); err == nil {
				return response, nil
			}
		}

		// Вызываем оригинальный метод
		resp, err := handler(ctx, req)
		if err != nil {
			return nil, err
		}

		// Сохраняем в кеш
		if data, err := json.Marshal(resp); err == nil {
			i.redis.Set(ctx, cacheKey, data, ttl)
		}

		return resp, nil
	}
}

// ------------------------------------------ //
// ---------------- example ----------------- //

// ---------- on subscriber site: ----------- //

/*

// создаем клиент редиса
redisClient := redis.NewClient(
	&redis.Options{
		Addr:     os.Getenv("REDIS_ADDRESS"),
    	Password: os.Getenv("REDIS_PASSWORD"),
	    DB:       os.Getenv("REDIS_DB"),
	},
)

// Создаем интерсептор
cacheInterceptor := interceptors.NewRedisCacheInterceptor(redisClient)
cacheInterceptor.Subscribe("markets:list", "markets:invalidated")

// Регистрируем интерсептор в сервере
server := grpc.NewServer(
    grpc.UnaryInterceptor(cacheInterceptor.Unary("markets:list", spot_inst_pb.SpotInstrumentService_ViewMarkets_FullMethodName), 5*time.Minute),
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
