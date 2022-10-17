package sender

import (
	"context"
	"fmt"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/logger"
	"github.com/pkg/errors"
)

type ISender interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	Send(ctx context.Context) error
}

type Sequence interface {
	Pull(ctx context.Context, queue string) (<-chan string, error)
}

type Sender struct {
	ISender
	sequence Sequence

	log      *logger.Logger
	doneChan chan interface{}

	queueToPullNotifications string
}

func NewSender(
	ctx context.Context,
	log *logger.Logger,
	sequence Sequence,
	queueToPullNotifications string,
) *Sender {
	return &Sender{
		log:                      log,
		sequence:                 sequence,
		queueToPullNotifications: queueToPullNotifications,
	}
}

func (s Sender) Start(ctx context.Context) error {
	s.log.Info().Msg("Starting sender...")
	if err := s.Send(ctx); err != nil {
		return errors.Wrap(err, "failed to send messages")
	}
	s.log.Info().Msg("successfully stop sending notifies")
	return nil
}

func (s Sender) Stop(ctx context.Context) error {
	s.log.Info().Msg("start stopping sender")
	close(s.doneChan)
	return nil
}

func (s Sender) Send(ctx context.Context) error {
	s.log.Debug().Msgf("start consuming messages")
	messages, err := s.sequence.Pull(ctx, s.queueToPullNotifications)
	if err != nil {
		return errors.Wrap(err, "failed to start pulling, messages from queue")
	}
	for {
		select {
		case <-s.doneChan:
			s.log.Info().Msg("sender is stopped")
			return nil
		case msg, ok := <-messages:
			if !ok {
				s.log.Info().Msg("queue is closed, stopping sender")
				return nil
			}
			fmt.Println("consumed message: ", msg)
		}
	}
}
