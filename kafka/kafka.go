package kafka

import (
	"finance-chatbot/api/logger"
	"finance-chatbot/api/worker"
	"os"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"go.uber.org/zap"
)

const (
	MessageTopic         string = "user_message"
	TransactionsJobTopic string = "save_transactions"
	GroupID              string = "ai-response-consumer"
)

var (
	MessageProducer *kafka.Producer
	WorkerPool      *worker.WorkerPool
	ResponseTopic   string = "ai_response" // THIS IS A CONSTANT NEVER CHANGE IT
)

func InitProducer() error {
	config := &kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("KAFKA_SERVER"),
		"sasl.username":     os.Getenv("KAFKA_USERNAME"),
		"sasl.password":     os.Getenv("KAFKA_PASSWORD"),
		"security.protocol": "SASL_SSL",
		"sasl.mechanism":    "PLAIN",
	}

	var err error
	MessageProducer, err = kafka.NewProducer(config)
	if err != nil {
		logger.Get().Error("failed to initialize Kafka producer",
			zap.String("bootstrap_servers", os.Getenv("KAFKA_BOOTSTRAP_SERVERS")),
			zap.Error(err))
		return err
	}

	logger.Get().Info("Kafka producer initialized successfully",
		zap.String("bootstrap_servers", os.Getenv("KAFKA_BOOTSTRAP_SERVERS")))
	return nil
}

func ProduceMessage(topic string, message []byte) error {
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          message,
	}

	err := MessageProducer.Produce(msg, nil)
	if err != nil {
		logger.Get().Error("failed to produce message",
			zap.String("topic", topic),
			zap.Error(err))
		return err
	}

	logger.Get().Debug("message produced successfully",
		zap.String("topic", topic))
	return nil
}

func StartKafkaConsumer() error {
	// Get the number of partitions for the topic
	admin, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("KAFKA_SERVER"),
		"security.protocol": "SASL_SSL",
		"sasl.mechanisms":   "PLAIN",
		"sasl.username":     os.Getenv("KAFKA_USERNAME"),
		"sasl.password":     os.Getenv("KAFKA_PASSWORD"),
	})
	if err != nil {
		logger.Get().Error("failed to create admin client", zap.Error(err))
		return err
	}
	defer admin.Close()

	metadata, err := admin.GetMetadata(&ResponseTopic, false, 10000)
	if err != nil {
		logger.Get().Error("failed to get topic metadata", zap.Error(err))
		return err
	}

	numPartitions := len(metadata.Topics[ResponseTopic].Partitions)
	logger.Get().Info("Topic partition count",
		zap.String("topic", ResponseTopic),
		zap.Int("partitions", numPartitions))

	// Initialize worker pool with number of workers matching partitions
	WorkerPool = worker.NewWorkerPool(numPartitions)
	WorkerPool.Start()

	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  os.Getenv("KAFKA_SERVER"),
		"security.protocol":  "SASL_SSL",
		"sasl.mechanisms":    "PLAIN",
		"sasl.username":      os.Getenv("KAFKA_USERNAME"),
		"sasl.password":      os.Getenv("KAFKA_PASSWORD"),
		"session.timeout.ms": "45000",
		"client.id":          "python-client-1",
		"group.id":           GroupID,
		"auto.offset.reset":  "latest",
	})
	if err != nil {
		logger.Get().Error("failed to create consumer",
			zap.String("bootstrap_servers", os.Getenv("KAFKA_BOOTSTRAP_SERVERS")),
			zap.Error(err))
		return err
	}

	err = consumer.Subscribe(ResponseTopic, nil)
	if err != nil {
		logger.Get().Error("failed to subscribe to topic",
			zap.String("topic", ResponseTopic),
			zap.Error(err))
		return err
	}

	logger.Get().Info("Kafka consumer started successfully",
		zap.String("topic", ResponseTopic),
		zap.String("group_id", GroupID),
		zap.Int("partitions", numPartitions))

	go func() {
		for {
			msg, err := consumer.ReadMessage(-1)
			if err == nil {
				logger.Get().Debug("received message",
					zap.String("topic", ResponseTopic),
					zap.String("value", string(msg.Value)),
					zap.Int32("partition", msg.TopicPartition.Partition))

				// Submit the message to the worker pool with its partition
				WorkerPool.Submit(msg.Value, msg.TopicPartition.Partition)
			} else {
				logger.Get().Error("consumer error",
					zap.String("topic", ResponseTopic),
					zap.Error(err))
			}
		}
	}()
	return nil
}
