package config

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
	Logger             LoggerConf    `config:"logger"`
	Database           DatabaseConf  `config:"database"`
	Server             ServerConf    `config:"server"`
	Rabbit             RabbitConf    `config:"rabbit"`
	Sender             SenderConf    `config:"sender"`
	Scheduler          SchedulerConf `config:"scheduler"`
	UseInMemoryStorage bool          `config:"use_in_memory_storage"`
}

type SenderConf struct {
	QueueToPullNotifications string `config:"queuetopullnotifications"`
}

type RabbitConf struct {
	RabbitURL      string   `config:"rabbiturl"`
	ExchangesNames []string `config:"exchangesnames"`
	QueueNames     []string `config:"queuenames"`
	Bindings       []Bind   `config:"bindings"`
}

type Bind struct {
	QueueName    string `config:"queuename"`
	Key          string `config:"key"`
	ExchangeName string `config:"exchangename"`
}

type SchedulerConf struct {
	ScanPeriod                 time.Duration `config:"scanperiod"`
	CleanPeriod                time.Duration `config:"cleanperiod"`
	NotifyPeriod               time.Duration `config:"notifyperiod"`
	EventsDeprecationAgeInDays int64         `config:"eventsdeprecationageindays"`
	ExchangeToNotifyEvents     string        `config:"exchangetonotifyevents"`
}

type ServerConf struct {
	Host            string        `config:"host"`
	GRPCPort        string        `config:"grpcport"`
	HTTPPort        string        `config:"httpport"`
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
