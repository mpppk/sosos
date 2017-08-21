package etc

import "github.com/spf13/viper"

type Webhook struct {
	Name string
	Url  string
}

type Config struct {
	Webhooks []Webhook
}

func LoadConfigFromFile() (*Config, error) {
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *Config) FindWebhook(name string) (*Webhook, bool) {
	for _, webhook := range c.Webhooks {
		if webhook.Name == name {
			return &webhook, true
		}
	}
	return nil, false
}
