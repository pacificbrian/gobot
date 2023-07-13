package sparki

import (
	"io"
	"strconv"
	"go.bug.st/serial"
	"gobot.io/x/gobot/v2"
	"gobot.io/x/gobot/v2/platforms/sparki/client"
)

type sparkiBoard interface {
	Connect(io.ReadWriteCloser) error
	Disconnect() error
	DigitalWrite(int, int) error
	//Beep
	Move(int, int, float32) error
	SetRGBLED(uint, uint, uint) error
	//Stop
	gobot.Eventer
}

type SparkiAdaptor interface {
	Connect() error
	Finalize() error
	Name() string
	SetName(n string)
	gobot.Eventer
}

// Adaptor is the Gobot Adaptor for Firmata based boards
type Adaptor struct {
	name       string
	port       string
	Board      sparkiBoard
	conn       io.ReadWriteCloser
	PortOpener func(port string) (io.ReadWriteCloser, error)
	gobot.Eventer
}

// NewAdaptor returns a new Firmata Adaptor which optionally accepts:
//
//	string: port the Adaptor uses to connect to a serial port with a baude rate of 57600
//	io.ReadWriteCloser: connection the Adaptor uses to communication with the hardware
//
// If an io.ReadWriteCloser is not supplied, the Adaptor will open a connection
// to a serial port with a baude rate of 57600. If an io.ReadWriteCloser
// is supplied, then the Adaptor will use the provided io.ReadWriteCloser and use the
// string port as a label to be displayed in the log and api.
func NewAdaptor(args ...interface{}) *Adaptor {
	f := &Adaptor{
		name:  gobot.DefaultName("Sparki"),
		port:  "",
		conn:  nil,
		Board: client.New(),
		PortOpener: func(port string) (io.ReadWriteCloser, error) {
			return serial.Open(port, &serial.Mode{BaudRate: 57600})
		},
		Eventer: gobot.NewEventer(),
	}

	for _, arg := range args {
		switch a := arg.(type) {
		case string:
			f.port = a
		case io.ReadWriteCloser:
			f.conn = a
		}
	}

	return f
}

// Connect starts a connection to the board.
func (f *Adaptor) Connect() error {
	if f.conn == nil {
		sp, err := f.PortOpener(f.Port())
		if err != nil {
			return err
		}
		f.conn = sp
	}
	return f.Board.Connect(f.conn)
}

// Disconnect closes the io connection to the Board
func (f *Adaptor) Disconnect() error {
	if f.Board != nil {
		return f.Board.Disconnect()
	}
	return nil
}

// Finalize terminates the firmata connection
func (f *Adaptor) Finalize() error {
	return f.Disconnect()
}

// Port returns the Firmata Adaptors port
func (f *Adaptor) Port() string { return f.port }

// Name returns the Firmata Adaptors name
func (f *Adaptor) Name() string { return f.name }

// SetName sets the Firmata Adaptors name
func (f *Adaptor) SetName(n string) { f.name = n }

// DigitalWrite writes a value to the pin. Acceptable values are 1 or 0.
func (f *Adaptor) DigitalWrite(pin string, level byte) error {
	p, err := strconv.Atoi(pin)
	if err != nil {
		return err
	}

	return f.Board.DigitalWrite(p, int(level))
}

func (f *Adaptor) Move(left float32, right float32, time float32) error {
	return f.Board.Move(int(left*10), int(right*10), time)
}
