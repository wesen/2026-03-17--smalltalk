package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

func main() {
	title := flag.String("title", "SDL Hello", "window title")
	width := flag.Int("width", 640, "window width")
	height := flag.Int("height", 480, "window height")
	scale := flag.Int("scale", 1, "extra size multiplier")
	maxSeconds := flag.Int("max-seconds", 0, "auto-exit after N seconds (0 means run until quit)")
	eventDebug := flag.Bool("event-debug", true, "log SDL events")
	flag.Parse()

	if *scale <= 0 {
		*scale = 1
	}

	var runErr error
	sdl.Main(func() {
		runErr = run(*title, *width**scale, *height**scale, *maxSeconds, *eventDebug)
	})
	if runErr != nil {
		fmt.Fprintf(os.Stderr, "sdl-hello: %v\n", runErr)
		os.Exit(1)
	}
}

func run(title string, width int, height int, maxSeconds int, eventDebug bool) error {
	if err := doSDL(func() error {
		return sdl.Init(sdl.INIT_VIDEO | sdl.INIT_EVENTS)
	}); err != nil {
		return err
	}
	defer func() {
		_ = doSDL(func() error {
			sdl.Quit()
			return nil
		})
	}()

	var window *sdl.Window
	var renderer *sdl.Renderer
	if err := doSDL(func() error {
		var err error
		window, renderer, err = sdl.CreateWindowAndRenderer(int32(width), int32(height), sdl.WINDOW_SHOWN|sdl.WINDOW_RESIZABLE)
		if err != nil {
			return err
		}
		window.SetTitle(title)
		window.Raise()
		if id, err := window.GetID(); err == nil {
			fmt.Printf("[sdl-hello] created-window windowID=%d title=%q size=%dx%d\n", id, title, width, height)
		}
		return nil
	}); err != nil {
		return err
	}
	defer func() {
		_ = doSDL(func() error {
			if renderer != nil {
				_ = renderer.Destroy()
			}
			if window != nil {
				_ = window.Destroy()
			}
			return nil
		})
	}()

	start := time.Now()
	lastMouseFocusID := uint32(^uint32(0))
	lastKeyboardFocusID := uint32(^uint32(0))
	lastColorFlip := time.Now()
	blue := true

	for {
		quit := false
		if err := doSDL(func() error {
			for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
				switch e := event.(type) {
				case *sdl.QuitEvent:
					if eventDebug {
						fmt.Printf("[sdl-hello] quit\n")
					}
					quit = true
				case *sdl.MouseMotionEvent:
					if eventDebug {
						fmt.Printf("[sdl-hello] mouse-motion windowID=%d x=%d y=%d state=0x%X ts=%d\n",
							e.WindowID, e.X, e.Y, e.State, e.Timestamp)
					}
				case *sdl.MouseButtonEvent:
					if eventDebug {
						fmt.Printf("[sdl-hello] mouse-button windowID=%d button=%d state=%d x=%d y=%d ts=%d\n",
							e.WindowID, e.Button, e.State, e.X, e.Y, e.Timestamp)
					}
				case *sdl.KeyboardEvent:
					if eventDebug {
						fmt.Printf("[sdl-hello] keyboard windowID=%d type=%d repeat=%d sym=%d ts=%d\n",
							e.WindowID, e.Type, e.Repeat, e.Keysym.Sym, e.Timestamp)
					}
				case *sdl.TextInputEvent:
					if eventDebug {
						fmt.Printf("[sdl-hello] text-input windowID=%d text=%q ts=%d\n",
							e.WindowID, e.GetText(), e.Timestamp)
					}
				case *sdl.WindowEvent:
					if eventDebug {
						fmt.Printf("[sdl-hello] window event=%s windowID=%d data1=%d data2=%d\n",
							windowEventName(e.Event), e.WindowID, e.Data1, e.Data2)
					}
				}
			}

			mouseFocusID := focusWindowID(sdl.GetMouseFocus)
			if mouseFocusID != lastMouseFocusID {
				fmt.Printf("[sdl-hello] mouse-focus windowID=%s\n", formatFocusID(mouseFocusID))
				lastMouseFocusID = mouseFocusID
			}
			keyboardFocusID := focusWindowID(sdl.GetKeyboardFocus)
			if keyboardFocusID != lastKeyboardFocusID {
				fmt.Printf("[sdl-hello] keyboard-focus windowID=%s\n", formatFocusID(keyboardFocusID))
				lastKeyboardFocusID = keyboardFocusID
			}

			if time.Since(lastColorFlip) >= 750*time.Millisecond {
				blue = !blue
				lastColorFlip = time.Now()
			}
			if blue {
				if err := renderer.SetDrawColor(0x22, 0x55, 0x99, 0xFF); err != nil {
					return err
				}
			} else {
				if err := renderer.SetDrawColor(0x99, 0x55, 0x22, 0xFF); err != nil {
					return err
				}
			}
			if err := renderer.Clear(); err != nil {
				return err
			}
			renderer.Present()
			return nil
		}); err != nil {
			return err
		}
		if quit {
			return nil
		}
		if maxSeconds > 0 && time.Since(start) >= time.Duration(maxSeconds)*time.Second {
			return nil
		}
		time.Sleep(16 * time.Millisecond)
	}
}

func doSDL(fn func() error) error {
	var err error
	sdl.Do(func() {
		err = fn()
	})
	return err
}

func focusWindowID(getFocus func() *sdl.Window) uint32 {
	focused := getFocus()
	if focused == nil {
		return 0
	}
	id, err := focused.GetID()
	if err != nil {
		return 0
	}
	return id
}

func formatFocusID(id uint32) string {
	if id == 0 {
		return "none"
	}
	return fmt.Sprintf("%d", id)
}

func windowEventName(event sdl.WindowEventID) string {
	switch event {
	case sdl.WINDOWEVENT_SHOWN:
		return "shown"
	case sdl.WINDOWEVENT_HIDDEN:
		return "hidden"
	case sdl.WINDOWEVENT_EXPOSED:
		return "exposed"
	case sdl.WINDOWEVENT_MOVED:
		return "moved"
	case sdl.WINDOWEVENT_RESIZED:
		return "resized"
	case sdl.WINDOWEVENT_SIZE_CHANGED:
		return "size-changed"
	case sdl.WINDOWEVENT_MINIMIZED:
		return "minimized"
	case sdl.WINDOWEVENT_MAXIMIZED:
		return "maximized"
	case sdl.WINDOWEVENT_RESTORED:
		return "restored"
	case sdl.WINDOWEVENT_ENTER:
		return "enter"
	case sdl.WINDOWEVENT_LEAVE:
		return "leave"
	case sdl.WINDOWEVENT_FOCUS_GAINED:
		return "focus-gained"
	case sdl.WINDOWEVENT_FOCUS_LOST:
		return "focus-lost"
	case sdl.WINDOWEVENT_CLOSE:
		return "close"
	case sdl.WINDOWEVENT_TAKE_FOCUS:
		return "take-focus"
	case sdl.WINDOWEVENT_HIT_TEST:
		return "hit-test"
	default:
		return fmt.Sprintf("%d", event)
	}
}
