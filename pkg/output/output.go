package output

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/ralexstokes/relay-monitor/pkg/config"
	"go.uber.org/zap"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type Output struct {
	Path          string
	f             *os.File
	lock          sync.Mutex
	kConf         *config.KafkaConfig
	ctx           context.Context
	producer *kafka.Producer
	deliveryCh    chan kafka.Event
}

func NewFileOutput(ctx context.Context, filePath string, kafkaConfig *config.KafkaConfig) (*Output, error) {
	// check and prepare file
	err := CheckFile(filePath)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}

	output := &Output{
		Path:  filePath,
		f:     f,
		kConf: kafkaConfig,
	}

	// check and prepare kafka producer
	if kafkaConfig != nil {
		lp, err := kafka.NewProducer(&kafka.ConfigMap{
			"bootstrap.servers":  strings.Join(kafkaConfig.BootstrapServers, ","),
			"message.max.bytes":  10 * 1024 * 1024, // 10MB
			"client.id":          "relay-monitor",
			"enable.idempotence": true,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize librdkafka producer: %w", err)
		}
		output.producer = lp
		go output.consumeEvents()
	}

	return output, nil
}

func (o *Output) WriteEntry(entry []byte) error {
	o.lock.Lock()
	defer o.lock.Unlock()
	_, err := o.f.Write(append(entry, byte('\n')))
	if err != nil {
		return err
	}
	if o.producer != nil {
		err := o.producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &o.kConf.Topic,
				Partition: kafka.PartitionAny,
			},
			Value: entry,
		}, nil)
		if err != nil {
			return fmt.Errorf("could not write to kafka: %v", err)
		}
	}
	return nil
}

func (o *Output) consumeEvents() {
	for {
		select {
		case <-o.ctx.Done():
			return
		case e := <-o.producer.Events():
			switch e := e.(type) {
			case *kafka.Message:
				if e.TopicPartition.Error != nil {
					zap.S().Errorw("encountered unexpected error producing a message", zap.Error(e.TopicPartition.Error))
				}
			case kafka.Error:
				if e.IsFatal() {
					zap.S().Fatalw("kafka producer encountered fatal error", zap.Error(e))
				} else {
					zap.S().Errorw("kafka producer encountered an error", "error", e)
				}
			}
		}

	}
}

func (fo *Output) Close() error {
	return fo.f.Close()
}
