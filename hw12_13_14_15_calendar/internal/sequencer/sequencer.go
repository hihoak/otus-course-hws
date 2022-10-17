package sequencer

type Sequencer interface {
	Connect() error
	Close() error
}
