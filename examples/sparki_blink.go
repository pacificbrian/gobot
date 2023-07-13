//go:build example
// +build example

//
// Do not build by default.

/*
 How to run
 Pass serial port to use as the first param:

	go run examples/sparki_blink.go /dev/ttyACM0
*/

package main

import (
	"fmt"
	"os"
	"time"

	"gobot.io/x/gobot/v2"
	"gobot.io/x/gobot/v2/drivers/gpio"
	"gobot.io/x/gobot/v2/platforms/sparki"
)

func main() {
	sparkiAdaptor := sparki.NewAdaptor(os.Args[1])
	led := gpio.NewLedDriver(sparkiAdaptor, "13")

	work := func() {
		gobot.Every(3*time.Second, func() {
			led.Toggle()
		})
	}

	robot := gobot.NewRobot("bot",
		[]gobot.Connection{sparkiAdaptor},
		[]gobot.Device{led},
		work,
	)

	err := robot.Start()
	if err != nil {
		fmt.Println(err)
	}
}
