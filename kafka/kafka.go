package kafka

import (
	"encoding/json"
	"finance-chatbot/api/models"
	"finance-chatbot/api/sse"
	"log"
	"os"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

var MessageProducer *kafka.Producer
var MessageTopic string = "user_message"
var ResponseTopic string = "ai_response"
var GroupID string = "ai-response-consumer"

func InitProducer() error {
	config := &kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("KAFKA_BOOTSTRAP_SERVERS"), // e.g., "your-cluster-name.us-west-2.aws.confluent.cloud:9092"
		"sasl.username":     os.Getenv("KAFKA_API_KEY"),           // Your API key
		"sasl.password":     os.Getenv("KAFKA_API_SECRET"),        // Your API secret
		"security.protocol": "SASL_SSL",
		"sasl.mechanism":    "PLAIN",
	}

	var err error
	MessageProducer, err = kafka.NewProducer(config)
	if err != nil {
		log.Printf("Failure initializing Kafka producer with bootstrap servers: %s", os.Getenv("KAFKA_BOOTSTRAP_SERVERS"))
		return err
	}
	return nil
}

func ProduceMessage(topic string, message []byte) error {
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          message,
	}

	err := MessageProducer.Produce(msg, nil)
	if err != nil {
		log.Printf("Failed to produce message: %s", err)
		return err
	}
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
		log.Fatalf("Failed to create consumer: %v", err)
		return err
	}

	err = consumer.Subscribe(ResponseTopic, nil)
	if err != nil {
		log.Fatalf("Failed to subscribe to topic: %v", err)
		return err
	}

	go func() {
		for {
			msg, err := consumer.ReadMessage(-1)
			if err == nil {
				// Log when a message is received
				log.Printf("Received message from ai_response topic: %s", string(msg.Value))

				// Assume the message key is requestID, value is the chunk
				var aiResponse models.AIResponse
				chunk := string(msg.Value)
				if err := json.Unmarshal(msg.Value, &aiResponse); err != nil {
					log.Printf("Failed to unmarshal message to AIResponse: %v", err)
					continue
				}
				conversationID := aiResponse.ConversationID
				sse.SendChunkToClient(conversationID, chunk)
			} else {
				log.Printf("Consumer error: %v", err)
			}
		}
	}()
	return nil
}
