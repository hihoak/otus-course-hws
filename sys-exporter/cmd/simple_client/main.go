package main

import (
	"context"
	"flag"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/config"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/logger"
	desc "github.com/hihoak/otus-course-hws/sys-exporter/pkg/api/sys-exporter"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultConfigPath = ".exporter.yaml"

	maxSizeOfData   = 1000
	truncSizeOfData = 100
)

var configPath string

func init() {
	flag.StringVar(&configPath, "config", defaultConfigPath, "path to config file")
}

func main() {
	ctx := context.Background()
	flag.Parse()
	cfg := config.New(configPath)
	logg := logger.New(cfg.Logger)

	logg.Info().Msgf("Start dialing connection with '%s'", cfg.Server.Address)
	conn, connErr := grpc.DialContext(ctx, cfg.Server.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if connErr != nil {
		logg.Fatal().Err(connErr).Msgf("failed connect to '%s'", cfg.Server.Address)
	}

	exporterClient := desc.NewExporterServiceClient(conn)
	logg.Info().Msg("starting new stream connection...")
	stream, reqErr := exporterClient.SendStreamSnapshots(ctx, &desc.SendStreamSnapshotsRequest{})
	if reqErr != nil {
		logg.Fatal().Err(reqErr).Msg("failed to establish connection to the server")
	}
	logg.Info().Msg("successfully establish connection! Check your metrics on 'localhost:7000'")
	mu := &sync.Mutex{}
	var timestamps []string
	var loadAverageFor1Min, loadAverageFor5Min, loadAverageFor15Min, cpuUsage []opts.LineData
	go func() {
		for {
			resp, respErr := stream.Recv()
			if respErr != nil {
				if errors.Is(respErr, io.EOF) {
					logg.Info().Msg("stop fetching data from server because of EOF")
					return
				}
				logg.Fatal().Err(respErr).Msg("unexpected error from server")
			}
			mu.Lock()
			if len(timestamps) > maxSizeOfData {
				timestamps = timestamps[truncSizeOfData:]
				loadAverageFor1Min = loadAverageFor1Min[truncSizeOfData:]
				loadAverageFor5Min = loadAverageFor5Min[truncSizeOfData:]
				loadAverageFor15Min = loadAverageFor15Min[truncSizeOfData:]
			}
			loadAverageFor1Min = append(loadAverageFor1Min, opts.LineData{
				Value: resp.Snapshot.LoadAverage.For1Min,
			})
			loadAverageFor5Min = append(loadAverageFor5Min, opts.LineData{
				Value: resp.Snapshot.LoadAverage.For5Min,
			})
			loadAverageFor15Min = append(loadAverageFor15Min, opts.LineData{
				Value: resp.Snapshot.LoadAverage.For15Min,
			})
			cpuUsage = append(cpuUsage, opts.LineData{
				Value: resp.Snapshot.LoadAverage.CpuUsageWin,
			})
			timestamps = append(timestamps, time.Unix(0, resp.Snapshot.Timestamp).Format("15:04:05"))
			mu.Unlock()
		}
	}()
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		render(writer, mu, loadAverageFor1Min, loadAverageFor5Min, loadAverageFor15Min, cpuUsage, timestamps)
	})
	server := http.Server{
		Addr:              "0.0.0.0:7000",
		ReadTimeout:       time.Second,
		WriteTimeout:      time.Second,
		ReadHeaderTimeout: time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		logg.Error().Err(err).Msg("server is stopped")
	}
}

func render(w http.ResponseWriter, mu *sync.Mutex, la1, la5, la15, cpuUsage []opts.LineData, timestamp []string) {
	mu.Lock()
	defer mu.Unlock()
	// create a new line instance
	line := charts.NewLine()
	// set some global options like Title/Legend/ToolTip or anything else
	line.SetGlobalOptions(
		charts.WithLegendOpts(opts.Legend{Show: true}),
		charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeInfographic}),
		charts.WithTitleOpts(opts.Title{
			Title: "System metrics",
		}))

	line.SetXAxis(timestamp).
		AddSeries("Load Average 1 min", la1).
		AddSeries("Load Average 5 min", la5).
		AddSeries("Load Average 15 min", la15).
		AddSeries("Cpu usage (windows only)", cpuUsage).
		SetSeriesOptions(charts.WithLineChartOpts(opts.LineChart{Smooth: false}))
	line.Render(w)
}
