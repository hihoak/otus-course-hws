package snapshots

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	datastructures "github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/data-structures"

	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/config"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/logger"
)

func TestSnapshots(t *testing.T) {
	ctx := context.Background()
	logg := logger.New(config.LoggerSection{LogLevel: "debug"})

	cases := []struct {
		name            string
		datas           []*datastructures.SysData
		expectedResults *datastructures.SysData
	}{
		{
			name: "give 1 return 1",
			datas: []*datastructures.SysData{
				{
					LoadAverage: &datastructures.LoadAverage{
						For1Min:  1.5,
						For5min:  2.0,
						For15min: 3.0,
					},
				},
			},
			expectedResults: &datastructures.SysData{
				LoadAverage: &datastructures.LoadAverage{
					For1Min:  1.5,
					For5min:  2.0,
					For15min: 3.0,
				},
			},
		},
		{
			name: "give 3 return 1",
			datas: []*datastructures.SysData{
				{
					LoadAverage: &datastructures.LoadAverage{
						For1Min:  1.5,
						For5min:  2.0,
						For15min: 3.0,
					},
				},
				{
					LoadAverage: &datastructures.LoadAverage{
						For1Min:  1.0,
						For5min:  3.0,
						For15min: 4.0,
					},
				},
				{
					LoadAverage: &datastructures.LoadAverage{
						For1Min:  3.5,
						For5min:  10.0,
						For15min: 33.0,
					},
				},
			},
			expectedResults: &datastructures.SysData{
				LoadAverage: &datastructures.LoadAverage{
					For1Min:  2.0,
					For5min:  5.0,
					For15min: 13.333333,
				},
			},
		},
	}

	for _, tc := range cases {
		snapsh := New(ctx, logg, config.SnapshotsSection{})

		for _, d := range tc.datas {
			snapsh.Push(ctx, d)
		}
		res := snapsh.calculateSnapshot()
		require.InDelta(t, tc.expectedResults.LoadAverage.For1Min, res.LoadAverage.For1Min, 0.0001)
		require.InDelta(t, tc.expectedResults.LoadAverage.For5min, res.LoadAverage.For5min, 0.0001)
		require.InDelta(t, tc.expectedResults.LoadAverage.For15min, res.LoadAverage.For15min, 0.0001)
	}
}

func TestMultiSnapshots(t *testing.T) {
	ctx := context.Background()
	logg := logger.New(config.LoggerSection{LogLevel: "debug"})

	cases := []struct {
		name            string
		datas           [][]*datastructures.SysData
		expectedResults []*datastructures.SysData
	}{
		{
			name: "give 1 return snapshot, give 3 return snapshot",
			datas: [][]*datastructures.SysData{
				{
					{
						LoadAverage: &datastructures.LoadAverage{
							For1Min:  1.2,
							For5min:  2.3,
							For15min: 3.3,
						},
					},
				},
				{
					{
						LoadAverage: &datastructures.LoadAverage{
							For1Min:  1.6,
							For5min:  2.7,
							For15min: 3.8,
						},
					},
					{
						LoadAverage: &datastructures.LoadAverage{
							For1Min:  10.2,
							For5min:  19.3,
							For15min: 22.3,
						},
					},
					{
						LoadAverage: &datastructures.LoadAverage{
							For1Min:  4.5,
							For5min:  7.7,
							For15min: 6.8,
						},
					},
				},
			},
			expectedResults: []*datastructures.SysData{
				{
					LoadAverage: &datastructures.LoadAverage{
						For1Min:  1.2,
						For5min:  2.3,
						For15min: 3.3,
					},
				},
				{
					LoadAverage: &datastructures.LoadAverage{
						For1Min:  5.433333,
						For5min:  9.9,
						For15min: 10.96666,
					},
				},
			},
		},
	}

	for _, tc := range cases {
		snapsh := New(ctx, logg, config.SnapshotsSection{})

		for idx, datas := range tc.datas {
			for _, d := range datas {
				snapsh.Push(ctx, d)
			}
			res := snapsh.calculateSnapshot()
			require.InDelta(t, tc.expectedResults[idx].LoadAverage.For1Min, res.LoadAverage.For1Min, 0.0001)
			require.InDelta(t, tc.expectedResults[idx].LoadAverage.For5min, res.LoadAverage.For5min, 0.0001)
			require.InDelta(t, tc.expectedResults[idx].LoadAverage.For15min, res.LoadAverage.For15min, 0.0001)
		}
	}
}
