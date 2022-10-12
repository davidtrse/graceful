package kafkas

import (
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/gommon/log"
	k "github.com/segmentio/kafka-go"
)

// BalancersWrapper stores Balancer per topic.
// The original LeaseBytes balancer doesn't save data per topic, it's ok if the topic is set on Writer
// But if Writer is not set Topic, and is used to WriteMessage with different topic, it will override the counters.
// Which is bad design...
type BalancersWrapper struct {
	HeaderName string
	Writer     *k.Writer
	Fallback   k.Balancer
	mutex      sync.Mutex
	balancers  map[string]k.Balancer
}

func NewBalancer(Writer *k.Writer, HeaderName string, Fallback k.Balancer) k.Balancer {
	return &BalancersWrapper{
		HeaderName: HeaderName,
		Writer:     Writer,
		Fallback:   Fallback,
		balancers:  map[string]k.Balancer{},
	}
}

func (l *BalancersWrapper) Balance(msg k.Message, partitions ...int) (partition int) {
	b := l.getBalancer(msg)
	return b.Balance(msg, partitions...)
}

func (l *BalancersWrapper) getBalancer(msg k.Message) k.Balancer {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	topic := l.getTopic(msg)
	b := l.balancers[topic]
	if b == nil {
		b = &LeastSize{HeaderName: l.HeaderName, Fallback: l.Fallback}
		l.balancers[topic] = b
	}
	return b
}

func (l *BalancersWrapper) getTopic(msg k.Message) string {
	if msg.Topic != "" {
		return msg.Topic
	}
	return l.Writer.Topic
}

type LeastSize struct {
	HeaderName string
	Fallback   k.Balancer
	mutex      sync.Mutex
	counters   []leastBytesCounter
}

type leastBytesCounter struct {
	partition int
	bytes     uint64
}

// Balance satisfies the Balancer interface.
func (lb *LeastSize) Balance(msg k.Message, partitions ...int) int {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	// If there is no header, fall back balancer does the job
	sizeHeader := getHeaderUint64(msg, lb.HeaderName)
	if sizeHeader == 0 {
		return lb.Fallback.Balance(msg, partitions...)
	}

	// partitions change
	if len(partitions) != len(lb.counters) {
		lb.counters = lb.makeCounters(partitions...)
	}

	minBytes := lb.counters[0].bytes
	minIndex := 0

	for i, c := range lb.counters[1:] {
		if c.bytes < minBytes {
			minIndex = i + 1
			minBytes = c.bytes
		}
	}

	c := &lb.counters[minIndex]
	c.bytes += sizeHeader
	log.Debugf("topic: %s, size: %d, partition: %d", msg.Topic, sizeHeader, c.partition)
	return c.partition
}

func getHeaderUint64(msg k.Message, headerName string) uint64 {
	for _, h := range msg.Headers {
		if headerName == h.Key {
			res, err := strconv.ParseUint(string(h.Value), 10, 64)
			if err != nil {
				return 0
			}
			return res
		}
	}
	return 0
}

func (lb *LeastSize) makeCounters(partitions ...int) (counters []leastBytesCounter) {
	length := len(partitions)
	counters = make([]leastBytesCounter, length)

	randomRange := int64(7*length - 1)
	r := rand.New(rand.NewSource(time.Now().UnixMilli()))
	for i, p := range partitions {
		counters[i].partition = p
		// init with a random size, to avoid all uploader nodes pick partitions in sequence at the very beginning.
		counters[i].bytes = uint64(r.Int63n(randomRange))
	}

	sort.Slice(counters, func(i int, j int) bool {
		return counters[i].partition < counters[j].partition
	})
	return
}

func (lb *LeastSize) Info() {
	for _, c := range lb.counters {
		log.Infof("%d, %d", c.partition, c.bytes)
	}
}
