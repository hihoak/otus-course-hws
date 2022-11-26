package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/logger"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage"
)

type Storage interface {
	ListEventsToNotify(ctx context.Context, fromTime time.Time, countOfEvents int) ([]*storage.Event, error)
	DeleteOldEventsBeforeTime(ctx context.Context,
		fromTime time.Time, maxLiveTime time.Duration) ([]*storage.Event, error)
}

type Sequence interface {
	Push(ctx context.Context, exchange string, message []byte) error
}

type Scheduler struct {
	storage  Storage
	sequence Sequence

	log      *logger.Logger
	doneChan chan interface{}

	exchangeToSendNotifications string
	scanPeriod                  time.Duration
	eventsDeprecationAge        time.Duration
	notifyPeriod                time.Duration
}

func NewSchedulerImpl(
	logger *logger.Logger,
	storage Storage,
	sequence Sequence,
	scanPeriod time.Duration,
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
		eventsDeprecationAge: eventsDeprecationAge,
		notifyPeriod:         notifyPeriod,

		exchangeToSendNotifications: exchangeToSendNotifications,
	}
}

func (s *Scheduler) cleanOldEvents(ctx context.Context) error {
	s.log.Info().Msg("start deleting old events...")
	events, err := s.storage.DeleteOldEventsBeforeTime(ctx, time.Now().Local(), s.eventsDeprecationAge)
	if err != nil {
		return fmt.Errorf("failed to clean old events: %w", err)
	}
	s.log.Info().Msgf("successfully clean old events total: '%d'", len(events))
	return nil
}

func (s *Scheduler) produceNotifications(ctx context.Context) error {
	s.log.Info().Msg("start sending notifications about coming events...")
	events, err := s.storage.ListEventsToNotify(ctx, time.Now().Local(), 2000)
	if err != nil {
		return fmt.Errorf("failed to get events to notify: %w", err)
	}
	s.log.Info().Msgf("successfully got events to notify total: '%d'. Now start produce it in a sequence", len(events))
	var multiError error
	for _, event := range events {
		sendErr := s.sendEvent(ctx, event)
		if sendErr != nil {
			multiError = multierror.Append(multiError, sendErr)
		}
	}

	return multiError
}

func (s *Scheduler) sendEvent(ctx context.Context, event *storage.Event) error {
	notification := FromEventToNotification(event)
	message, marshallErr := json.Marshal(notification)
	if marshallErr != nil {
		return fmt.Errorf("something goes wrong when trying to marshall notification: %w", marshallErr)
	}
	err := s.sequence.Push(ctx, s.exchangeToSendNotifications, message)
	if err != nil {
		return fmt.Errorf("failed to push notification %s to channle %s: %w",
			s.exchangeToSendNotifications, message, err)
	}

	s.log.Debug().Msgf("successfully send message %s to channel %s",
		s.exchangeToSendNotifications, message)
	return nil
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.log.Info().Msg("Starting scheduler...")
	ticker := time.NewTicker(s.scanPeriod)
	for {
		select {
		case <-s.doneChan:
			s.log.Info().Msgf("got a signal to stop scheduling...")
			return nil
		case <-ticker.C:
			// TODO: add support to trigger by a signal
		}

		if err := s.cleanOldEvents(ctx); err != nil {
			s.log.Error().Err(err).Msgf("failed to clean old events")
			return err
		}

		if err := s.produceNotifications(ctx); err != nil {
			s.log.Error().Err(err).Msgf("failed to produce notifications")
			return err
		}
	}
}

func (s *Scheduler) Stop(_ context.Context) error {
	s.log.Info().Msg("Stopping scheduler work...")
	close(s.doneChan)
	return nil
}
