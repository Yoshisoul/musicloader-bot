package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	TelegramToken string

	Messages Messages
}

type Messages struct {
	Errors
	Resopnses
}

type Errors struct {
	Internal       string `mapstructure:"internal"`
	UnknownCommand string `mapstructure:"unknown_command"`
	InvalidLink    string `mapstructure:"invalid_link"`
	MakeChoice     string `mapstructure:"make_choice"`
	NoFormat       string `mapstructure:"noFormat"`
}

type Resopnses struct {
	Start    string `mapstructure:"start"`
	Cancel   string `mapstructure:"cancel"`
	Download string `mapstructure:"download"`
	Timeout  string `mapstructure:"timeout"`
	Send     string `mapstructure:"send"`
	NoChoice string `mapstructure:"no_choice"`
	Quality  string `mapstructure:"quality"`
}

func Init() (*Config, error) {
	viper.AddConfigPath("configs")
	viper.SetConfigName("main")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	if err := viper.UnmarshalKey("messages.responses", &cfg.Messages.Resopnses); err != nil {
		return nil, err
	}

	if err := viper.UnmarshalKey("messages.errors", &cfg.Messages.Errors); err != nil {
		return nil, err
	}

	if err := parseEnv(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func parseEnv(cfg *Config) error {
	if err := viper.BindEnv("token"); err != nil {
		return err
	}

	cfg.TelegramToken = viper.GetString("token")

	return nil
}
