package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nsf/termbox-go"
)

func main() {
	drawCh, exitCh, displayWidth, displayHeight := display()

	w := NewWorld(displayWidth, displayHeight)
	drawCh <- w.Plane

	go func() {
		ticker := time.Tick(100 * time.Millisecond)
		for _ = range ticker {
			w.Tick()
			drawCh <- w.Plane
		}
	}()

	<-exitCh
	fmt.Println(w.LifeCycles, "life cycles")
}

type World struct {
	Plane      [][]bool
	BackPlain  [][]bool
	LifeCycles int
}

func NewWorld(width, height int) (world *World) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	plane := make([][]bool, width)

	for i := range plane {
		plane[i] = make([]bool, height)
		for j := range plane[i] {
			if r.Intn(10) > 5 {
				plane[i][j] = true
			} else {
				plane[i][j] = false
			}
		}
	}

	return &World{plane, make([][]bool, 0), 0}
}

func (w *World) Tick() {
	w.BackPlain = make([][]bool, len(w.Plane))

	for i := range w.Plane {
		w.BackPlain[i] = make([]bool, len(w.Plane[i]))

		for j := range w.Plane[i] {
			lnc := w.liveNeighbours(i, j)

			if w.Plane[i][j] == true && (lnc == 2 || lnc == 3) {
				w.BackPlain[i][j] = true
			} else if w.Plane[i][j] == false && lnc == 3 {
				w.BackPlain[i][j] = true
			} else {
				w.BackPlain[i][j] = false
			}
		}
	}

	w.LifeCycles += 1
	w.Plane = w.BackPlain
}

type cords struct {
	X int
	Y int
}

func (w *World) liveNeighbours(x, y int) int {
	lnc := 0

	isset := func(x, y int) bool {
		if x < 0 ||
			y < 0 ||
			x+1 > len(w.Plane) ||
			y+1 > len(w.Plane[x]) {
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
		if isset(combo.X, combo.Y) {
			if w.Plane[combo.X][combo.Y] == true {
				lnc = lnc + 1
			}
		}
	}

	return lnc
}

func display() (drawCh chan [][]bool, exitCh chan struct{}, displayWidth, displayHeight int) {
	drawCh = make(chan [][]bool)
	exitCh = make(chan struct{})

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

	eventCh := make(chan termbox.Event)
	go func() {
		for {
			event := termbox.PollEvent()
			eventCh <- event
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill, syscall.SIGTERM)

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
			case event := <-eventCh:
				switch event.Type {
				case termbox.EventKey:
					switch event.Key {
					case termbox.KeyCtrlZ, termbox.KeyCtrlC:
						termbox.Close()
						close(exitCh)
						return
					}
				case termbox.EventError: // quit
					termbox.Close()
					log.Fatalf("Quitting because of termbox error: \n%s\n", event.Err)
				}
			case signal := <-sigCh:
				termbox.Close()
				log.Printf("Recived signal: \n%s", signal)
				close(exitCh)
				return
			}
		}
	}()

	return
}
