package ui

import (
	"fmt"

	"github.com/veandco/go-sdl2/sdl"
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
// designated Smalltalk display form in an SDL window until the window closes
// or MaxCycles is reached.
func Run(opts Options) error {
	if opts.ImagePath == "" {
		opts.ImagePath = "data/VirtualImage"
	}
	if opts.CyclesPerFrame == 0 {
		opts.CyclesPerFrame = 50000
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

	var runErr error
	sdl.Main(func() {
		runErr = runLoop(interp, opts)
	})
	return runErr
}

func runLoop(interp *interpreter.Interpreter, opts Options) error {
	if err := doSDL(func() error {
		return sdl.Init(sdl.INIT_VIDEO | sdl.INIT_EVENTS)
	}); err != nil {
		return err
	}
	defer func() {
		_ = doSDL(func() error {
			sdl.StopTextInput()
			sdl.Quit()
			return nil
		})
	}()
	if err := doSDL(func() error {
		sdl.StartTextInput()
		return nil
	}); err != nil {
		return err
	}

	var window *sdl.Window
	var renderer *sdl.Renderer
	var texture *sdl.Texture
	var textureWidth int
	var textureHeight int
	var pixels []uint32
	var lastInputStats interpreter.InputStats
	defer func() {
		_ = doSDL(func() error {
			if texture != nil {
				_ = texture.Destroy()
			}
			if renderer != nil {
				_ = renderer.Destroy()
			}
			if window != nil {
				_ = window.Destroy()
			}
			return nil
		})
	}()

	for {
		if opts.MaxCycles > 0 && interp.CycleCount() >= opts.MaxCycles {
			return nil
		}

		steps := opts.CyclesPerFrame
		if opts.MaxCycles > 0 {
			remaining := opts.MaxCycles - interp.CycleCount()
			if remaining < steps {
				steps = remaining
			}
		}
		if err := interp.RunSteps(steps); err != nil {
			return err
		}

		snapshot, ok := interp.DisplaySnapshot()
		if ok {
			cursor, hasCursor := interp.CursorSnapshot()
			if texture == nil || snapshot.Width != textureWidth || snapshot.Height != textureHeight {
				if err := doSDL(func() error {
					if texture != nil {
						_ = texture.Destroy()
						texture = nil
					}
					if renderer != nil {
						_ = renderer.Destroy()
						renderer = nil
					}
					if window != nil {
						_ = window.Destroy()
						window = nil
					}

					var err error
					window, renderer, err = sdl.CreateWindowAndRenderer(
						int32(snapshot.Width)*opts.Scale,
						int32(snapshot.Height)*opts.Scale,
						sdl.WINDOW_SHOWN|sdl.WINDOW_RESIZABLE,
					)
					if err != nil {
						return err
					}
					window.SetTitle(opts.WindowTitle)
					if err := renderer.SetLogicalSize(int32(snapshot.Width), int32(snapshot.Height)); err != nil {
						return err
					}
					texture, err = renderer.CreateTexture(
						sdl.PIXELFORMAT_ARGB8888,
						sdl.TEXTUREACCESS_STREAMING,
						int32(snapshot.Width),
						int32(snapshot.Height),
					)
					if err != nil {
						return err
					}
					if err := texture.SetScaleMode(sdl.ScaleModeNearest); err != nil {
						return err
					}
					textureWidth = snapshot.Width
					textureHeight = snapshot.Height
					return nil
				}); err != nil {
					return err
				}
			}

			if len(pixels) != snapshot.Width*snapshot.Height {
				pixels = make([]uint32, snapshot.Width*snapshot.Height)
			}
			copyDisplayBits(pixels, snapshot, hasCursor, cursor)
		}

		quit, err := processEventsAndPresent(interp, window, renderer, texture, pixels, textureWidth, textureHeight, opts.EventDebug)
		if err != nil {
			return err
		}
		if quit {
			return nil
		}
		if opts.InputDebug {
			stats := interp.InputStats()
			if stats != lastInputStats {
				fmt.Printf("[input-debug cycle=%d] motions=%d buttons=%d keys=%d enqueued=%d dequeued=%d queue=%d\n",
					interp.CycleCount(),
					stats.MouseMotionsRecorded,
					stats.MouseButtonsRecorded,
					stats.DecodedKeysRecorded,
					stats.WordsEnqueued,
					stats.WordsDequeued,
					stats.QueueDepth,
				)
				lastInputStats = stats
			}
		}
	}
}

func processEventsAndPresent(interp *interpreter.Interpreter, window *sdl.Window, renderer *sdl.Renderer, texture *sdl.Texture, pixels []uint32, width int, height int, eventDebug bool) (bool, error) {
	quit := false
	err := doSDL(func() error {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				if eventDebug {
					fmt.Printf("[event-debug cycle=%d] quit\n", interp.CycleCount())
				}
				quit = true
			case *sdl.MouseMotionEvent:
				if eventDebug {
					fmt.Printf("[event-debug cycle=%d] mouse-motion x=%d y=%d state=0x%X ts=%d\n",
						interp.CycleCount(), e.X, e.Y, e.State, e.Timestamp)
				}
				logicalX, logicalY, ok := mapWindowToLogicalPoint(window, width, height, e.X, e.Y)
				if ok {
					interp.RecordMouseMotion(logicalX, logicalY, e.Timestamp)
				}
			case *sdl.MouseButtonEvent:
				if eventDebug {
					fmt.Printf("[event-debug cycle=%d] mouse-button button=%d state=%d x=%d y=%d ts=%d\n",
						interp.CycleCount(), e.Button, e.State, e.X, e.Y, e.Timestamp)
				}
				logicalX, logicalY, ok := mapWindowToLogicalPoint(window, width, height, e.X, e.Y)
				if ok {
					interp.SetMousePoint(logicalX, logicalY)
					if parameter, ok := mouseButtonParameter(e.Button); ok {
						interp.RecordMouseButton(parameter, e.State == sdl.PRESSED, logicalX, logicalY, e.Timestamp)
					}
				}
			case *sdl.TextInputEvent:
				if eventDebug {
					fmt.Printf("[event-debug cycle=%d] text-input text=%q ts=%d\n",
						interp.CycleCount(), e.GetText(), e.Timestamp)
				}
				for _, r := range e.GetText() {
					if r < 0 || r > 0x7F {
						continue
					}
					interp.RecordDecodedKey(uint16(r), e.Timestamp)
				}
			case *sdl.KeyboardEvent:
				if eventDebug {
					fmt.Printf("[event-debug cycle=%d] keyboard type=%d repeat=%d sym=%d ts=%d\n",
						interp.CycleCount(), e.Type, e.Repeat, e.Keysym.Sym, e.Timestamp)
				}
				if e.Type == sdl.KEYDOWN && e.Repeat == 0 {
					if parameter, ok := specialKeyParameter(e.Keysym.Sym); ok {
						interp.RecordDecodedKey(parameter, e.Timestamp)
					}
				}
			}
		}
		if renderer == nil || texture == nil || width <= 0 || height <= 0 || len(pixels) == 0 {
			return nil
		}
		if err := texture.UpdateRGBA(nil, pixels, width); err != nil {
			return err
		}
		if err := renderer.Clear(); err != nil {
			return err
		}
		if err := renderer.Copy(texture, nil, nil); err != nil {
			return err
		}
		renderer.Present()
		return nil
	})
	return quit, err
}

