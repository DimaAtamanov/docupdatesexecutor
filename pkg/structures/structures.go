package structures

import (
	"time"
)

type TDocument struct {
	Url            string
	PubDate        uint64
	FetchTime      uint64
	Text           string
	FirstFetchTime uint64
}

type Config struct {
	Database     DatabaseConfig `yaml:"database"`
	Redis        RedisConfig    `yaml:"redis"`
	Kafka        KafkaConfig    `yaml:"kafka"`
	SyncInterval time.Duration  `yaml:"syncInterval"`
	BufferSize   int            `yaml:"bufferSize"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	User     string
	Password string
	Dbname   string `yaml:"dbname"`
	Port     string `yaml:"port"`
	Sslmode  string `yaml:"sslmode"`
	TimeZone string `yaml:"TimeZone"`
}

type RedisConfig struct {
	Addr                 string `yaml:"addr"`
	User                 string
	Password             string
	DB                   int    `yaml:"db"`
	MaxMemory            string `yaml:"maxmemory"`
	MaxMemoryPolicy      string `yaml:"maxmemory_policy"`
	LazyfreeLazyEviction string `yaml:"lazyfree_lazy_eviction"`
}

type KafkaConfig struct {
	IncomingBrokers []string `yaml:"incomingBrokers"`
	OutgoingBrokers []string `yaml:"outgoingBrokers"`
	ConsumerGroup   string   `yaml:"consumerGroup"`
	IncomingTopic   string   `yaml:"incomingTopic"`
	OutgoingTopic   string   `yaml:"outgoingTopic"`
}
