package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"docupdatesexecutor/pkg/configutil"
	"docupdatesexecutor/pkg/connector"
	"docupdatesexecutor/pkg/processor"
	"docupdatesexecutor/pkg/structures"
)

// Точка входа в программу
func main() {
	// Загружаем конфигурацию из файлов config.yaml и .env.example
	config, err := configutil.SetConfigs("config/config.yaml", "config/.env.example")
	if err != nil {
		// Завершаем работу, если не удалось загрузить конфигурацию
		log.Fatalf("Failed to load config: %v", err)
	}

	// Создаем строку подключения к базе данных на основе загруженной конфигурации
	DSN := configutil.BuildDSN(&config.Database)

	// Инициализируем соединение с базой данных
	dbConnector, err := connector.NewDBConnector(DSN)
	if err != nil {
		// Завершаем работу, если не удалось инициализировать соединение с базой данных
		log.Fatalf("Failed to initialize DB connector: %v", err)
	}

	// Инициализируем соединение с кэшем Redis
	cacheConnector := connector.NewCacheConnector(&config.Redis)
	// Инициализируем соединение с Kafka
	eventBusConnector := connector.NewEventBusConnector(&config.Kafka)
	if err != nil {
		// Завершаем работу, если не удалось инициализировать соединение с Kafka
		log.Fatalf("Failed to initialize EventBus connector: %v", err)
	}

	// Инициализируем процессор документов
	docProcessor := processor.NewDocumentProcessor(dbConnector, cacheConnector, eventBusConnector)

	// Создаем буферизированные каналы для обработки сообщений
	incomingMessages := make(chan *structures.TDocument, config.BufferSize)
	processedMessages := make(chan *structures.TDocument, config.BufferSize)

	// Создаем контекст для управления горутинами
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем фоновую синхронизацию кэша с базой данных
	cacheConnector.StartBackgroundSync(ctx, dbConnector, config.SyncInterval)

	// Используем WaitGroup для ожидания завершения всех горутин
	var wg sync.WaitGroup

	// Добавляем три горутины в WaitGroup
	wg.Add(3)
	// Запускаем горутину для потребления сообщений из Kafka
	go consumeMessages(ctx, &wg, eventBusConnector, incomingMessages)
	// Запускаем горутину для обработки сообщений
	go processMessages(ctx, &wg, docProcessor, incomingMessages, processedMessages)
	// Запускаем горутину для отправки обработанных сообщений обратно в Kafka
	go produceMessages(ctx, &wg, eventBusConnector, processedMessages)

	// Создаем канал для получения системных сигналов завершения работы (SIGINT, SIGTERM)
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	// Ожидаем получения сигнала завершения работы
	<-sigterm
	// Отменяем контекст для завершения работы горутин
	cancel()

	// Ожидаем завершения всех горутин
	wg.Wait()
}

// Функция для потребления сообщений из Kafka
func consumeMessages(ctx context.Context, wg *sync.WaitGroup, connector connector.EventBusConnector, incoming chan<- *structures.TDocument) {
	// Сообщаем WaitGroup о завершении работы горутины
	defer wg.Done()
	// Потребляем сообщения из Kafka и отправляем их в канал incoming
	connector.ConsumeMessages(ctx, incoming)
}

// Функция для обработки сообщений
func processMessages(ctx context.Context, wg *sync.WaitGroup, proc processor.Processor, incoming <-chan *structures.TDocument, processed chan<- *structures.TDocument) {
	// Сообщаем WaitGroup о завершении работы горутины
	defer wg.Done()
	// Бесконечный цикл для обработки сообщений
	for {
		select {
		case <-ctx.Done():
			// Завершаем работу, если контекст был отменен
			return
		case doc := <-incoming:
			// Обрабатываем документ
			processedDoc, err := proc.Process(doc)
			if err == nil {
				// Если обработка успешна, отправляем обработанный документ в канал processed
				processed <- processedDoc
			} else {
				// Если произошла ошибка, логируем ее
				log.Printf("Failed to process document: %v", err)
			}
		}
	}
}

// Функция для отправки обработанных сообщений обратно в Kafka
func produceMessages(ctx context.Context, wg *sync.WaitGroup, connector connector.EventBusConnector, processed <-chan *structures.TDocument) {
	// Сообщаем WaitGroup о завершении работы горутины
	defer wg.Done()
	// Отправляем обработанные сообщения в Kafka
	connector.ProduceMessages(ctx, processed)
}
