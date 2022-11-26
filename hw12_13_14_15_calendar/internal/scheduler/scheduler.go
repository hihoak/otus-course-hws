package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	multiErr "github.com/hashicorp/go-multierror"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/logger"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage"
)

type Storage interface {
	ListEventsToNotify(ctx context.Context, fromTime time.Time, countOfEvents int) ([]*storage.Event, error)
	DeleteOldEventsBeforeTime(ctx context.Context,
		fromTime time.Time, maxLiveTime time.Duration) error
}

type Sequence interface {
	Push(ctx context.Context, exchange string, messages [][]byte) error
}

type Scheduler struct {
	storage  Storage
	sequence Sequence

	log      *logger.Logger
	doneChan chan interface{}

	exchangeToSendNotifications string
	scanPeriod                  time.Duration
	cleanPeriod                 time.Duration
	eventsDeprecationAge        time.Duration
	notifyPeriod                time.Duration
}

func NewSchedulerImpl(
	logger *logger.Logger,
	storage Storage,
	sequence Sequence,
	scanPeriod time.Duration,
	cleanPeriod time.Duration,
	eventsDeprecationAge time.Duration,
	notifyPeriod time.Duration,
	exchangeToSendNotifications string,
) *Scheduler {
	return &Scheduler{
		log:      logger,
		storage:  storage,
		sequence: sequence,

		doneChan: make(chan interface{}),

		scanPeriod:           scanPeriod,
		cleanPeriod:          cleanPeriod,
		eventsDeprecationAge: eventsDeprecationAge,
		notifyPeriod:         notifyPeriod,

		exchangeToSendNotifications: exchangeToSendNotifications,
	}
}

func (s *Scheduler) cleanOldEvents(ctx context.Context) error {
	s.log.Info().Msg("start deleting old events...")
	err := s.storage.DeleteOldEventsBeforeTime(ctx, time.Now().Local(), s.eventsDeprecationAge)
	if err != nil {
		return fmt.Errorf("failed to clean old events: %w", err)
	}
	s.log.Info().Msgf("successfully clean old events total")
	return nil
}

func (s *Scheduler) produceNotifications(ctx context.Context) error {
	s.log.Info().Msg("start sending notifications about coming events...")
	events, err := s.storage.ListEventsToNotify(ctx, time.Now().Local(), 2000)
	if err != nil {
		return fmt.Errorf("failed to get events to notify: %w", err)
	}
	s.log.Info().Msgf("successfully got events to notify total: '%d'. Now start produce it in a sequence", len(events))
	sendErr := s.sendEvents(ctx, events)
	if sendErr != nil {
		return fmt.Errorf("failed to sent notifications: %w", sendErr)
	}

	return nil
}

func (s *Scheduler) sendEvents(ctx context.Context, events []*storage.Event) error {
	messages := make([][]byte, len(events))
	var multiError *multiErr.Error
	for idx, notification := range fromEventsToNotifications(events) {
		message, marshallErr := json.Marshal(notification)
		if marshallErr != nil {
			multiError = multiErr.Append(multiError,
				fmt.Errorf("something goes wrong when trying to marshall notification: %w", marshallErr))
		}
		messages[idx] = message
	}
	if multiError != nil {
		return multiError
	}
	err := s.sequence.Push(ctx, s.exchangeToSendNotifications, messages)
	if err != nil {
		return fmt.Errorf("failed to push to channle %s notifications %d: %w",
			s.exchangeToSendNotifications, len(messages), err)
	}

	s.log.Debug().Msgf("successfully send message %d to channel %s",
		messages, s.exchangeToSendNotifications)
	return nil
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.log.Info().Msg("Starting scheduler...")
	scanTicker := time.NewTicker(s.scanPeriod)
	cleanTicker := time.NewTicker(s.cleanPeriod)
	for {
		select {
		case <-s.doneChan:
			s.log.Info().Msgf("got a signal to stop scheduling...")
			return nil
		case <-scanTicker.C:
			go func() {
				if err := s.cleanOldEvents(ctx); err != nil {
					s.log.Error().Err(err).Msgf("failed to clean old events")
				}
			}()
		case <-cleanTicker.C:
			go func() {
				if err := s.produceNotifications(ctx); err != nil {
					s.log.Error().Err(err).Msgf("failed to produce notifications")
				}
			}()
		}
	}
}

func (s *Scheduler) Stop(_ context.Context) error {
	s.log.Info().Msg("Stopping scheduler work...")
	close(s.doneChan)
	return nil
}
