package rabbit

import (
	"context"
	"fmt"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/logger"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/pkg/config"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/sequencer"
	amqp "github.com/rabbitmq/amqp091-go"
)

const calendarKey = "calendar"

type Client struct {
	log        *logger.Logger
	config     amqp.Config
	rabbitURL  string
	connection *amqp.Connection

	exchanges map[string]interface{}
	queues    map[string]interface{}
	bindings  []config.Bind

	sequencer.Sequencer
}

func NewCLient(
	logger *logger.Logger,
	rabbitURL, clientName string,
	exchanges []string,
	queues []string,
	bindings []config.Bind,
) *Client {
	cfg := amqp.Config{
		Properties: amqp.NewConnectionProperties(),
	}
	cfg.Properties.SetClientConnectionName(clientName)
	clExchanges := make(map[string]interface{}, len(exchanges))
	for _, exchangeName := range exchanges {
		clExchanges[exchangeName] = nil
	}
	clQueues := make(map[string]interface{}, len(queues))
	for _, queue := range queues {
		clQueues[queue] = nil
	}

	return &Client{
		log:       logger,
		rabbitURL: rabbitURL,
		config:    cfg,
		exchanges: clExchanges,
		queues:    clQueues,
		bindings:  bindings,
	}
}

func (c *Client) Connect() error {
	c.log.Info().Msgf("starting new connection to %s", c.rabbitURL)
	conn, err := amqp.DialConfig(c.rabbitURL, c.config)
	if err != nil {
		c.log.Error().Err(err).Msgf("failed connect to %s", c.rabbitURL)
		return fmt.Errorf("failed connect to %s: %w", c.rabbitURL, err)
	}
	c.log.Info().Msgf("successfully connected to %s", c.rabbitURL)
	c.connection = conn
	c.log.Info().Msg("making a connection channel...")
	channel, err := c.connection.Channel()
	if err != nil {
		return fmt.Errorf("failed to establish channel: %w", err)
	}
	defer func() {
		if closeErr := channel.Close(); closeErr != nil {
			c.log.Error().Err(closeErr).Msg("failed to close channel")
		}
	}()
	c.log.Info().Msgf("Successfully create a connection channel:"+
		" %v. Now starting declare exchanges from config...", channel)
	for exchangeName := range c.exchanges {
		if exchangeErr := channel.ExchangeDeclare(
			exchangeName, amqp.ExchangeDirect,
			true, false,
			false, true, nil); exchangeErr != nil {
			return fmt.Errorf("failed to declare exchange: %w", exchangeErr)
		}
	}

	for queueName := range c.queues {
		if _, queueErr := channel.QueueDeclare(
			queueName, true,
			false, false,
			true, nil); queueErr != nil {
			return fmt.Errorf("failed to declare queue: %w", queueErr)
		}
	}

	for _, binding := range c.bindings {
		if bindErr := channel.QueueBind(binding.QueueName, binding.Key, binding.ExchangeName, false, nil); bindErr != nil {
			return fmt.Errorf("failed to bind exchange '%s' and queue '%s': %w",
				binding.ExchangeName, binding.QueueName, bindErr)
		}
	}

	return nil
}

func (c *Client) Close() error {
	c.log.Info().Msgf("start closing connection to rabbit %s", c.connection.RemoteAddr())
	if err := c.connection.Close(); err != nil {
		c.log.Error().Err(err).Msgf("failed to close connection with rabbit")
		return fmt.Errorf("failed to close connection with rabbit: %w", err)
	}
	c.log.Info().Msgf("successfully close connection with rabbit %s", c.connection.RemoteAddr())
	return nil
}

func (c *Client) Push(ctx context.Context, exchange string, messages [][]byte) error {
	c.log.Debug().Msgf("start sending messages %d to exchange '%s'", len(messages), exchange)
	if _, ok := c.exchanges[exchange]; !ok {
		return fmt.Errorf("exchange '%s' doesn't exists", exchange)
	}
	go func() {
		err := <-c.connection.NotifyClose(make(chan *amqp.Error))
		fmt.Println(err)
	}()
	channel, err := c.connection.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel to send message: %w", err)
	}
	defer func() {
		if closeErr := channel.Close(); closeErr != nil {
			c.log.Error().Err(err).Msg("failed to close channel")
		}
	}()
	for _, message := range messages {
		err = channel.PublishWithContext(ctx, exchange, calendarKey, false, false, amqp.Publishing{
			Body: message,
		})
		if err != nil {
			return fmt.Errorf("failed to deliver a message '%s' to exchange '%s' with key '%s': %w",
				message, exchange, exchange, err)
		}
		c.log.Debug().Msgf("successfully published message"+
			" message '%s' to exchange '%s' with key '%s'",
			message, exchange, exchange)
	}
	return nil
}

func (c Client) Pull(ctx context.Context, queue string) (<-chan []byte, error) {
	c.log.Debug().Msgf("start consuming from queue '%s'", queue)
	if _, ok := c.queues[queue]; !ok {
		return nil, fmt.Errorf("queue '%s' doesn't exists", queue)
	}
	channel, err := c.connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to start connection: %w", err)
	}

	ch, err := channel.Consume(queue, "calendar-sender", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start consuming: %w", err)
	}

	messages := make(chan []byte)
	go func() {
		for msg := range ch {
			c.log.Debug().Msgf("got a message from '%s': %s", queue, msg.Body)
			messages <- msg.Body
		}
		c.log.Debug().Msg("stop consuming")
	}()

	c.log.Debug().Msgf("successfully start consuming messages from queue '%s'", queue)
	return messages, nil
}
