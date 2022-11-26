package sender

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/logger"
)

type Storager interface {
	SetSentStatusToEvents(ctx context.Context, ids []string) error
}

type Sequence interface {
	Pull(ctx context.Context, queue string) (<-chan []byte, error)
}

type Sender struct {
	storage  Storager
	sequence Sequence

	log      *logger.Logger
	doneChan chan interface{}

	queueToPullNotifications string
}

type Notification struct {
	EventID string `json:"eventId"`
}

func NewSender(
	_ context.Context,
	log *logger.Logger,
	sequence Sequence,
	storage Storager,
	queueToPullNotifications string,
) *Sender {
	return &Sender{
		log:                      log,
		sequence:                 sequence,
		storage:                  storage,
		queueToPullNotifications: queueToPullNotifications,
	}
}

func (s *Sender) Start(ctx context.Context) error {
	s.log.Info().Msg("Starting sender...")
	if err := s.send(ctx); err != nil {
		return fmt.Errorf("failed to send messages: %w", err)
	}
	s.log.Info().Msg("successfully stop sending notifies")
	return nil
}

func (s *Sender) Stop(_ context.Context) error {
	s.log.Info().Msg("start stopping sender")
	close(s.doneChan)
	return nil
}

func (s *Sender) send(ctx context.Context) error {
	s.log.Debug().Msgf("start consuming messages")
	messages, err := s.sequence.Pull(ctx, s.queueToPullNotifications)
	if err != nil {
		return fmt.Errorf("failed to start pulling, messages from queue: %w", err)
	}
	successIDs := make(chan string)
	go s.setStatuses(ctx, successIDs)
	defer close(successIDs)
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
			notification := &Notification{}
			if marshErr := json.Unmarshal(msg, &notification); marshErr != nil {
				s.log.Error().Err(marshErr).Msgf("Sender: failed to unmarshall to notification: %s", msg)
				continue
			}
			fmt.Println("consumed message: ", msg)
			successIDs <- notification.EventID
		}
	}
}

func (s *Sender) setStatuses(ctx context.Context, successSentIDs <-chan string) {
	flushTicker := time.NewTicker(time.Second * 2)
	idsToFlush := make([]string, 0, 100)
	for {
		select {
		case id, ok := <-successSentIDs:
			if !ok {
				s.log.Debug().Msg("Sender: stop flushing channel is closed")
				return
			}
			idsToFlush = append(idsToFlush, id)
			if len(idsToFlush) == 100 {
				if err := s.storage.SetSentStatusToEvents(ctx, idsToFlush); err != nil {
					s.log.Error().Err(err).Msg("Sender: failed to set status sent to events")
					// TODO: сохранять такие ивенты в какое-то хранилище в какую-нибудь кафку или тот
					// же рэббит, чтобы потом еще раз подтвердить им отправку
				}
				idsToFlush = idsToFlush[:0]
			}
		case <-flushTicker.C:
			if err := s.storage.SetSentStatusToEvents(ctx, idsToFlush); err != nil {
				s.log.Error().Err(err).Msg("Sender: failed to set status not sent to events")
				// TODO: сохранять такие ивенты в какое-то хранилище в какую-нибудь кафку или тот
				// же рэббит, чтобы потом еще раз подтвердить им отправку
			}
			idsToFlush = idsToFlush[:0]
		}
	}
}
