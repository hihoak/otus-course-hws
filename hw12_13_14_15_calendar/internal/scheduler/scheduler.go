package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/logger"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage"
	"github.com/pkg/errors"
)

type IScheduler interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	CleanOldEvents(ctx context.Context) error
	ProduceNotifications(ctx context.Context) error
}

type Storage interface {
	ListEventsToNotify(ctx context.Context, fromTime time.Time, period time.Duration) ([]*storage.Event, error)
	DeleteOldEventsBeforeTime(ctx context.Context,
		fromTime time.Time, maxLiveTime time.Duration) ([]*storage.Event, error)
}

type Sequence interface {
	Push(ctx context.Context, exchange string, message []byte) error
}

type Scheduler struct {
	IScheduler
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

func (s *Scheduler) CleanOldEvents(ctx context.Context) error {
	s.log.Info().Msg("start deleting old events...")
	events, err := s.storage.DeleteOldEventsBeforeTime(ctx, time.Now().Local(), s.eventsDeprecationAge)
	if err != nil {
		return errors.Wrap(err, "failed to clean old events")
	}
	s.log.Info().Msgf("successfully clean old events total: '%d'", len(events))
	return nil
}

func (s *Scheduler) ProduceNotifications(ctx context.Context) error {
	s.log.Info().Msg("start sending notifications about coming events...")
	events, err := s.storage.ListEventsToNotify(ctx, time.Now().Local(), s.notifyPeriod)
	if err != nil {
		return errors.Wrap(err, "failed to get events to notify")
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
		return errors.Wrap(marshallErr, "something goes wrong when trying to marshall notification")
	}
	err := s.sequence.Push(ctx, s.exchangeToSendNotifications, message)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to push notification %s to channle %s",
			s.exchangeToSendNotifications, message))
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

		if err := s.CleanOldEvents(ctx); err != nil {
			s.log.Error().Err(err).Msgf("failed to clean old events")
			return err
		}

		if err := s.ProduceNotifications(ctx); err != nil {
			s.log.Error().Err(err).Msgf("failed to produce notifications")
			return err
		}
	}
}

func (s *Scheduler) Stop(ctx context.Context) error {
	s.log.Info().Msg("Stopping scheduler work...")
	close(s.doneChan)
	return nil
}
