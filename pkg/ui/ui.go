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
			sdl.Quit()
			return nil
		})
	}()

	var window *sdl.Window
	var renderer *sdl.Renderer
	var texture *sdl.Texture
	var textureWidth int
	var textureHeight int
	var pixels []uint32
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
			copyDisplayBits(pixels, snapshot)
		}

		quit, err := processEventsAndPresent(renderer, texture, pixels, textureWidth, textureHeight)
		if err != nil {
			return err
		}
		if quit {
			return nil
		}
	}
}

func processEventsAndPresent(renderer *sdl.Renderer, texture *sdl.Texture, pixels []uint32, width int, height int) (bool, error) {
	quit := false
	err := doSDL(func() error {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent:
				quit = true
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

func copyDisplayBits(dst []uint32, snapshot interpreter.DisplaySnapshot) (blackPixels int, whitePixels int) {
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
	return blackPixels, whitePixels
}

func doSDL(fn func() error) error {
	var err error
	sdl.Do(func() {
		err = fn()
	})
	return err
}
