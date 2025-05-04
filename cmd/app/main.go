package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"go.uber.org/zap"
)

// Post представляет структуру данных из API JSONPlaceholder
type Post struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	UserID int    `json:"userId"`
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

func main() {
	//TODO: разделить создание логера для prod и dev окружения
	// Инициализация логгера
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("ошибка инициализации логгера: %v", err))
	}
	defer logger.Sync()

	// Контекст с таймаутом (на случай, если API не отвечает)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//TODO: прячем логгер в контекст
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
