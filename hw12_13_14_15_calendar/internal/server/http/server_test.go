package internalhttp

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	server_mocks "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/server/http/mocks"
	"github.com/stretchr/testify/require"
)

func TestStart(t *testing.T) {
	t.Run("Successfully done by context", func(t *testing.T) {
		mc := gomock.NewController(t)
		s := server_mocks.NewMockServerer(mc)
		s.EXPECT().ListenAndServe().Times(1).Do(func() { time.Sleep(time.Second * 10) })
		l := server_mocks.NewMockLogger(mc)
		l.EXPECT().Info().Times(1)
		a := server_mocks.NewMockApplication(mc)
		server := Server{
			Server:          s,
			Logg:            l,
			App:             a,
			ShutdownTimeout: time.Second * 10,
		}

		errChan := make(chan error)
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			errChan <- server.Start(ctx)
			close(errChan)
		}()
		// не придумал ничего более интересного, чтобы точно был вызван метод ListenAndServe
		time.Sleep(time.Second)
		cancel()
		require.NoError(t, <-errChan)
	})

	t.Run("Failed to start server", func(t *testing.T) {
		mc := gomock.NewController(t)
		s := server_mocks.NewMockServerer(mc)
		s.EXPECT().ListenAndServe().Times(1).Return(http.ErrAbortHandler)
		l := server_mocks.NewMockLogger(mc)
		a := server_mocks.NewMockApplication(mc)
		server := Server{
			Server:          s,
			Logg:            l,
			App:             a,
			ShutdownTimeout: time.Second * 10,
		}

		errChan := make(chan error)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		go func() {
			errChan <- server.Start(ctx)
			close(errChan)
		}()

		require.ErrorIs(t, <-errChan, http.ErrAbortHandler)
	})

	t.Run("Start + Stop", func(t *testing.T) {
		mc := gomock.NewController(t)
		s := server_mocks.NewMockServerer(mc)
		// long work
		s.EXPECT().ListenAndServe().Times(1).DoAndReturn(func() { time.Sleep(time.Second * 10) }).Return(http.ErrServerClosed)
		s.EXPECT().Shutdown(gomock.Any()).Times(1).Return(http.ErrServerClosed)
		l := server_mocks.NewMockLogger(mc)
		l.EXPECT().Error().Times(0)
		// 1 - Start + 2 - Stop
		l.EXPECT().Info().Times(3)
		a := server_mocks.NewMockApplication(mc)
		server := Server{
			Server:          s,
			Logg:            l,
			App:             a,
			ShutdownTimeout: time.Second * 10,
		}

		ctx, cancel := context.WithCancel(context.Background())
		startError := make(chan error)
		go func() {
			defer close(startError)
			startError <- server.Start(ctx)
		}()

		err := server.Stop(ctx)
		// не придумал ничего более интересного, чтобы точно был вызван метод ListenAndServe
		time.Sleep(time.Second)
		cancel()

		require.ErrorIs(t, err, http.ErrServerClosed)
		require.ErrorIs(t, <-startError, nil)
	})
}
