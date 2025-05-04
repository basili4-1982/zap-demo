package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Определены в другом файле
//
// YOUR_CHAT_ID
// YOUR_TELEGRAM_BOT_TOKEN
//
// НЕ обходимо определить
// const (
//
//	YOUR_CHAT_ID            = -100*
//	YOUR_TELEGRAM_BOT_TOKEN = ""
//
// )
//
// Post представляет структуру данных из API JSONPlaceholder
type Post struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	UserID int    `json:"userId"`
}

// TelegramLogger отправляет логи в Telegram
type TelegramLogger struct {
	bot    *tgbotapi.BotAPI
	chatID int64
}

// NewTelegramLogger создает новый логгер для Telegram
func NewTelegramLogger(botToken string, chatID int64) (*TelegramLogger, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания бота: %w", err)
	}

	return &TelegramLogger{
		bot:    bot,
		chatID: chatID,
	}, nil
}

// Write реализует интерфейс zapcore.WriteSyncer
func (t *TelegramLogger) Write(p []byte) (n int, err error) {
	msg := tgbotapi.NewMessage(t.chatID, string(p))
	_, err = t.bot.Send(msg)
	if err != nil {
		return 0, fmt.Errorf("ошибка отправки в Telegram: %w", err)
	}
	return len(p), nil
}

// Sync для совместимости с zapcore.WriteSyncer
func (t *TelegramLogger) Sync() error {
	return nil
}

// fetchPosts получает список постов с внешнего API
func fetchPosts(ctx context.Context, logger *zap.Logger) ([]Post, error) {
	url := "https://jsonplaceholder.typicode.com/posts"

	// Настраиваем HTTP-клиент с таймаутом
	client := &http.Client{Timeout: 5 * time.Second}

	// Создаем запрос с контекстом (для отмены)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	logger.Info("Запрос к API", zap.String("url", url))

	// Выполняем запрос
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка HTTP-запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("неверный статус код: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка прочтения: %w", err)
	}

	logger.Debug("Получил ответ: ",
		zap.String("url", url),
		zap.Int("status", resp.StatusCode),
		zap.String("body", string(body)))

	// Декодируем JSON
	var posts []Post
	if err := json.NewDecoder(bytes.NewBuffer(body)).Decode(&posts); err != nil {
		return nil, fmt.Errorf("ошибка декодирования JSON: %w", err)
	}

	logger.Info("Успешно получены данные", zap.Int("количество постов", len(posts)))

	return posts, nil
}

// sortPosts сортирует посты по ID (в обратном порядке)
func sortPosts(posts []Post) []Post {
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].ID > posts[j].ID
	})
	return posts
}

func printPost(posts Post, logger *zap.Logger) {
	logger.Debug("Вывод поста", zap.Int("PostID", posts.ID), zap.Int("UserID", posts.UserID))
	fmt.Printf("title: %s, body: %s\n",
		posts.Title, posts.Body)
}

func setupLog(env string) (*zap.Logger, error) {
	// Настройка Encoder (формат логов)
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		MessageKey:     "msg",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}

	// Базовый Core для всех логов
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		zapcore.DebugLevel,
	)

	// Дополнительные выходы для production
	if env == "production" {
		fileWriter := zapcore.AddSync(&lumberjack.Logger{
			Filename: "logs/all.log",
		})

		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			fileWriter,
			zapcore.InfoLevel, // В production логируем только INFO и выше
		)

		// 1. Инициализация Telegram-логгера
		tgLogger, err := NewTelegramLogger(YOUR_TELEGRAM_BOT_TOKEN, YOUR_CHAT_ID)
		if err != nil {
			panic(err)
		}

		// 2. Настройка уровней логирования
		highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapcore.ErrorLevel // Только ERROR и FATAL
		})

		// 3. Создаем Core для Telegram
		tgCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(tgLogger),
			highPriority,
		)

		core = zapcore.NewTee(core, fileCore, tgCore)
	}

	return zap.New(core), nil
}

func main() {
	env := "production"
	if os.Getenv("ENVIRONMENT") == "development" {
		env = "development"
	}

	// Инициализация логгера
	logger, err := setupLog(env)
	if err != nil {
		panic(fmt.Sprintf("ошибка инициализации логгера: %v", err))
	}
	defer logger.Sync()

	// Контекст с таймаутом (на случай, если API не отвечает)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Запрашиваем данные с внешнего API
	posts, err := fetchPosts(ctx, logger)
	if err != nil {
		logger.Error("Ошибка при получении постов", zap.Error(err))
		return
	}

	// 2. Выполняем алгоритм (сортировку)
	sortedPosts := sortPosts(posts)
	logger.Info("Посты отсортированы", zap.Int("первый ID", sortedPosts[0].ID))

	// 3. Выводим результат (первые 3 поста)
	for i := range 3 {
		printPost(sortedPosts[i], logger)
	}
}
