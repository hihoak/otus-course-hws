package config

import (
	"log"
	"time"

	"github.com/cristalhq/aconfig"
	"github.com/cristalhq/aconfig/aconfigyaml"
)

type ExporterSection struct {
	// how often exporter take metrics from system
	ScrapeInterval time.Duration `default:"1s" env:"SCRAPE_INTERVAL"`

	// scrapes channel storage buffer
	DataChannelBuffer int `default:"1000" env:"DATA_CHANNEL_BUFFER"`

	// time for we be sure that all data what we collect will be saved and sended
	GracefullyShutdownTimeout time.Duration `default:"3s" env:"GRACEFULLY_SHUTDOWN_TIMEOUT"`
}

type CollectorSection struct {
	DisableMetrics DisableMetrics
}

type DisableMetrics struct {
	LoadAverage bool `default:"false" env:"LOAD_AVERAGE"`
}

type SnapshotsSection struct {
	// initial time on start of exporter when it's collecting metrics for first snapshot
	WarmupInterval time.Duration `default:"5s" env:"WARMUP_INTERVAL"`
	// interval of sending snapshots
	SnapshotInterval time.Duration `default:"2s" env:"SNAPSHOT_INTERVAL"`
}

type LoggerSection struct {
	LogLevel string `default:"info" env:"LOG_LEVEL"`
}

type MemoryStorageSection struct {
	// path where exporter will be store snapshots files
	SnapshotsStoragePath string `default:"/tmp/sys-exporter" env:"SNAPSHOTS_STORAGE_PATH"`
	// maximum size of snapshot file in bytes, when new will be created
	MaximumSizeOfSnapshotFile int64 `default:"1000000" env:"MAXIMUM_SIZE_OF_SNAPSHOT_FILE"`
}

type ServerSection struct {
	Address string `default:"127.0.0.1:8000" env:"ADDRESS"`
}

type Config struct {
	Logger        LoggerSection
	Exporter      ExporterSection
	MemoryStorage MemoryStorageSection
	Snapshots     SnapshotsSection
	Server        ServerSection
	Collector     CollectorSection
}

func New(configPath string) *Config {
	var cfg Config
	loader := aconfig.LoaderFor(&cfg, aconfig.Config{
		SkipFlags: true,

		FailOnFileNotFound: true,

		EnvPrefix: "EXPORTER",
		Files:     []string{configPath},
		FileDecoders: map[string]aconfig.FileDecoder{
			".yaml": aconfigyaml.New(),
		},
	})

	if loadErr := loader.Load(); loadErr != nil {
		log.Fatal("failed to load configuration", loadErr)
	}

	return &cfg
}
