package output

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ralexstokes/relay-monitor/pkg/config"
	kafka "github.com/segmentio/kafka-go"
)

type Output struct {
	Path        string
	f           *os.File
	lock        sync.Mutex
	kConf       *config.KafkaConfig
	kafkaWriter *kafka.Writer
}

func NewFileOutput(filePath string, kafkaConfig *config.KafkaConfig) (*Output, error) {
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
		output.kafkaWriter = &kafka.Writer{
			Addr:         kafka.TCP(kafkaConfig.BootstrapServers...),
			Topic:        kafkaConfig.Topic,
			BatchBytes:   10 * 1024 * 1024, // 10MB
			BatchSize:    20,
			BatchTimeout: time.Second,
			RequiredAcks: 1,
		}
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
	if o.kafkaWriter != nil {

		ctx, cancel := context.WithTimeout(context.Background(), o.kConf.Timeout)
		defer cancel()
		err = o.kafkaWriter.WriteMessages(ctx, kafka.Message{Value: entry})
		if err != nil {
			return fmt.Errorf("could not write to kafka: %v", err)
		}
	}
	return nil
}

func (fo *Output) Close() error {
	return fo.f.Close()
}
