package kafka

import (
	"log"
	"os"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

var MessageProducer *kafka.Producer
var MessageTopic string = "user_message"

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
