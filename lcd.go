package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	lcd1602 "github.com/pimvanhespen/go-pi-lcd1602"
	"github.com/pimvanhespen/go-pi-lcd1602/animations"
	"github.com/pimvanhespen/go-pi-lcd1602/stringutils"
	"github.com/pimvanhespen/go-pi-lcd1602/synchronized"

	"github.com/fhs/gompd/mpd"
)

func c(s string) string {
	return stringutils.Center(s, 16)
}

func main() {
	lcdi := lcd1602.New(
		25,                    //rs
		17,                    //enable
		[]int{27, 22, 23, 24}, //datapins
		16,                    //lineSize
	)
	lcd := synchronized.NewSynchronizedLCD(lcdi)
	lcd.Initialize()
	//            1234567812345678  1234567812345678
	lcd.WriteLines(c("Radio booting "), c("¯\\(°_o)/¯"))
	// does not seem to work
	//defer lcd.Clear()
	//defer lcd.Close()

	w, err := mpd.NewWatcher("tcp", ":6600", "")
	if err != nil {
		log.Println("Error:", err.Error())
		lcd.WriteLines("Error", err.Error())
	}
	defer w.Close()

	// Log errors.
	go func() {
		for err := range w.Error {
			log.Println("Error:", err.Error())
			lcd.WriteLines("Error", err.Error())
		}
	}()

	// Log events.
	go func() {
		line1 := ""
		line2 := ""
		for subsystem := range w.Event {
			log.Println("Changed subsystem:", subsystem)
			conn, err := mpd.Dial("tcp", ":6600")
			if err != nil {
				return
			}
			status, err := conn.Status()
			if err != nil {
				log.Fatalln(err)
			}
			fmt.Printf("status: %#v\n", status)
			song, err := conn.CurrentSong()
			if err != nil {
				log.Println("Error:", err.Error())
				lcd.WriteLines("Error", err.Error())
			} else {
				fmt.Printf("song: %#v\n", song)
				if status["state"] == "play" {
					line1n := song["Artist"]
					line2n := song["Title"]
					// guess artist from title by splitting of first segment
					// might be used by streams
					if line1n == "" {
						split := strings.SplitN(song["Title"], " - ", 2)
						if len(split) > 1 {
							line1n = split[0]
							line2n = split[1]
						} else {
							line2n = split[0]
						}
					}

					// slide in artist and title when playing
					// TODO implement marqee for longer strings
					if line1n != line1 {
						line1 = line1n
						<-lcd.Animate(animations.SlideInRight(c(line1)), lcd1602.LINE_1)
					}

					if line2n != line2 {
						line2 = line2n
						<-lcd.Animate(animations.SlideInRight(c(line2)), lcd1602.LINE_2)
					}
				} else {
					lcd.WriteLines("", c(status["state"]))
				}
			}
		}
	}()

	time.Sleep(1 * time.Second)
	lcd.WriteLines(c("Hey"), "")

	// wait for signal
	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	<-gracefulStop
	lcd.WriteLines(c("Bye Bye"), "")
	lcd.Clear()
	lcd.Close()
	os.Exit(0)
}
