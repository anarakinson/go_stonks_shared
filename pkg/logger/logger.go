package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Log - глобальный логгер
	Log *zap.Logger
	// SugaredLog - глобальный sugared логгер
	SugaredLog *zap.SugaredLogger
)

// Init инициализирует логгер в зависимости от среды
func Init(env string) error {
	var cfg zap.Config

	if env == "production" {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	logger, err := cfg.Build()
	if err != nil {
		return err
	}

	Log = logger
	SugaredLog = logger.Sugar()

	// Заменяем глобальные логгеры
	zap.ReplaceGlobals(logger)

	return nil
}

// Sync закрывает логгер
func Sync() error {
	if Log != nil {
		return Log.Sync()
	}
	return nil
}

/*
				Альтернативы стандартным функциям log
|-------------------+---------------------------+------------------------|
|Стандартная функция| zap эквивалент 			| zap Sugared эквивалент |
|-------------------+---------------------------+------------------------|
|log.Print()		| Log.Info()				| SugaredLog.Info()		 |
|log.Printf()		| Log.Info() с zap.String() | SugaredLog.Infof()	 |
|log.Println()		| Log.Info()				| SugaredLog.Info()		 |
|log.Fatal()		| Log.Fatal()				| SugaredLog.Fatal()	 |
|log.Fatalf()		| Log.Fatal() с zap.String()| SugaredLog.Fatalf()	 |
|log.Panic()		| Log.Panic()				| SugaredLog.Panic()	 |
|-					| Log.Debug()				| SugaredLog.Debug()	 |
|-					| Log.Warn()				| SugaredLog.Warn()		 |
|-					| Log.Error()				| SugaredLog.Error()	 |
|-------------------+---------------------------+------------------------|

*/

// Где можно добавить логирование

// Инициализация приложения (main.go):
// if err := logger.Init("development"); err != nil {
//     panic(err)
// }
// defer logger.Sync()

// HTTP-сервер:
// logger.Log.Info("Starting server",
//     zap.String("address", addr),
//     zap.String("env", env),
// )

// HTTP middleware:
// logger.Log.Debug("Request received",
//     zap.String("method", r.Method),
//     zap.String("path", r.URL.Path),
//     zap.String("ip", r.RemoteAddr),
// )

// Обработчики ошибок:
// if err != nil {
//     logger.Log.Error("Database operation failed",
//         zap.Error(err),
//         zap.String("query", query),
//     )
// }

// Критические ошибки:
// if fatalErr != nil {
//     logger.Log.Fatal("Cannot start application",
//         zap.Error(fatalErr),
//     )
// }
