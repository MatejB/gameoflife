package main

import (
	"log"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/nsf/termbox-go"
)

func main() {
	drawCh, exitCh, displayWidth, displayHeight := display()

	w := initWorld(displayWidth, displayHeight)

	go func() {
		for {
			drawCh <- w
			w = tick(w)
			time.Sleep(100 * time.Millisecond)
		}
	}()

	<-exitCh
}

func initWorld(width, height int) [][]bool {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	w := make([][]bool, width)

	for i := range w {
		w[i] = make([]bool, height)
		for j := range w[i] {
			if r.Intn(10) > 5 {
				w[i][j] = true
			} else {
				w[i][j] = false
			}
		}
	}

	return w
}

func tick(w [][]bool) [][]bool {
	var newW [][]bool

	newW = make([][]bool, len(w))

	for i := range w {
		newW[i] = make([]bool, len(w[i]))

		for j := range w[i] {
			lnc := liveNeighbours(w, i, j)

			if w[i][j] == true && (lnc == 2 || lnc == 3) {
				newW[i][j] = true
			} else if w[i][j] == false && lnc == 3 {
				newW[i][j] = true
			} else {
				newW[i][j] = false
			}
		}
	}

	return newW
}

type cords struct {
	X int
	Y int
}

func liveNeighbours(w [][]bool, x, y int) int {
	lnc := 0

	isset := func(w [][]bool, x, y int) bool {
		if x < 0 ||
			y < 0 ||
			x+1 > len(w) ||
			y+1 > len(w[x]) {
			return false
		}
		return true
	}

	neighboursCombos := []cords{
		{x - 1, y - 1},
		{x - 1, y},
		{x - 1, y + 1},
		{x, y - 1},
		{x, y + 1},
		{x + 1, y - 1},
		{x + 1, y},
		{x + 1, y + 1}}

	for _, combo := range neighboursCombos {
		if isset(w, combo.X, combo.Y) {
			if w[combo.X][combo.Y] == true {
				lnc = lnc + 1
			}
		}
	}

	return lnc
}

func display() (drawCh chan [][]bool, exitCh chan bool, displayWidth, displayHeight int) {
	drawCh = make(chan [][]bool)
	exitCh = make(chan bool)

	err := termbox.Init()
	if err != nil {
		log.Printf("Cannot start, termbox.Init() gave an error:\n%s\n", err)
		os.Exit(1)
	}
	termbox.HideCursor()
	termbox.Clear(termbox.ColorBlack, termbox.ColorBlack)

	displayWidth, displayHeight = termbox.Size()

	fpsSleepTime := time.Duration(1000000/60) * time.Microsecond
	go func() {
		for {
			time.Sleep(fpsSleepTime)
			termbox.Flush()
		}
	}()

	eventChan := make(chan termbox.Event)
	go func() {
		for {
			event := termbox.PollEvent()
			eventChan <- event
		}
	}()

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, os.Kill)

	go func() {
		for {
			select {
			case wToDraw := <-drawCh:
				termbox.Clear(termbox.ColorBlack, termbox.ColorBlack)
				for i := range wToDraw {
					for j := range wToDraw[i] {
						char := ' '
						if wToDraw[i][j] == true {
							char = 'X'
						}
						termbox.SetCell(i, j, char, termbox.ColorWhite, termbox.ColorBlack)
					}
				}
			case event := <-eventChan:
				switch event.Type {
				case termbox.EventKey:
					switch event.Key {
					case termbox.KeyCtrlZ, termbox.KeyCtrlC:
						termbox.Close()
						exitCh <- true
						return
					}
				case termbox.EventError: // quit
					termbox.Close()
					log.Fatalf("Quitting because of termbox error: \n%s\n", event.Err)
				}
			case signal := <-sigChan:
				log.Printf("Have signal: \n%s", spew.Sdump(signal))
				termbox.Close()
				exitCh <- true
				return
			}
		}
	}()

	return
}
