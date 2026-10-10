package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/afsutulov/DangerousDave/internal/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed icon.png
var iconData embed.FS
var version = "1.0.0"

type application struct {
	data                         *game.Data
	world                        *game.World
	art                          *art
	sound                        *soundBank
	mode, previousMode, savePath string
	frames                       int
	capture                      string
	captureTicks                 int
	captured                     bool
	captureErr                   error
	demo                         bool
}

func pressed(keys ...ebiten.Key) bool {
	for _, k := range keys {
		if inpututil.IsKeyJustPressed(k) {
			return true
		}
	}
	return false
}
func held(keys ...ebiten.Key) bool {
	for _, k := range keys {
		if ebiten.IsKeyPressed(k) {
			return true
		}
	}
	return false
}
func (a *application) newGame(level int) { a.world = game.New(a.data, level); a.mode = "play" }
func (a *application) load() {
	w, e := game.ReadSave(a.data, a.savePath)
	if e != nil {
		a.world.Notify("NO VALID SAVED GAME", 120)
		if a.mode == "title" {
			a.mode = "play"
		}
		return
	}
	a.world = w
	a.mode = "play"
}
func (a *application) Update() error {
	if a.captured {
		return ebiten.Termination
	}
	a.frames++
	if pressed(ebiten.KeyF11) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	if pressed(ebiten.KeyF2) {
		a.sound.toggle()
		a.world.Notify(fmt.Sprintf("SOUND %s", map[bool]string{true: "ON", false: "OFF"}[a.sound.enabled]), 90)
	}
	if pressed(ebiten.KeyF1) {
		if a.mode == "help" {
			a.mode = a.previousMode
		} else {
			a.previousMode = a.mode
			a.mode = "help"
		}
		return nil
	}
	if a.mode == "title" {
		if pressed(ebiten.KeyEscape) {
			return ebiten.Termination
		}
		if pressed(ebiten.KeyEnter, ebiten.KeySpace) {
			a.newGame(0)
		}
		if pressed(ebiten.KeyC) {
			a.load()
		}
		return nil
	}
	if a.mode == "help" {
		if pressed(ebiten.KeyEscape, ebiten.KeyEnter) {
			a.mode = a.previousMode
		}
		return nil
	}
	if pressed(ebiten.KeyF5) {
		if e := a.world.Save(a.savePath); e != nil {
			a.world.Notify("CANNOT SAVE NOW", 90)
		} else {
			a.world.Notify("GAME SAVED", 90)
		}
	}
	if pressed(ebiten.KeyF9) {
		a.load()
		return nil
	}
	if a.mode == "restart" {
		if pressed(ebiten.KeyEnter) {
			a.world.LoadLevel(a.world.Level)
			a.mode = "play"
		}
		if pressed(ebiten.KeyEscape) {
			a.mode = "pause"
		}
		return nil
	}
	if a.mode == "pause" {
		if pressed(ebiten.KeyEscape, ebiten.KeyP) {
			a.mode = "play"
		}
		if pressed(ebiten.KeyR) {
			a.mode = "restart"
		}
		if pressed(ebiten.KeyM) {
			a.mode = "title"
		}
		return nil
	}
	if a.world.Won || a.world.GameOver {
		if pressed(ebiten.KeyEnter) {
			if a.world.Won {
				a.newGame(0)
			} else {
				a.newGame(a.world.Level)
			}
		}
		if pressed(ebiten.KeyM, ebiten.KeyEscape) {
			a.mode = "title"
		}
		return nil
	}
	if pressed(ebiten.KeyEscape, ebiten.KeyP) {
		a.mode = "pause"
		return nil
	}
	in := game.Input{Left: held(ebiten.KeyArrowLeft, ebiten.KeyA), Right: held(ebiten.KeyArrowRight, ebiten.KeyD), Up: held(ebiten.KeyArrowUp, ebiten.KeyW), Down: held(ebiten.KeyArrowDown, ebiten.KeyS), Jump: pressed(ebiten.KeyControlLeft, ebiten.KeyControlRight, ebiten.KeySpace), Shoot: held(ebiten.KeyAltLeft, ebiten.KeyAltRight, ebiten.KeyX), Use: pressed(ebiten.KeyArrowUp, ebiten.KeyW, ebiten.KeyE)}
	if a.demo {
		in = game.Input{Right: a.frames < 160, Shoot: a.frames > 120 && a.frames%60 < 20, Jump: a.frames == 80}
	}
	a.world.Update(in)
	for _, e := range a.world.Events {
		a.sound.play(e)
	}
	return nil
}
func (a *application) captureImage(screen *ebiten.Image) {
	p := make([]byte, 320*200*4)
	screen.ReadPixels(p)
	im := &image.RGBA{Pix: p, Stride: 320 * 4, Rect: image.Rect(0, 0, 320, 200)}
	scaled := image.NewRGBA(image.Rect(0, 0, 960, 600))
	for y := 0; y < 600; y++ {
		for x := 0; x < 960; x++ {
			scaled.SetRGBA(x, y, im.RGBAAt(x/3, y/3))
		}
	}
	if e := os.MkdirAll(filepath.Dir(a.capture), 0755); e != nil {
		a.captureErr = e
		a.captured = true
		return
	}
	f, e := os.Create(a.capture)
	if e == nil {
		e = png.Encode(f, scaled)
		if closeErr := f.Close(); e == nil {
			e = closeErr
		}
	}
	a.captureErr = e
	a.captured = true
}
func defaultSavePath() string {
	dir, e := os.UserConfigDir()
	if e != nil {
		dir = "."
	}
	return filepath.Join(dir, "DangerousDaveGo", "save.json")
}
func main() {
	level := flag.Int("level", 0, "start a level directly (1-8); 0 opens the title screen")
	mute := flag.Bool("mute", false, "disable sound")
	fullscreen := flag.Bool("fullscreen", false, "start fullscreen")
	scale := flag.Int("scale", 3, "initial window scale (1-6)")
	capture := flag.String("capture", "", "save a screenshot and exit")
	ticks := flag.Int("ticks", 1, "frames before --capture")
	demo := flag.Bool("demo", false, "scripted movement for screenshot capture")
	save := flag.String("save", defaultSavePath(), "save game location")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("Dangerous Dave Go %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
		return
	}
	if *level < 0 || *level > 8 || *scale < 1 || *scale > 6 || *ticks < 0 {
		log.Fatal("level must be 0-8, scale 1-6, ticks >= 0")
	}
	data, e := game.LoadData()
	if e != nil {
		log.Fatal(e)
	}
	art, e := loadArt()
	if e != nil {
		log.Fatal(e)
	}
	app := &application{data: data, world: game.New(data, max(0, *level-1)), art: art, mode: "title", savePath: *save, capture: *capture, captureTicks: *ticks, demo: *demo}
	app.sound = newSoundBank(!*mute)
	if *level > 0 {
		app.mode = "play"
	}
	ebiten.SetWindowTitle("Dangerous Dave in the Haunted Mansion - Go")
	ebiten.SetWindowSize(320**scale, 200**scale)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(*fullscreen)
	ebiten.SetTPS(game.TickRate)
	if b, e := iconData.ReadFile("icon.png"); e == nil {
		if im, _, e := image.Decode(bytes.NewReader(b)); e == nil {
			ebiten.SetWindowIcon([]image.Image{im})
		}
	}
	if e = ebiten.RunGame(app); e != nil {
		log.Fatal(e)
	}
	if app.captureErr != nil {
		log.Fatal(app.captureErr)
	}
}
