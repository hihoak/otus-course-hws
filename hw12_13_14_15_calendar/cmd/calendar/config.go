package main

import (
	"context"
	"time"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/file"
)

// При желании конфигурацию можно вынести в internal/config.
// Организация конфига в main принуждает нас сужать API компонентов, использовать
// при их конструировании только необходимые параметры, а также уменьшает вероятность циклической зависимости.
type Config struct {
	Logger             LoggerConf   `config:"logger"`
	Database           DatabaseConf `config:"database"`
	Server             ServerConf   `config:"server"`
	UseInMemoryStorage bool         `config:"use_in_memory_storage"`
}

type ServerConf struct {
	Host            string
	Port            string
	ReadTimeout     time.Duration `config:"readtimeout"`
	WriteTimeout    time.Duration `config:"writetimeout"`
	ShutDownTimeout time.Duration `config:"shutdowntimeout"`
}

type DatabaseConf struct {
	Host     string `config:"Host"`
	Port     string `config:"port"`
	User     string `config:"user"`
	Password string `config:"password"`
	DBName   string `config:"dbname"`
	// почему то со снейк кейсом не работало, оставил уж так
	ConnectionTimeout time.Duration `config:"connectiontimeout"`
	OperationTimeout  time.Duration `config:"operationtimeout"`
}

type LoggerConf struct {
	Level string `config:"level"`
}

func NewConfig(ctx context.Context, configPath string) (*Config, error) {
	loader := confita.NewLoader(file.NewBackend(configPath))
	var cfg Config
	err := loader.Load(ctx, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, err
}
