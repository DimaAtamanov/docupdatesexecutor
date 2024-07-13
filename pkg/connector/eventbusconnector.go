package connector

import (
	"context"
	"docupdatesexecutor/pkg/structures"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

type EventBusConnector interface {
	ConsumeMessages(ctx context.Context, messages chan<- *structures.TDocument) error
	ProduceMessages(ctx context.Context, messages <-chan *structures.TDocument) error
}

type KafkaConnector struct {
	reader *kafka.Reader
	writer *kafka.Writer
}

func NewEventBusConnector(config *structures.KafkaConfig) EventBusConnector {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: config.IncomingBrokers,
		GroupID: config.ConsumerGroup,
		Topic:   config.IncomingTopic,
	})

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: config.OutgoingBrokers,
		Topic:   config.OutgoingTopic,
	})

	return &KafkaConnector{
		reader: reader,
		writer: writer,
	}
}

func (kc *KafkaConnector) ConsumeMessages(ctx context.Context, messages chan<- *structures.TDocument) error {
	for {
		m, err := kc.reader.ReadMessage(ctx)
		if err != nil {
			return err
		}

		var doc structures.TDocument
		if err := json.Unmarshal(m.Value, &doc); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		messages <- &doc
	}
}

func (kc *KafkaConnector) ProduceMessages(ctx context.Context, messages <-chan *structures.TDocument) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case message := <-messages:
			value, err := json.Marshal(message)
			if err != nil {
				log.Printf("Failed to marshal message: %v", err)
				return err
			}
			err = kc.writer.WriteMessages(ctx, kafka.Message{
				Topic: kc.writer.Topic,
				Value: value,
			})
			if err != nil {
				log.Printf("Failed to write message: %v", err)
				return err
			}
		}
	}
}