func mapWindowToLogicalPoint(window *sdl.Window, logicalWidth int, logicalHeight int, x int32, y int32) (int, int, bool) {
	if window == nil || logicalWidth <= 0 || logicalHeight <= 0 {
		return 0, 0, false
	}
	windowWidth, windowHeight := window.GetSize()
	if windowWidth <= 0 || windowHeight <= 0 {
		return 0, 0, false
	}
	logicalX := int(int64(x) * int64(logicalWidth) / int64(windowWidth))
	logicalY := int(int64(y) * int64(logicalHeight) / int64(windowHeight))
	if logicalX >= logicalWidth {
		logicalX = logicalWidth - 1
	}
	if logicalY >= logicalHeight {
		logicalY = logicalHeight - 1
	}
	return logicalX, logicalY, true
}

func mouseButtonParameter(button sdl.Button) (uint16, bool) {
	switch button {
	case sdl.ButtonLeft:
		return 128, true
	case sdl.ButtonMiddle:
		return 129, true
	case sdl.ButtonRight:
		return 130, true
	default:
		return 0, false
	}
}

func specialKeyParameter(sym sdl.Keycode) (uint16, bool) {
	switch sym {
	case sdl.K_BACKSPACE:
		return 8, true
	case sdl.K_TAB:
		return 9, true
	case sdl.K_RETURN, sdl.K_RETURN2:
		return 13, true
	case sdl.K_ESCAPE:
		return 27, true
	case sdl.K_DELETE:
		return 127, true
	default:
		return 0, false
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

func doSDL(fn func() error) error {
	var err error
	sdl.Do(func() {
		err = fn()
	})
	return err
}
