package ui

import (
	"fmt"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/wesen/st80/pkg/image"
	"github.com/wesen/st80/pkg/interpreter"
)

// Options control the host UI loop.
type Options struct {
	ImagePath      string
	CyclesPerFrame uint64
	MaxCycles      uint64
	Scale          int32
	WindowTitle    string
	InputDebug     bool
	EventDebug     bool
}

// Run boots the image, advances the interpreter in chunks, and displays the
// designated Smalltalk display form in an Ebiten window until the window closes
// or MaxCycles is reached.
func Run(opts Options) error {
	if opts.ImagePath == "" {
		opts.ImagePath = "data/VirtualImage"
	}
	if opts.CyclesPerFrame == 0 {
		opts.CyclesPerFrame = 5000
	}
	if opts.Scale <= 0 {
		opts.Scale = 2
	}
	if opts.WindowTitle == "" {
		opts.WindowTitle = "Smalltalk-80"
	}

	om, err := image.LoadImage(opts.ImagePath)
	if err != nil {
		return fmt.Errorf("load image: %w", err)
	}
	interp := interpreter.New(om)
	interp.SetSnapshotPath(opts.ImagePath)

	game := &hostGame{
		interp:        interp,
		opts:          opts,
		logicalWidth:  640,
		logicalHeight: 480,
		hostStart:     time.Now(),
		lastMouseX:    -1,
		lastMouseY:    -1,
	}

	ebiten.SetWindowTitle(opts.WindowTitle)
	ebiten.SetWindowSize(game.logicalWidth*int(opts.Scale), game.logicalHeight*int(opts.Scale))
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(game); err != nil {
		return err
	}
	return nil
}

type hostGame struct {
	interp *interpreter.Interpreter
	opts   Options

	logicalWidth  int
	logicalHeight int

	frame      *ebiten.Image
	pixels32   []uint32
	pixelBytes []byte

	hostStart time.Time

	lastInputStats  interpreter.InputStats
	lastFocus       bool
	lastMouseInside bool
	lastMouseX      int
	lastMouseY      int
	createdLogged   bool

	textRunes []rune
}

func (g *hostGame) Update() error {
	if !g.createdLogged {
		g.logEvent("created-window title=%q size=%dx%d", g.opts.WindowTitle, g.logicalWidth, g.logicalHeight)
		g.createdLogged = true
	}

	g.pollInput()

	if g.opts.MaxCycles > 0 && g.interp.CycleCount() >= g.opts.MaxCycles {
		return ebiten.Termination
	}

	steps := g.opts.CyclesPerFrame
	if g.opts.MaxCycles > 0 {
		remaining := g.opts.MaxCycles - g.interp.CycleCount()
		if remaining < steps {
			steps = remaining
		}
	}
	if steps > 0 {
		if err := g.interp.RunSteps(steps); err != nil {
			return err
		}
	}

	if snapshot, ok := g.interp.DisplaySnapshot(); ok {
		cursor, hasCursor := g.interp.CursorSnapshot()
		g.ensureFrameSize(snapshot.Width, snapshot.Height)
		copyDisplayBits(g.pixels32, snapshot, hasCursor, cursor)
		packARGBToRGBA(g.pixelBytes, g.pixels32)
		g.frame.WritePixels(g.pixelBytes)
	}

	if g.opts.InputDebug {
		stats := g.interp.InputStats()
		if stats != g.lastInputStats {
			fmt.Printf("[input-debug cycle=%d] motions=%d buttons=%d keys=%d enqueued=%d dequeued=%d queue=%d\n",
				g.interp.CycleCount(),
				stats.MouseMotionsRecorded,
				stats.MouseButtonsRecorded,
				stats.DecodedKeysRecorded,
				stats.WordsEnqueued,
				stats.WordsDequeued,
				stats.QueueDepth,
			)
			g.lastInputStats = stats
		}
	}

	if g.opts.MaxCycles > 0 && g.interp.CycleCount() >= g.opts.MaxCycles {
		return ebiten.Termination
	}

	return nil
}

func (g *hostGame) Draw(screen *ebiten.Image) {
	screen.Fill(color.White)
	if g.frame != nil {
		screen.DrawImage(g.frame, nil)
	}
}

func (g *hostGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.logicalWidth, g.logicalHeight
}

func (g *hostGame) ensureFrameSize(width int, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	if g.frame != nil && width == g.logicalWidth && height == g.logicalHeight {
		return
	}
	g.logicalWidth = width
	g.logicalHeight = height
	g.frame = ebiten.NewImage(width, height)
	g.pixels32 = make([]uint32, width*height)
	g.pixelBytes = make([]byte, width*height*4)
	ebiten.SetWindowSize(width*int(g.opts.Scale), height*int(g.opts.Scale))
	g.logEvent("resized logical=%dx%d scale=%d", width, height, g.opts.Scale)
}

