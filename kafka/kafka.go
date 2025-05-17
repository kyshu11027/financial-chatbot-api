package kafka

import (
	"encoding/json"
	"finance-chatbot/api/logger"
	"finance-chatbot/api/models"
	"finance-chatbot/api/sse"
	"os"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"go.uber.org/zap"
)

var (
	MessageProducer *kafka.Producer
	MessageTopic    string = "user_message"
	ResponseTopic   string = "ai_response"
	GroupID         string = "ai-response-consumer"
)

func InitProducer() error {
	config := &kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("KAFKA_BOOTSTRAP_SERVERS"),
		"sasl.username":     os.Getenv("KAFKA_API_KEY"),
		"sasl.password":     os.Getenv("KAFKA_API_SECRET"),
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
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  os.Getenv("KAFKA_BOOTSTRAP_SERVERS"),
		"security.protocol":  "SASL_SSL",
		"sasl.mechanisms":    "PLAIN",
		"sasl.username":      os.Getenv("KAFKA_API_KEY"),
		"sasl.password":      os.Getenv("KAFKA_API_SECRET"),
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
		zap.String("group_id", GroupID))

	go func() {
		for {
			msg, err := consumer.ReadMessage(-1)
			if err == nil {
				logger.Get().Debug("received message",
					zap.String("topic", ResponseTopic),
					zap.String("value", string(msg.Value)))

				var aiResponse models.AIResponse
				chunk := string(msg.Value)
				if err := json.Unmarshal(msg.Value, &aiResponse); err != nil {
					logger.Get().Error("failed to unmarshal message",
						zap.String("topic", ResponseTopic),
						zap.Error(err))
					continue
				}

				conversationID := aiResponse.ConversationID
				logger.Get().Debug("sending chunk to client",
					zap.String("conversation_id", conversationID))
				sse.SendChunkToClient(conversationID, chunk)
			} else {
				logger.Get().Error("consumer error",
					zap.String("topic", ResponseTopic),
					zap.Error(err))
			}
		}
	}()
	return nil
}
