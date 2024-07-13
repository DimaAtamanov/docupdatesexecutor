# DocUpdatesExecutor

## Описание проекта

`DocUpdatesExecutor` — это сервис, предназначенный для обработки обновлений документов, поступающих из очереди сообщений (Kafka). Сервис обеспечивает обновление и поддержание актуальной информации о документах, используя кеширование в Redis и хранение данных в PostgreSQL. Основные задачи сервиса включают в себя чтение сообщений из Kafka, обработку этих сообщений, синхронизацию данных между Redis и PostgreSQL, и запись обработанных сообщений обратно в Kafka.

## Структура проекта

```
docupdatesexecutor/
├── cmd/
│   ├── mainapp/
│   │   └── main.go
├── configs/
│   └── config.yaml
│   └── .env.example
├── pkg/
│   ├── configutil/
│   │   └── configsetup.go
│   ├── connector/
│   │   └── cacheconnector.go
│   │   └── dbconnector.go
│   │   └── eventbusconnector.go
│   ├── processor/
│   │   └── processor.go
│   ├── structures/
│   │   └── structures.go
├── go.mod
├── go.sum
```

## Основные компоненты

### cmd/mainapp/main.go
Главный файл приложения, инициализирующий все компоненты и запускающий основную логику работы сервиса.

### pkg/processor/processor.go
Содержит логику обработки документов.

### pkg/configutil/
Пакет для парсинга конфигураций:
- `configsetup.go` — парсинг `config.yaml` и `.env.example`.

### pkg/connector/
Пакет содержит коннекторы для взаимодействия с Redis, PostgreSQL и Kafka. Под коннектором в данном случае имеется ввиду интерфейс для взаимодейтсвия с соответствующей частью сервиса.
- `cacheconnector.go` — взаимодействие с Redis.
- `dbconnector.go` — взаимодействие с PostgreSQL.
- `eventbusconnector.go` — взаимодействие с Kafka.

### pkg/structures/
Пакет содержит описание структур, используемых в работе сервиса.

### configs/
Содержит файлы конфигураций:
- `config.yaml` — основные конфигурации.
- `.env.example` — логины и пароли.


## Основная логика работы сервиса

1. **Загрузка конфигурации**:
   - Загрузка конфигурации из файлов `config.yaml` и `.env.example`.

2. **Инициализация коннекторов**:
   - Инициализация коннекторов для работы с PostgreSQL, Redis и Kafka.

3. **Создание и запуск горутин**:
   - Создание трех групп горутин:
     - Чтение сообщений из Kafka и запись в канал входящих сообщений.
     - Обработка сообщений из канала входящих сообщений и запись в канал обработанных сообщений.
     - Запись обработанных сообщений из канала обработанных сообщений обратно в Kafka.

4. **Синхронизация данных**:
   - Периодическая синхронизация данных из Redis в PostgreSQL с использованием механизма снепшотов RDB.

## Масштабируемость
В рамках одной машины масштабировать систему можно с помощью управления количеством горутин. Однако, следует учитывать, что Redis не поддерживает многопоточность. Это может стать узким местом в программе. В таком случе следует либо писать систему кэширования с нуля или подобрать key-value хранилище, которое поддерживает многопоточность с поддержкой golang.

Для горизонтального масштабирования на кластере удобно применить следующий подход:

1. Развернуть данный сервис на требуемом количестве машин.
2. Сервис формирующий сообщения в Kafka должен иметь возможность для каждого url назначать уникальный идентификатор партиции Kafka. При чем, когда документ создается впервые его url равновероятным образом ставится в соответствие идентификатор партиции и далее все обновления этого документа отправляются в одну и туже партицию.
3. Каждый экземпляр DocUpdatesExecutor будет считывать сообщения из своей партиции по идентификатору заданному в kafka.Reader. Таким образом не будет требоваться синхронизация между разными машинами.
4. При этом на каждую партицию следует назначить несколько одинаковых DocUpdatesExecutor, которые должны получать одно и тоже сообщение. Это позволит повысить отказоустойчивость системы.

Следует также оптимизировать процесс синхронизации кэша с БД.