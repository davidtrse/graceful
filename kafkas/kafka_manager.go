package kafkas

import (
	"context"
	"errors"
	"fmt"
	"io"

	"runtime/debug"
	"strings"
	"time"

	"github.com/labstack/gommon/log"
	"github.com/segmentio/kafka-go"
)

type IKafkaManager interface {
	CreateReader()
	CreateWriter()
	ReadMessage(topic string) (kafka.Message, error)
	WriteMessage(topic string, key []byte, value []byte) error
	WriteMessageWithHeader(topic string, key []byte, value []byte, headerName string, headerValue string) error
	StopReadMessage()
	StopWriteMessage()
}

func IsEmptyMessage(msg kafka.Message) bool {
	return msg.Key == nil
}

func IsNotEmpty(msg kafka.Message) bool {
	return msg.Key != nil
}

type KafkaManager struct {
	Config     *KafkaConfig
	Context    context.Context
	CancelFunc context.CancelFunc
	Brokers    []string
	Topics     []string
	GroupId    string
	Readers    map[string]*kafka.Reader
	Writer     *kafka.Writer
}

var (
	kafkaManagers map[string]*KafkaManager
	err           error
)

func GetKafkaManager(topic string) *KafkaManager {
	return kafkaManagers[topic]
}

func (this *KafkaManager) loadKafkaConfig() error {

	this.Topics = strings.Split(this.Config.Topics, ",")
	for i, v := range this.Topics {
		this.Topics[i] = strings.ToLower(strings.TrimSpace(v))
	}

	this.Brokers = strings.Split(this.Config.Hosts, ",")

	this.GroupId = strings.ToLower(strings.TrimSpace(this.Config.GroupId))

	return nil

}

func NewKafkaManager(kConfig *KafkaConfig) (*KafkaManager, error) {
	km := &KafkaManager{Config: kConfig}
	_ = km.loadKafkaConfig()
	if err != nil {
		log.Errorf("NewKafkaManager: Failed to load kafka config, err: %v", err)
		return nil, err
	}
	if len(km.Topics) == 0 {
		log.Errorf("NewKafkaManager: No topics in configuration file.")
		return nil, errors.New("No topics")
	}
	ctx, cancel := context.WithCancel(context.Background())
	km.Context = ctx
	km.CancelFunc = cancel
	//km.createReaderWriters()dad

	return km, nil
}

func (this *KafkaManager) CreateReader() {
	this.Readers = make(map[string]*kafka.Reader)
	for _, topic := range this.Topics {
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:  this.Brokers,
			GroupID:  this.GroupId,
			Topic:    topic,
			MinBytes: 10e3, // 10KB
			MaxBytes: 10e6, // 10MB
			//Logger:      kafka.LoggerFunc(logf),
			ErrorLogger:           kafka.LoggerFunc(logf),
			WatchPartitionChanges: true,
		})
		this.Readers[topic] = r
	}
}

func (this *KafkaManager) CreateWriter() {
	//w := &kafka.Writer{
	//	Addr:     kafka.TCP(sb.String()),
	//	Topic: 	  topic,
	//	Balancer: &kafka.LeastSize{},
	//}
	// the Writer() only accepts one ip address
	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers: this.Brokers,
		//Topic:   topic,  // set topic when writing a message
		//Logger:      kafka.LoggerFunc(logf),
		ErrorLogger:  kafka.LoggerFunc(logf),
		RequiredAcks: -1,
	})
	b := NewBalancer(w, "size", &kafka.Hash{})
	w.Balancer = b
	this.Writer = w
}

// If topic is empty, then read from high to medium to low
// if topic is not empty, then read only that topic
func (this *KafkaManager) ReadMessage(topic string) (kafka.Message, error) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("ReadMessageByPriority caught panic: %v, stack trace: %s", err, string(debug.Stack()))
		}
	}()

	if topic != "" {
		msg, err := this.readMessageWithTimeout(this.Context, this.Readers[topic], time.Second*1)
		if err == nil {
			fmt.Printf("ReadMessageByPriority: Got message from topic: %s \n", topic)
			return msg, nil
		} else {
			fmt.Printf("ReadMessageByPriority: no message on topic: %s \n", topic)
			return kafka.Message{}, nil
		}
	} else {
		for _, topic := range this.Topics {
			msg, err := this.readMessageWithTimeout(this.Context, this.Readers[topic], time.Second*1)
			fmt.Println("Readed more message")
			if err == nil {
				fmt.Printf("ReadMessageByPriority: Got message from topic: %s \n", topic)
				return msg, nil
			} else {
				fmt.Printf("ReadMessageByPriority: no message\n")
				fmt.Printf("ReadMessageByPriority:err=%s\n", err.Error())
				if err == io.EOF {
					return kafka.Message{}, io.EOF
				}
			}
		}
		return kafka.Message{}, nil
	}
}

func (this *KafkaManager) WriteMessage(topic string, key []byte, value []byte) error {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("WriteMessage caught panic: %v, stack trace: %s", err, string(debug.Stack()))
		}
	}()
	err := this.Writer.WriteMessages(this.Context, kafka.Message{Topic: topic, Key: key, Value: value})
	return err
}

func (this *KafkaManager) WriteMessageWithHeader(topic string, key []byte, value []byte, headerName string, headerValue string) error {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("WriteMessage caught panic: %v, stack trace: %s", err, string(debug.Stack()))
		}
	}()
	h := kafka.Header{Key: headerName, Value: []byte(headerValue)}
	err := this.Writer.WriteMessages(this.Context, kafka.Message{Topic: topic, Key: key, Value: value, Headers: []kafka.Header{h}})
	return err
}

func (this *KafkaManager) StopWriteMessage() {
	if this.Writer != nil {
		fmt.Println("Closing writer...")
		this.Writer.Close()
		this.Writer = nil
	}
}

func (this *KafkaManager) StopReadMessage() {
	if this.Readers != nil {
		for _, r := range this.Readers {
			fmt.Println("Closing reader...")
			r.Close()
		}
		this.Readers = nil
	}
}
func logf(msg string, a ...interface{}) {
	log.Infof("kafka: "+msg, a...)
}

func (this *KafkaManager) readMessageWithTimeout(ctx context.Context, reader *kafka.Reader, timeout time.Duration) (kafka.Message, error) {
	msg, err := reader.ReadMessage(ctx)
	return msg, err
}
