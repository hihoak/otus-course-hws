package config

import (
	"context"
	"time"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
	"github.com/heetch/confita/backend/file"
)

// При желании конфигурацию можно вынести в internal/config.
// Организация конфига в main принуждает нас сужать API компонентов, использовать
// при их конструировании только необходимые параметры, а также уменьшает вероятность циклической зависимости.
type Config struct {
	Logger             LoggerConf    `config:"logger"`
	Database           DatabaseConf  `config:"database"`
	Server             ServerConf    `config:"server"`
	Rabbit             RabbitConf    `config:"rabbit"`
	Sender             SenderConf    `config:"sender"`
	Scheduler          SchedulerConf `config:"scheduler"`
	UseInMemoryStorage bool          `config:"use_in_memory_storage"`
}

type SenderConf struct {
	QueueToPullNotifications string `config:"sender_queue_to_pull_notifications"`
	OutputFile               string `config:"output_file"`
}

type RabbitConf struct {
	RabbitURL          string   `config:"rabbit_url"`
	ExchangesNames     []string `config:"rabbit_exchanges_names"`
	QueueNames         []string `config:"rabbit_queue_names"`
	Bindings           []Bind   `config:"rabbit_bindings"`
	ConnectionAttempts int      `config:"connection_attempts"`
}

type Bind struct {
	QueueName    string `config:"bind_queue_name"`
	Key          string `config:"bind_key"`
	ExchangeName string `config:"bind_exchange_name"`
}

type SchedulerConf struct {
	ScanPeriod                 time.Duration `config:"scanperiod"`
	CleanPeriod                time.Duration `config:"cleanperiod"`
	NotifyPeriod               time.Duration `config:"notifyperiod"`
	EventsDeprecationAgeInDays int64         `config:"eventsdeprecationageindays"`
	ExchangeToNotifyEvents     string        `config:"exchangetonotifyevents"`
}

type ServerConf struct {
	Host             string        `config:"server_host"`
	GRPCPort         string        `config:"server_grpc_port"`
	HTTPPort         string        `config:"server_http_port"`
	ReadTimeout      time.Duration `config:"server_read_timeout"`
	WriteTimeout     time.Duration `config:"server_write_timeout"`
	ShutDownTimeout  time.Duration `config:"server_shutdown_timeout"`
	GracefulShutdown time.Duration `config:"server_graceful_shutdown"`
}

type DatabaseConf struct {
	Host     string `config:"db_host"`
	Port     string `config:"db_port"`
	User     string `config:"db_user"`
	Password string `config:"db_password"`
	DBName   string `config:"db_name"`
	// почему то со снейк кейсом не работало, оставил уж так
	ConnectionTimeout time.Duration `config:"db_connection_timeout"`
	OperationTimeout  time.Duration `config:"db_operation_timeout"`
}

type LoggerConf struct {
	Level string `config:"log_level"`
}

func NewConfig(ctx context.Context, configPath string) (*Config, error) {
	loader := confita.NewLoader(file.NewBackend(configPath), env.NewBackend())
	var cfg Config
	err := loader.Load(ctx, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, err
}
