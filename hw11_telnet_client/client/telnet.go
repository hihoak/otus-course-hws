package client

import (
	"io"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type TelnetClient interface {
	Connect() error
	io.Closer
	Send() error
	Receive() error
}

type Telnet struct {
	address    string
	timeout    time.Duration
	in         io.ReadCloser
	out        io.Writer
	connection net.Conn

	stopChan chan interface{}
	log      zerolog.Logger
}

func NewTelnetClient(address string, timeout time.Duration, in io.ReadCloser, out io.Writer) TelnetClient {
	return &Telnet{
		address:  address,
		timeout:  timeout,
		in:       in,
		out:      out,
		log:      zerolog.Logger{},
		stopChan: make(chan interface{}),
	}
}

func (t *Telnet) Connect() error {
	var err error
	t.connection, err = net.DialTimeout("tcp", t.address, t.timeout)
	if err != nil {
		return errors.Wrap(ErrorEstablishConnection{}, err.Error())
	}

	t.log.Info().Msgf("Successfully connected to %s", t.address)
	return nil
}

func (t *Telnet) Close() error {
	close(t.stopChan)
	if err := t.connection.Close(); err != nil {
		return errors.Wrap(err, "failed to close connection")
	}
	t.log.Info().Msgf("Successfully close connection to %s", t.address)
	return nil
}

func (t *Telnet) Send() error {
	defer func() {
		t.log.Err(t.in.Close()).Msg("failed to close input")
	}()

	// read worker
	stopChan := make(chan interface{})
	defer close(stopChan)

	messageChan := make(chan []byte)
	readWorkerErrorChan := make(chan error)
	go func() {
		defer close(messageChan)
		defer close(readWorkerErrorChan)
		// defer fmt.Println("sender read worker finished!")
		for {
			select {
			case <-stopChan:
				return
			default:
				buffer := make([]byte, 4096)
				n, err := t.in.Read(buffer)
				if err != nil {
					if errors.Is(err, io.EOF) {
						t.log.Info().Msg("Input is finished. Stop receiving...")
					} else {
						readWorkerErrorChan <- errors.Wrap(err, "something goes wrong while trying to read input data")
					}
					return
				}
				messageChan <- buffer[:n]
			}
		}
	}()

	// write worker
	writeWorkerErrorChan := make(chan error)
	defer close(writeWorkerErrorChan)
	for {
		select {
		case <-t.stopChan:
			// fmt.Println("sender finished because of global stop channel")
			return nil

		case err := <-readWorkerErrorChan:
			// fmt.Println("sender finished because of read worker error: ", err)
			return err
		case err := <-writeWorkerErrorChan:
			// fmt.Println("sender finished because of write worker error: ", err)
			return err

		case msg, ok := <-messageChan:
			if !ok {
				// fmt.Println("sender finished because message channel is closed")
				return nil
			}
			go func(b []byte) {
				if _, err := t.connection.Write(b); err != nil {
					writeWorkerErrorChan <- errors.Wrap(err, "can't send message")
				}
			}(msg)
		}
	}
}

func (t *Telnet) Receive() error {
	stopChan := make(chan interface{})
	defer close(stopChan)
	// read worker
	messageChan := make(chan []byte)
	readWorkerErrorsChan := make(chan error, 1)
	go func() {
		defer close(messageChan)
		defer close(readWorkerErrorsChan)
		// defer fmt.Println("receiver read worker finished!")
		for {
			select {
			case <-stopChan:
				return
			default:
				buffer := make([]byte, 4096)
				n, err := t.connection.Read(buffer)
				if err != nil {
					if !errors.Is(err, io.EOF) {
						readWorkerErrorsChan <- errors.Wrap(err, "can't read message")
					}
					return
				}
				messageChan <- buffer[:n]
			}
		}
	}()

	// write worker
	for {
		select {
		case <-t.stopChan:
			// fmt.Println("receiver finished by global stopChannel")
			return nil

		case err := <-readWorkerErrorsChan:
			// fmt.Println("receiver finished by a error from readWorker: ", err)
			return err

		case msg, ok := <-messageChan:
			if !ok {
				// fmt.Println("receiver finished because message channel is closed")
				return nil
			}
			if _, err := t.out.Write(msg); err != nil {
				// fmt.Println("receiver finished because of write error: ", err)
				return errors.Wrap(err, "can't write into out")
			}
		}
	}
}
