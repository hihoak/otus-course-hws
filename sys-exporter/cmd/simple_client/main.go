package main

import (
	"context"
	"flag"
	"io"
	"net/http"
	"time"

	"google.golang.org/grpc/credentials/insecure"

	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"

	"github.com/go-echarts/go-echarts/v2/charts"

	desc "github.com/hihoak/otus-course-hws/sys-exporter/pkg/api/sys-exporter"

	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/config"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/logger"
	"google.golang.org/grpc"
)

const (
	defaultConfigPath = ".exporter.yaml"
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
	logg.Info().Msg("successfully establish connection!")
	var timestamps []string
	var loadAverageFor1Min, loadAverageFor5Min, loadAverageFor15Min []opts.LineData
	go func() {
		for {
			resp, respErr := stream.Recv()
			if respErr != nil {
				if respErr == io.EOF {
					logg.Info().Msg("stop fetching data from server because of EOF")
					return
				}
				logg.Fatal().Err(respErr).Msg("unexpected error from server")
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
			timestamps = append(timestamps, time.Unix(0, resp.Snapshot.Timestamp).Format("15:04:05"))
		}
	}()
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		render(writer, loadAverageFor1Min, loadAverageFor5Min, loadAverageFor15Min, timestamps)
	})
	if err := http.ListenAndServe("localhost:7000", nil); err != nil {
		logg.Error().Err(err).Msg("server is stopped")
	}
}

func render(w http.ResponseWriter, la1, la5, la15 []opts.LineData, timestamp []string) {
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
		SetSeriesOptions(charts.WithLineChartOpts(opts.LineChart{Smooth: true}))
	line.Render(w)
}
