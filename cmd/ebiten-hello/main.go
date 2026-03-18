package main

import (
	"flag"
	"fmt"
	"image/color"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func main() {
	title := flag.String("title", "Ebiten Hello", "window title")
	width := flag.Int("width", 640, "logical window width")
	height := flag.Int("height", 480, "logical window height")
	scale := flag.Int("scale", 1, "extra size multiplier")
	maxSeconds := flag.Int("max-seconds", 0, "auto-exit after N seconds (0 means run until quit)")
	eventDebug := flag.Bool("event-debug", true, "log observed Ebiten input and focus changes")
	flag.Parse()

	if *width <= 0 || *height <= 0 {
		fmt.Fprintln(os.Stderr, "ebiten-hello: width and height must be positive")
		os.Exit(1)
	}
	if *scale <= 0 {
		*scale = 1
	}

	game := &helloGame{
		title:       *title,
		width:       *width,
		height:      *height,
		maxSeconds:  *maxSeconds,
		eventDebug:  *eventDebug,
		start:       time.Now(),
		lastMouseX:  -1,
		lastMouseY:  -1,
		colorFlipAt: time.Now(),
	}

	ebiten.SetWindowTitle(*title)
	ebiten.SetWindowSize(*width**scale, *height**scale)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(game); err != nil {
		fmt.Fprintf(os.Stderr, "ebiten-hello: %v\n", err)
		os.Exit(1)
	}
}

type helloGame struct {
	title      string
	width      int
	height     int
	maxSeconds int
	eventDebug bool

	start       time.Time
	colorFlipAt time.Time
	blue        bool

	lastFocus       bool
	lastMouseInside bool
	lastMouseX      int
	lastMouseY      int
	createdLogged   bool
	textRunes       []rune
}

func (g *helloGame) Update() error {
	if !g.createdLogged {
		g.log("created-window title=%q size=%dx%d", g.title, g.width, g.height)
		g.createdLogged = true
	}

	focused := ebiten.IsFocused()
	if focused != g.lastFocus {
		g.log("focus focused=%t", focused)
		g.lastFocus = focused
	}

	x, y := ebiten.CursorPosition()
	inside := x >= 0 && y >= 0 && x < g.width && y < g.height
	if inside != g.lastMouseInside {
		g.log("mouse-inside inside=%t x=%d y=%d", inside, x, y)
		g.lastMouseInside = inside
	}
	if inside && (x != g.lastMouseX || y != g.lastMouseY) {
		g.log("mouse-motion x=%d y=%d", x, y)
	}
	g.lastMouseInside = inside
	g.lastMouseX = x
	g.lastMouseY = y

	g.pollMouseButton(ebiten.MouseButtonLeft, "left", x, y)
	g.pollMouseButton(ebiten.MouseButtonMiddle, "middle", x, y)
	g.pollMouseButton(ebiten.MouseButtonRight, "right", x, y)
	g.pollKey(ebiten.KeyBackspace, "Backspace")
	g.pollKey(ebiten.KeyTab, "Tab")
	g.pollKey(ebiten.KeyEnter, "Enter")
	g.pollKey(ebiten.KeyEscape, "Escape")
	g.pollKey(ebiten.KeyDelete, "Delete")

	g.textRunes = ebiten.AppendInputChars(g.textRunes[:0])
	for _, r := range g.textRunes {
		g.log("text-input text=%q", string(r))
	}

	if time.Since(g.colorFlipAt) >= 750*time.Millisecond {
		g.blue = !g.blue
		g.colorFlipAt = time.Now()
	}
	if g.maxSeconds > 0 && time.Since(g.start) >= time.Duration(g.maxSeconds)*time.Second {
		return ebiten.Termination
	}
	return nil
}

func (g *helloGame) Draw(screen *ebiten.Image) {
	if g.blue {
		screen.Fill(color.RGBA{R: 0x22, G: 0x55, B: 0x99, A: 0xFF})
		return
	}
	screen.Fill(color.RGBA{R: 0x99, G: 0x55, B: 0x22, A: 0xFF})
}

func (g *helloGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.width, g.height
}

func (g *helloGame) pollMouseButton(button ebiten.MouseButton, name string, x int, y int) {
	if inpututil.IsMouseButtonJustPressed(button) {
		g.log("mouse-button button=%s state=pressed x=%d y=%d", name, x, y)
	}
	if inpututil.IsMouseButtonJustReleased(button) {
		g.log("mouse-button button=%s state=released x=%d y=%d", name, x, y)
	}
}

func (g *helloGame) pollKey(key ebiten.Key, name string) {
	if inpututil.IsKeyJustPressed(key) {
		g.log("keyboard key=%s", name)
	}
}

func (g *helloGame) log(format string, args ...any) {
	if !g.eventDebug {
		return
	}
	fmt.Printf("[ebiten-hello] %s\n", fmt.Sprintf(format, args...))
}