func (g *hostGame) pollInput() {
	focused := ebiten.IsFocused()
	if focused != g.lastFocus {
		g.logEvent("focus focused=%t", focused)
		g.lastFocus = focused
	}

	x, y := ebiten.CursorPosition()
	inside := x >= 0 && y >= 0 && x < g.logicalWidth && y < g.logicalHeight
	if inside != g.lastMouseInside {
		g.logEvent("mouse-inside inside=%t x=%d y=%d", inside, x, y)
		g.lastMouseInside = inside
	}
	if inside {
		g.interp.SetMousePoint(x, y)
	}
	if inside && (x != g.lastMouseX || y != g.lastMouseY) {
		g.logEvent("mouse-motion x=%d y=%d", x, y)
		g.interp.RecordMouseMotion(x, y, g.timestamp())
	}
	g.lastMouseX = x
	g.lastMouseY = y

	g.pollMouseButton(ebiten.MouseButtonLeft, "left", 128, x, y, inside)
	g.pollMouseButton(ebiten.MouseButtonMiddle, "middle", 129, x, y, inside)
	g.pollMouseButton(ebiten.MouseButtonRight, "right", 130, x, y, inside)

	g.textRunes = ebiten.AppendInputChars(g.textRunes[:0])
	for _, r := range g.textRunes {
		if r < 0 || r > 0x7F {
			continue
		}
		g.logEvent("text-input text=%q", string(r))
		g.interp.RecordDecodedKey(uint16(r), g.timestamp())
	}

	g.pollSpecialKey(ebiten.KeyBackspace, "Backspace", 8)
	g.pollSpecialKey(ebiten.KeyTab, "Tab", 9)
	g.pollSpecialKey(ebiten.KeyEnter, "Enter", 13)
	g.pollSpecialKey(ebiten.KeyEscape, "Escape", 27)
	g.pollSpecialKey(ebiten.KeyDelete, "Delete", 127)
}

func (g *hostGame) pollMouseButton(button ebiten.MouseButton, name string, parameter uint16, x int, y int, inside bool) {
	if inpututil.IsMouseButtonJustPressed(button) {
		g.logEvent("mouse-button button=%s state=pressed x=%d y=%d", name, x, y)
		if inside {
			g.interp.RecordMouseButton(parameter, true, x, y, g.timestamp())
		}
	}
	if inpututil.IsMouseButtonJustReleased(button) {
		g.logEvent("mouse-button button=%s state=released x=%d y=%d", name, x, y)
		if inside {
			g.interp.RecordMouseButton(parameter, false, x, y, g.timestamp())
		}
	}
}

func (g *hostGame) pollSpecialKey(key ebiten.Key, name string, parameter uint16) {
	if inpututil.IsKeyJustPressed(key) {
		g.logEvent("keyboard key=%s", name)
		g.interp.RecordDecodedKey(parameter, g.timestamp())
	}
}

func (g *hostGame) timestamp() uint32 {
	return uint32(time.Since(g.hostStart).Milliseconds())
}

func (g *hostGame) logEvent(format string, args ...any) {
	if !g.opts.EventDebug {
		return
	}
	fmt.Printf("[event-debug cycle=%d] %s\n", g.interp.CycleCount(), fmt.Sprintf(format, args...))
}

func packARGBToRGBA(dst []byte, src []uint32) {
	for i, pixel := range src {
		base := i * 4
		dst[base+0] = byte(pixel >> 16)
		dst[base+1] = byte(pixel >> 8)
		dst[base+2] = byte(pixel)
		dst[base+3] = byte(pixel >> 24)
	}
}

func copyDisplayBits(dst []uint32, snapshot interpreter.DisplaySnapshot, hasCursor bool, cursor interpreter.CursorSnapshot) (blackPixels int, whitePixels int) {
	white := uint32(0xFFFFFFFF)
	black := uint32(0xFF000000)
	for y := 0; y < snapshot.Height; y++ {
		rowBase := y * snapshot.Raster
		pixelBase := y * snapshot.Width
		for x := 0; x < snapshot.Width; x++ {
			word := snapshot.Words[rowBase+x/16]
			mask := uint(15 - (x & 15))
			if (word>>mask)&1 != 0 {
				dst[pixelBase+x] = black
				blackPixels++
			} else {
				dst[pixelBase+x] = white
				whitePixels++
			}
		}
	}
	if hasCursor {
		overlayCursorBits(dst, snapshot.Width, snapshot.Height, cursor, black)
	}
	return blackPixels, whitePixels
}

func overlayCursorBits(dst []uint32, displayWidth int, displayHeight int, cursor interpreter.CursorSnapshot, black uint32) {
	for cy := 0; cy < cursor.Height; cy++ {
		y := cursor.Y + cy
		if y < 0 || y >= displayHeight {
			continue
		}
		rowBase := cy * cursor.Raster
		pixelBase := y * displayWidth
		for cx := 0; cx < cursor.Width; cx++ {
			x := cursor.X + cx
			if x < 0 || x >= displayWidth {
				continue
			}
			word := cursor.Words[rowBase+cx/16]
			mask := uint(15 - (cx & 15))
			if (word>>mask)&1 != 0 {
				dst[pixelBase+x] = black
			}
		}
	}
}
