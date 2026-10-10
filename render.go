package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"math"

	"github.com/afsutulov/DangerousDave/assets"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/basicfont"
)

type frame struct{ X, Y, W, H int }
type art struct {
	tiles  []*ebiten.Image
	frames map[string][]*ebiten.Image
	title  *ebiten.Image
	death  *ebiten.Image
	face   text.Face
	white  *ebiten.Image
}

func loadArt() (*art, error) {
	a := &art{frames: map[string][]*ebiten.Image{}, face: text.NewGoXFace(basicfont.Face7x13), white: ebiten.NewImage(1, 1)}
	a.white.Fill(color.White)
	read := func(name string) (*ebiten.Image, error) {
		f, e := assets.FS.Open(name)
		if e != nil {
			return nil, e
		}
		defer f.Close()
		im, _, e := image.Decode(f)
		if e != nil {
			return nil, e
		}
		return ebiten.NewImageFromImage(im), nil
	}
	tiles, e := read("tiles.png")
	if e != nil {
		return nil, e
	}
	for i := 0; i < 858; i++ {
		x, y := i%16*16, i/16*16
		a.tiles = append(a.tiles, tiles.SubImage(image.Rect(x, y, x+16, y+16)).(*ebiten.Image))
	}
	b, e := assets.FS.ReadFile("sprites.json")
	if e != nil {
		return nil, e
	}
	var fs map[string][]frame
	if e = json.Unmarshal(b, &fs); e != nil {
		return nil, e
	}
	for k, frames := range fs {
		atlas, e := read("s_" + k + ".png")
		if e != nil {
			return nil, e
		}
		for _, f := range frames {
			a.frames[k] = append(a.frames[k], atlas.SubImage(image.Rect(f.X, f.Y, f.X+f.W, f.Y+f.H)).(*ebiten.Image))
		}
	}
	a.title, e = read("title1.png")
	if e != nil {
		return nil, e
	}
	if _, e := assets.FS.ReadFile("death.png"); e == nil {
		a.death, e = read("death.png")
		if e != nil {
			return nil, e
		}
	}
	return a, nil
}

var palette = []color.RGBA{{0, 0, 0, 255}, {0, 0, 170, 255}, {0, 170, 0, 255}, {0, 170, 170, 255}, {170, 0, 0, 255}, {170, 0, 170, 255}, {170, 85, 0, 255}, {170, 170, 170, 255}, {85, 85, 85, 255}, {85, 85, 255, 255}, {85, 255, 85, 255}, {85, 255, 255, 255}, {255, 85, 85, 255}, {255, 85, 255, 255}, {255, 255, 85, 255}, {255, 255, 255, 255}}

