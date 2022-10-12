package kafkas

type KafkaConfig struct {
	Hosts   string `mapstructure:"hosts"`
	GroupId string `mapstructure:"groupId"`
	Topics  string `mapstructure:"topics"` // ordered by priority
}
