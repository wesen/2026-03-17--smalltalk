package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

func main() {
	runtime.LockOSThread()

	title := flag.String("title", "SDL Hello Raw", "window title")
	width := flag.Int("width", 640, "window width")
	height := flag.Int("height", 480, "window height")
	maxSeconds := flag.Int("max-seconds", 0, "auto-exit after N seconds (0 means run until quit)")
	eventDebug := flag.Bool("event-debug", true, "log SDL events")
	flag.Parse()

	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_EVENTS); err != nil {
		fmt.Fprintf(os.Stderr, "sdl-hello-raw: init: %v\n", err)
		os.Exit(1)
	}
	defer sdl.Quit()

	driver, err := sdl.GetCurrentVideoDriver()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sdl-hello-raw: current video driver: %v\n", err)
		os.Exit(1)
	}

	window, renderer, err := sdl.CreateWindowAndRenderer(int32(*width), int32(*height), sdl.WINDOW_SHOWN|sdl.WINDOW_RESIZABLE)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sdl-hello-raw: create window: %v\n", err)
		os.Exit(1)
	}
	defer window.Destroy()
	defer renderer.Destroy()

	window.SetTitle(*title)
	window.Raise()
	windowID, _ := window.GetID()
	fmt.Printf("[sdl-hello-raw] driver=%q created-window windowID=%d title=%q size=%dx%d\n", driver, windowID, *title, *width, *height)

	start := time.Now()
	lastMouseFocusID := uint32(^uint32(0))
	lastKeyboardFocusID := uint32(^uint32(0))
	lastColorFlip := time.Now()
	blue := true

	for {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				if *eventDebug {
					fmt.Printf("[sdl-hello-raw] quit\n")
				}
				return
			case *sdl.MouseMotionEvent:
				if *eventDebug {
					fmt.Printf("[sdl-hello-raw] mouse-motion windowID=%d x=%d y=%d state=0x%X ts=%d\n",
						e.WindowID, e.X, e.Y, e.State, e.Timestamp)
				}
			case *sdl.MouseButtonEvent:
				if *eventDebug {
					fmt.Printf("[sdl-hello-raw] mouse-button windowID=%d button=%d state=%d x=%d y=%d ts=%d\n",
						e.WindowID, e.Button, e.State, e.X, e.Y, e.Timestamp)
				}
			case *sdl.KeyboardEvent:
				if *eventDebug {
					fmt.Printf("[sdl-hello-raw] keyboard windowID=%d type=%d repeat=%d sym=%d ts=%d\n",
						e.WindowID, e.Type, e.Repeat, e.Keysym.Sym, e.Timestamp)
				}
			case *sdl.TextInputEvent:
				if *eventDebug {
					fmt.Printf("[sdl-hello-raw] text-input windowID=%d text=%q ts=%d\n",
						e.WindowID, e.GetText(), e.Timestamp)
				}
			case *sdl.WindowEvent:
				if *eventDebug {
					fmt.Printf("[sdl-hello-raw] window event=%s windowID=%d data1=%d data2=%d\n",
						windowEventName(e.Event), e.WindowID, e.Data1, e.Data2)
				}
			}
		}

		mouseFocusID := focusWindowID(sdl.GetMouseFocus)
		if mouseFocusID != lastMouseFocusID {
			fmt.Printf("[sdl-hello-raw] mouse-focus windowID=%s\n", formatFocusID(mouseFocusID))
			lastMouseFocusID = mouseFocusID
		}
		keyboardFocusID := focusWindowID(sdl.GetKeyboardFocus)
		if keyboardFocusID != lastKeyboardFocusID {
			fmt.Printf("[sdl-hello-raw] keyboard-focus windowID=%s\n", formatFocusID(keyboardFocusID))
			lastKeyboardFocusID = keyboardFocusID
		}

		if time.Since(lastColorFlip) >= 750*time.Millisecond {
			blue = !blue
			lastColorFlip = time.Now()
		}
		if blue {
			renderer.SetDrawColor(0x22, 0x55, 0x99, 0xFF)
		} else {
			renderer.SetDrawColor(0x99, 0x55, 0x22, 0xFF)
		}
		renderer.Clear()
		renderer.Present()

		if *maxSeconds > 0 && time.Since(start) >= time.Duration(*maxSeconds)*time.Second {
			return
		}
		sdl.Delay(16)
	}
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