func (a *art) box(dst *ebiten.Image, x, y, w, h float64, c color.Color) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w, h)
	op.GeoM.Translate(x, y)
	r, g, b, al := c.RGBA()
	op.ColorScale.Scale(float32(r)/65535, float32(g)/65535, float32(b)/65535, float32(al)/65535)
	dst.DrawImage(a.white, op)
}
func (a *art) label(dst *ebiten.Image, s string, x, y float64, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(math.Round(x), math.Round(y))
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, a.face, op)
}
func (a *art) center(dst *ebiten.Image, s string, y float64, c color.Color) {
	a.label(dst, s, float64(320-len(s)*7)/2, y, c)
}
func (a *art) image(dst, im *ebiten.Image, x, y float64, flip bool) {
	op := &ebiten.DrawImageOptions{}
	if flip {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(float64(im.Bounds().Dx()), 0)
	}
	op.GeoM.Translate(math.Round(x), math.Round(y))
	dst.DrawImage(im, op)
}
func (a *art) sprite(dst *ebiten.Image, set string, index int, x, y float64, flip bool) {
	s := a.frames[set]
	if index >= 0 && index < len(s) {
		a.image(dst, s[index], x, y, flip)
	}
}
func (app *application) drawWorld(screen *ebiten.Image) {
	a, w := app.art, app.world
	screen.Fill(palette[0])
	cx, cy := math.Floor(w.CameraX), math.Floor(w.CameraY)
	// The HUD is separate from the 320x176 world viewport.
	view := screen.SubImage(image.Rect(0, 24, 320, 200)).(*ebiten.Image)
	for y := int(cy / 16); y <= int((cy+176)/16); y++ {
		for x := int(cx / 16); x <= int((cx+320)/16); x++ {
			if x < 0 || y < 0 || x >= w.Map.Width || y >= w.Map.Height {
				continue
			}
			t := w.Map.Tiles[y*w.Map.Width+x]
			a.image(view, a.tiles[t], float64(x*16)-cx, float64(y*16)-cy+24, false)
		}
	}
	for _, d := range w.Doors {
		if d.Open {
			x, y := float64(d.X)-cx, float64(d.Y)-cy+24
			a.box(view, x-8, y-16, 31, 48, palette[0])
			a.box(view, x-8, y-16, 3, 48, palette[7])
			a.box(view, x+20, y-16, 3, 48, palette[8])
			a.box(view, x-8, y+29, 31, 3, palette[7])
		}
	}
	for _, l := range w.Loot {
		if !l.Taken {
			frame := 51 + (l.Kind-14)*2
			if l.Kind == 19 {
				frame = 68
			}
			a.sprite(view, "dave", frame, l.X-cx, l.Y-cy+24, false)
		}
	}
	// Delbert is visible in the last chamber, beyond the two energy columns.
	for index, tag := range w.Map.Objects {
		if tag == 13 {
			x, y := float64(index%w.Map.Width*16+8)-cx, float64(index/w.Map.Width*16+32)-cy+24
			a.sprite(view, "master", 6+w.Tick/20%3, x-32, y-56, false)
		}
	}
	for _, e := range w.Enemies {
		if e.HP <= 0 {
			continue
		}
		if e.X-cx < -90 || e.X-cx > 330 || e.Y-cy < -100 || e.Y-cy > 190 {
			continue
		}
		set, f, flip := "chunk1", 0, false
		switch e.Kind {
		case 1:
			f = e.Anim / 10 % 4
			if e.Facing < 0 {
				f += 7
			}
		case 2:
			f = 17 + e.Anim/9%4
			if e.Facing < 0 {
				f += 4
			}
		case 3:
			set = "chunk2"
			f = e.Anim / 10 % 4
			flip = e.Facing > 0
		case 4:
			set = "chunk2"
			f = 8 + e.Anim/8%8
			if !e.Ground {
				f = 16 + e.Anim/8%8
			}
			flip = e.Facing > 0
		case 5:
			set = "chunk2"
			f = 25 + e.Anim/8%4
			flip = e.Facing > 0
		case 6:
			set = "chunk2"
			f = 32 + e.Anim/8%4
			flip = e.Facing > 0
		case 7:
			set = "chunk2"
			f = 37 + e.Anim/8%4
			flip = e.Facing > 0
		case 9:
			set = "chunk2"
			f = 29
		case 11:
			set = "frank"
			f = e.Anim / 12 % 4 * 2
			if e.Facing < 0 {
				f++
			}
		case 12:
			set = "master"
			f = e.Anim / 18 % 2
			flip = e.Facing > 0
		}
		frames := a.frames[set]
		if f < len(frames) {
			im := frames[f]
			a.image(view, im, e.X+e.W/2-float64(im.Bounds().Dx())/2-cx, e.Y+e.H-float64(im.Bounds().Dy())-cy+24, flip)
		}
		if e.Kind >= 11 {
			a.box(screen, 80, 19, 160, 3, palette[4])
			a.box(screen, 80, 19, 160*float64(e.HP)/float64(e.MaxHP), 3, palette[12])
		}
	}
	p := &w.Player
	if w.Death == 0 && (p.Invulnerable == 0 || w.Tick%8 < 4) {
		f := 0
		if p.Facing < 0 {
			f = 8
		}
		if p.VX != 0 {
			f += 1 + p.Walk/6%4
		}
		if !p.Ground {
			f = 5
			if p.Facing < 0 {
				f = 13
			}
			if p.VY > 0 {
				f += 2
			}
		}
		if p.Reload > 15 && p.VX == 0 {
			f = 16
			if p.Facing < 0 {
				f = 17
			}
		}
		if p.Fire > 0 {
			f = 27
			if p.AimY < 0 {
				if p.AimX == 0 {
					f = 29
				}
			} else if p.AimY > 0 {
				f = 31
			}
			if p.Facing < 0 {
				f += 6
			}
		}
		a.sprite(view, "dave", f, p.X-5-cx, p.Y-4-cy+24, false)
	}
	for _, s := range w.Shots {
		if s.Enemy {
			if s.Kind == 2 {
				a.sprite(view, "chunk1", 16, s.X-cx-4, s.Y-cy+16, false)
			} else {
				a.sprite(view, "master", 2+w.Tick/6%4, s.X-cx-12, s.Y-cy+12, false)
			}
		} else {
			a.box(view, s.X-cx-2, s.Y-cy+23, 4, 2, palette[14])
		}
	}
	for _, p := range w.Particles {
		a.box(view, p.X-cx, p.Y-cy+24, 2, 2, palette[p.Color])
	}
	for _, p := range w.Popups {
		a.label(view, p.Text, p.X-cx, p.Y-cy+24, palette[14])
	}
	a.box(screen, 0, 0, 320, 24, palette[0])
	a.box(screen, 0, 23, 320, 1, palette[8])
	a.label(screen, fmt.Sprintf("%07d", w.Score), 4, 4, palette[14])
	a.label(screen, fmt.Sprintf("L%d", w.Level+1), 70, 4, palette[15])
	a.label(screen, fmt.Sprintf("LIVES %d", w.Lives), 98, 4, palette[11])
	a.sprite(screen, "dave", 18+p.Ammo, 165, 3, false)
	a.label(screen, "DAVE II", 242, 4, palette[12])
	for _, e := range w.Enemies {
		if e.Kind >= 11 && e.HP > 0 {
			a.box(screen, 98, 19, 160, 3, palette[4])
			a.box(screen, 98, 19, 160*float64(e.HP)/float64(e.MaxHP), 3, palette[12])
		}
	}
	if w.MessageTime > 0 {
		a.box(screen, 0, 183, 320, 17, color.RGBA{0, 0, 0, 220})
		a.center(screen, w.Message, 185, palette[14])
	}
	if w.Death > 0 {
		app.drawDeath(screen)
	}
}
func (app *application) drawDeath(screen *ebiten.Image) {
	a, w := app.art, app.world
	a.box(screen, 128, 69, 64, 64, palette[0])
	a.box(screen, 128, 69, 64, 1, palette[12])
	a.box(screen, 128, 132, 64, 1, palette[12])
	if a.death != nil {
		row := 7
		switch w.DeathKind {
		case 1, 11:
			row = 2
		case 2:
			row = 0
		case 3:
			row = 5
		case 4:
			row = 1
		case 5:
			row = 3
		case 6, 12:
			row = 6
		case 7:
			row = 4
		}
		frame := min(4, (120-w.Death)/20)
		im := a.death.SubImage(image.Rect(frame*48, row*48, frame*48+48, row*48+48)).(*ebiten.Image)
		a.image(screen, im, 136, 77, false)
	} else {
		a.sprite(screen, "dave", 42, 148, 82, false)
	}
	a.center(screen, "DAVE HAS DIED", 145, palette[12])
}
func (app *application) panel(screen *ebiten.Image, title string, lines []string) {
	a := app.art
	a.box(screen, 8, 35, 304, 154, color.RGBA{0, 0, 0, 240})
	a.box(screen, 8, 35, 304, 1, palette[11])
	a.center(screen, title, 43, palette[14])
	for i, s := range lines {
		a.center(screen, s, float64(65+i*15), palette[15])
	}
}
func (app *application) Draw(screen *ebiten.Image) {
	if app.mode == "title" {
		app.art.image(screen, app.art.title, 0, 0, false)
		app.art.box(screen, 0, 137, 320, 63, color.RGBA{0, 0, 0, 238})
		app.art.center(screen, "GO EDITION", 140, palette[11])
		app.art.center(screen, "ENTER  NEW GAME", 155, palette[15])
		app.art.center(screen, "C  CONTINUE    F1  HELP", 170, palette[15])
		app.art.center(screen, "ESC  QUIT", 185, palette[8])
	} else {
		app.drawWorld(screen)
		switch app.mode {
		case "pause":
			app.panel(screen, "PAUSED", []string{"P / ESC  RESUME", "F5  SAVE     F9  LOAD", "R  RESTART LEVEL", "M  RETURN TO TITLE", "F1  HELP    F2  SOUND", "F11  FULLSCREEN"})
		case "help":
			app.panel(screen, "DAVE COMMANDS", []string{"ARROWS / WASD  MOVE AND AIM", "CTRL / SPACE  JUMP", "DOWN + JUMP  DROP THROUGH", "ALT / X  FIRE SHOTGUN", "UP / E  OPEN / ENTER DOOR", "STAND STILL TO RELOAD", "ESC / P  PAUSE"})
		case "restart":
			app.panel(screen, "RESTART THIS LEVEL?", []string{"CURRENT LEVEL PROGRESS IS LOST", "ENTER  RESTART", "ESC  CANCEL"})
		}
		if app.world.GameOver {
			app.panel(screen, "GAME OVER", []string{"ENTER  RETRY LEVEL", "M  TITLE SCREEN"})
		}
		if app.world.Won {
			app.panel(screen, "DELBERT IS SAFE!", []string{"YOU SURVIVED THE HAUNTED MANSION", fmt.Sprintf("FINAL SCORE %d", app.world.Score), "ENTER  PLAY AGAIN", "M  TITLE SCREEN"})
		}
	}
	if app.capture != "" && app.frames >= app.captureTicks && !app.captured {
		app.captureImage(screen)
	}
}
func (app *application) Layout(_, _ int) (int, int) { return 320, 200 }
