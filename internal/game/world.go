// Package game implements the simulation without a windowing or audio dependency.
package game

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/afsutulov/DangerousDave/assets"
)

const (
	TileSize     = 16
	TickRate     = 60
	ViewWidth    = 320
	ViewHeight   = 176
	PlayerWidth  = 14.0
	PlayerHeight = 28.0
	Gravity      = 0.22
	JumpSpeed    = -5.65
	WalkSpeed    = 1.7
)

type Level struct {
	Width, Height  int
	Tiles, Objects []int
}
type TileInfo struct{ Up, Down, Right, Left, Behavior byte }
type Data struct {
	Levels []Level
	Tiles  []TileInfo
}

func LoadData() (*Data, error) {
	var d Data
	b, e := assets.FS.ReadFile("levels.json")
	if e != nil {
		return nil, e
	}
	if e = json.Unmarshal(b, &d.Levels); e != nil {
		return nil, e
	}
	b, e = assets.FS.ReadFile("tileinfo.bin")
	if e != nil {
		return nil, e
	}
	if len(b) != 858*5 {
		return nil, fmt.Errorf("invalid tile properties: %d bytes", len(b))
	}
	d.Tiles = make([]TileInfo, 858)
	for i := range d.Tiles {
		d.Tiles[i] = TileInfo{b[i], b[858+i], b[1716+i], b[2574+i], b[3432+i]}
	}
	if len(d.Levels) != 8 {
		return nil, fmt.Errorf("expected eight levels")
	}
	for n, l := range d.Levels {
		if l.Width < 1 || l.Height < 1 || len(l.Tiles) != l.Width*l.Height || len(l.Objects) != len(l.Tiles) {
			return nil, fmt.Errorf("invalid level %d", n+1)
		}
		count := 0
		for i, t := range l.Tiles {
			if t < 0 || t >= len(d.Tiles) {
				return nil, fmt.Errorf("level %d tile %d out of range", n+1, i)
			}
			if l.Objects[i] == 255 {
				count++
			}
		}
		if count != 1 {
			return nil, fmt.Errorf("level %d needs one player spawn", n+1)
		}
	}
	return &d, nil
}

type Rect struct{ X, Y, W, H float64 }

func (a Rect) Overlap(b Rect) bool {
	return a.X < b.X+b.W && a.X+a.W > b.X && a.Y < b.Y+b.H && a.Y+a.H > b.Y
}
func (a Rect) Center() (float64, float64) { return a.X + a.W/2, a.Y + a.H/2 }

type Body struct {
	X, Y, VX, VY, W, H float64
	Ground             bool
}

func (b *Body) Rect() Rect { return Rect{b.X, b.Y, b.W, b.H} }

type Player struct {
	Body
	Facing, Ammo, Reload, Cooldown, Drop, Invulnerable, Walk, Fire, JumpBuffer, Coyote int
	AimX, AimY                                                                         float64
}
type Enemy struct {
	Body
	Kind, HP, MaxHP, Facing, Timer, Anim, Dead int
	Active                                     bool
	HomeX, HomeY                               float64
}
type Loot struct {
	X, Y        float64
	Kind, Index int
	Taken       bool
}
type Door struct {
	X, Y, Tag, Index int
	Open             bool
}
type Shot struct {
	X, Y, VX, VY float64
	Life         int
	Enemy        bool
	Kind         int
}
type Particle struct {
	X, Y, VX, VY float64
	Life, Color  int
}
type Popup struct {
	X, Y float64
	Text string
	Life int
}
type Input struct{ Left, Right, Up, Down, Jump, Shoot, Use bool }
type World struct {
	Data                                                       *Data
	Level                                                      int
	Map                                                        Level
	Player                                                     Player
	Enemies                                                    []Enemy
	Loot                                                       []Loot
	Doors                                                      []Door
	Shots                                                      []Shot
	Particles                                                  []Particle
	Popups                                                     []Popup
	Score, Lives, NextLife, Tick, Death, Transition, Completed int
	CameraX, CameraY                                           float64
	Message                                                    string
	MessageTime                                                int
	Events                                                     []string
	Won, GameOver                                              bool
	seed                                                       uint32
	DeathKind                                                  int
}

func New(d *Data, level int) *World {
	w := &World{Data: d, Lives: 3, NextLife: 10000, seed: 0x1971}
	w.LoadLevel(level)
	return w
}
func (w *World) LoadLevel(level int) {
	level = max(0, min(level, len(w.Data.Levels)-1))
	w.Level = level
	src := w.Data.Levels[level]
	w.Map = Level{src.Width, src.Height, append([]int(nil), src.Tiles...), append([]int(nil), src.Objects...)}
	w.Enemies = nil
	w.Loot = nil
	w.Doors = nil
	w.Shots = nil
	w.Particles = nil
	w.Popups = nil
	w.Death = 0
	w.Transition = 0
	w.Won = false
	w.GameOver = false
	w.Player = Player{Body: Body{W: PlayerWidth, H: PlayerHeight}, Facing: 1, Ammo: 8, Invulnerable: 90, AimX: 1}
	for i, tag := range src.Objects {
		x, y := i%src.Width*16, i/src.Width*16
		switch {
		case tag == 255:
			w.Player.X = float64(x + 1)
			w.Player.Y = float64(y+32) - PlayerHeight
		case tag >= 1 && tag <= 12:
			if tag == 8 || tag == 10 {
				continue
			}
			e := Enemy{Kind: tag, HP: 1, Facing: -1, HomeX: float64(x), HomeY: float64(y), Timer: 90 + i%120}
			e.W, e.H = 16, 32
			switch tag {
			case 2:
				e.W, e.H = 18, 22
			case 3:
				e.H = 28
				e.HP = 2
			case 4:
				e.W, e.H = 20, 8
			case 5:
				e.W, e.H = 24, 8
			case 6:
				e.W, e.H = 20, 14
				e.HP = 2
			case 7:
				e.W, e.H = 26, 28
				e.HP = 3
			case 9:
				e.W, e.H = 12, 22
			case 11:
				e.W, e.H = 28, 76
				e.HP = 16
			case 12:
				e.W, e.H = 48, 76
				e.HP = 24
			}
			e.X = float64(x)
			e.Y = float64(y+32) - e.H
			e.MaxHP = e.HP
			w.Enemies = append(w.Enemies, e)
		case tag >= 14 && tag <= 19:
			w.Loot = append(w.Loot, Loot{X: float64(x), Y: float64(y + 8), Kind: tag, Index: i})
		case tag >= 256:
			w.Doors = append(w.Doors, Door{x, y, tag, i, false})
		}
	}
	// Player markers specify the top of a 2-tile actor in the DOS level format.
	for i, t := range src.Objects {
		if t == 255 {
			w.Player.X = float64(i%src.Width*16 + 1)
			w.Player.Y = float64(i/src.Width*16 + 4)
		}
	}
	w.Player.X = clamp(w.Player.X, 1, float64(src.Width*16)-PlayerWidth-1)
	// The supplied level 6 places its start marker inside the outer wall.
	// Find the nearest free position before beginning play.
	if w.bodyInWall(&w.Player.Body) {
		ox, oy := w.Player.X, w.Player.Y
		found := false
		for radius := 1; radius < 32 && !found; radius++ {
			for dx := -radius; dx <= radius && !found; dx++ {
				for dy := -radius; dy <= radius; dy++ {
					if absInt(dx)+absInt(dy) != radius {
						continue
					}
					w.Player.X, w.Player.Y = ox+float64(dx*16), oy+float64(dy*16)
					if w.Player.X >= 16 && w.Player.Y >= 16 && w.Player.X+w.Player.W < float64(src.Width*16-16) && w.Player.Y+w.Player.H < float64(src.Height*16-16) && !w.bodyInWall(&w.Player.Body) {
						found = true
						break
					}
				}
			}
		}
		if !found {
			w.Player.X, w.Player.Y = ox, oy
		}
	}
	w.CameraX = clamp(w.Player.X-100, 0, max(0, float64(src.Width*16-ViewWidth)))
	w.CameraY = clamp(w.Player.Y-95, 0, max(0, float64(src.Height*16-ViewHeight)))
	w.Notify(fmt.Sprintf("LEVEL %d", level+1), 100)
}
func (w *World) Notify(s string, t int) { w.Message = s; w.MessageTime = t }
func (w *World) Event(s string)         { w.Events = append(w.Events, s) }
func (w *World) random() float64 {
	w.seed = w.seed*1664525 + 1013904223
	return float64(w.seed>>8) / float64(1<<24)
}
func (w *World) tile(x, y int) int {
	if x < 0 || x >= w.Map.Width || y < 0 || y >= w.Map.Height {
		return 51
	}
	return w.Map.Tiles[y*w.Map.Width+x]
}
func (w *World) info(x, y int) TileInfo { return w.Data.Tiles[w.tile(x, y)] }
func clamp(x, a, b float64) float64     { return math.Max(a, math.Min(x, b)) }
func sign(x float64) float64 {
	if x < 0 {
		return -1
	}
	return 1
}

// Move separates axes and checks only the boundary crossed, preserving one-way platforms.
func (w *World) Move(b *Body, drop bool) bool {
	hit := false
	dx := b.VX
	steps := max(1, int(math.Ceil(math.Abs(dx)/4)))
	dx /= float64(steps)
	for n := 0; n < steps; n++ {
		old := b.X
		b.X += dx
		if dx > 0 {
			col := int(math.Floor((b.X + b.W - 0.001) / 16))
			from := int(math.Floor((old + b.W - 0.001) / 16))
			for y := int(b.Y / 16); y <= int((b.Y+b.H-0.001)/16); y++ {
				if col > from && w.info(col, y).Left != 0 {
					b.X = float64(col*16) - b.W
					b.VX = 0
					hit = true
					break
				}
			}
		} else if dx < 0 {
			col := int(math.Floor(b.X / 16))
			from := int(math.Floor(old / 16))
			for y := int(b.Y / 16); y <= int((b.Y+b.H-0.001)/16); y++ {
				if col < from && w.info(col, y).Right != 0 {
					b.X = float64((col + 1) * 16)
					b.VX = 0
					hit = true
					break
				}
			}
		}
		if b.VX == 0 {
			break
		}
	}
	dy := b.VY
	steps = max(1, int(math.Ceil(math.Abs(dy)/4)))
	dy /= float64(steps)
	b.Ground = false
	for n := 0; n < steps; n++ {
		old := b.Y
		b.Y += dy
		if dy >= 0 {
			row := int(math.Floor((b.Y + b.H - 0.001) / 16))
			from := int(math.Floor((old + b.H - 0.001) / 16))
			for x := int(b.X / 16); x <= int((b.X+b.W-0.001)/16); x++ {
				t := w.info(x, row)
				if row > from && t.Up != 0 && (!drop || t.Down != 0) {
					b.Y = float64(row*16) - b.H
					b.VY = 0
					b.Ground = true
					break
				}
			}
		} else {
			row := int(math.Floor(b.Y / 16))
			from := int(math.Floor(old / 16))
			for x := int(b.X / 16); x <= int((b.X+b.W-0.001)/16); x++ {
				if row < from && w.info(x, row).Down != 0 {
					b.Y = float64((row + 1) * 16)
					b.VY = 0
					break
				}
			}
		}
		if b.VY == 0 {
			break
		}
	}
	if !b.Ground && b.VY >= 0 && !drop {
		row := int(math.Floor((b.Y + b.H + 0.05) / 16))
		if math.Abs(float64(row*16)-b.Y-b.H) < 0.1 {
			for x := int(b.X / 16); x <= int((b.X+b.W-0.001)/16); x++ {
				if w.info(x, row).Up != 0 {
					b.Ground = true
					b.VY = 0
					break
				}
			}
		}
	}
	return hit
}

func (w *World) Update(in Input) {
	w.Events = w.Events[:0]
	w.Tick++
	if w.MessageTime > 0 {
		w.MessageTime--
	}
	w.effects()
	if w.Won || w.GameOver {
		return
	}
	if w.Transition > 0 {
		w.Transition--
		if w.Transition == 0 {
			w.LoadLevel(w.Level + 1)
			w.Completed = max(w.Completed, w.Level)
		}
		return
	}
	if w.Death > 0 {
		w.Death--
		if w.Death == 0 {
			w.Lives--
			if w.Lives <= 0 {
				w.GameOver = true
				w.Event("gameover")
			} else {
				w.LoadLevel(w.Level)
			}
		}
		return
	}
	p := &w.Player
	if p.Invulnerable > 0 {
		p.Invulnerable--
	}
	if p.Cooldown > 0 {
		p.Cooldown--
	}
	if p.Fire > 0 {
		p.Fire--
	}
	if p.Drop > 0 {
		p.Drop--
	}
	if p.Ground {
		p.Coyote = 5
	} else if p.Coyote > 0 {
		p.Coyote--
	}
	if in.Jump {
		p.JumpBuffer = 6
	} else if p.JumpBuffer > 0 {
		p.JumpBuffer--
	}
	p.VX = 0
	if in.Left {
		p.VX = -WalkSpeed
		p.Facing = -1
	}
	if in.Right {
		p.VX = WalkSpeed
		p.Facing = 1
	}
	if p.VX != 0 {
		p.Walk++
	} else {
		p.Walk = 0
	}
	if p.JumpBuffer > 0 && p.Coyote > 0 {
		p.JumpBuffer = 0
		p.Coyote = 0
		if in.Down {
			p.Drop = 18
			p.Y += 1
			p.Ground = false
		} else {
			p.VY = JumpSpeed
			p.Ground = false
			w.Event("jump")
		}
	}
	p.VY = math.Min(p.VY+Gravity, 6.5)
	w.Move(&p.Body, p.Drop > 0)
	p.AimX = float64(p.Facing)
	p.AimY = 0
	if in.Up {
		p.AimY = -1
	}
	if in.Down && !in.Jump {
		p.AimY = 1
	}
	if p.AimY != 0 && !(in.Left || in.Right) {
		p.AimX = 0
	}
	if in.Shoot && p.Cooldown == 0 && p.Ammo > 0 {
		p.Ammo--
		p.Cooldown = 15
		p.Fire = 6
		p.Reload = 0
		w.Event("shoot")
		dx, dy := p.AimX, p.AimY
		norm := math.Hypot(dx, dy)
		dx /= norm
		dy /= norm
		x, y := p.X+p.W/2+dx*12, p.Y+11+dy*7
		w.Shots = append(w.Shots, Shot{X: x, Y: y, VX: dx * 8, VY: dy * 8, Life: 35})
		w.burst(x, y, 4, 14)
	}
	if p.Ground && p.VX == 0 && !in.Shoot && p.Ammo < 8 {
		p.Reload++
		if p.Reload >= 30 {
			p.Reload = 0
			p.Ammo++
			w.Event("reload")
		}
	} else {
		p.Reload = 0
	}
	if in.Use {
		w.useDoor()
	}
	if w.Transition > 0 || w.Won {
		return
	}
	r := p.Rect()
	for y := int(p.Y / 16); y <= int((p.Y+p.H-1)/16); y++ {
		for x := int(p.X / 16); x <= int((p.X+p.W-1)/16); x++ {
			if w.info(x, y).Behavior == 13 {
				w.killPlayer(9)
			}
		}
	}
	if p.Y > float64(w.Map.Height*16)-32 {
		w.killPlayer(9)
	}
	for i := range w.Loot {
		l := &w.Loot[i]
		if !l.Taken && r.Overlap(Rect{l.X, l.Y, 16, 14}) {
			l.Taken = true
			w.collect(l.Kind, l.X, l.Y)
		}
	}
	w.updateEnemies()
	w.updateShots()
	w.CameraX += (clamp(p.X-150, 0, max(0, float64(w.Map.Width*16-ViewWidth))) - w.CameraX) * 0.15
	targetY := p.Y - 85
	if w.Level == 7 {
		for _, e := range w.Enemies {
			if e.Kind == 12 && e.HP > 0 && math.Abs(p.Y-e.Y) < 220 {
				targetY = math.Max(p.Y+p.H-ViewHeight+8, math.Min(targetY, e.Y-8))
			}
		}
	}
	w.CameraY += (clamp(targetY, 0, max(0, float64(w.Map.Height*16-ViewHeight))) - w.CameraY) * 0.12
}

func (w *World) collect(kind int, x, y float64) {
	if kind == 19 {
		w.Lives++
		w.Event("life")
		w.Popups = append(w.Popups, Popup{x, y, "1UP", 60})
		return
	}
	scores := []int{100, 200, 400, 800, 1600}
	if kind >= 14 && kind <= 18 {
		n := scores[kind-14]
		w.addScore(n)
		w.Event("loot")
		w.Popups = append(w.Popups, Popup{x, y, fmt.Sprint(n), 45})
	}
}
func (w *World) addScore(n int) {
	w.Score += n
	for w.Score >= w.NextLife {
		w.Lives++
		w.NextLife += 10000
		w.Event("life")
		w.Notify("EXTRA LIFE", 90)
	}
}
func (w *World) useDoor() {
	p := &w.Player
	cx, cy := p.Rect().Center()
	for i := range w.Doors {
		d := &w.Doors[i]
		// The object marker is in the middle row of a 2x3-tile door.
		if math.Abs(cx-float64(d.X+8)) > 21 || math.Abs(cy-float64(d.Y+8)) > 30 {
			continue
		}
		if d.Tag&0x8000 != 0 {
			if d.Open {
				return
			}
			d.Open = true
			w.Event("door")
			if d.Tag&0x7fff == 0 {
				w.finish()
				return
			}
			w.collect(13+(d.Tag&0x7fff), float64(d.X), float64(d.Y))
			return
		}
		tx, ty := d.Tag>>8, d.Tag&255
		if tx >= 0 && tx < w.Map.Width && ty >= 0 && ty < w.Map.Height {
			p.X = float64(tx*16 + 1)
			p.Y = float64((ty+2)*16) - p.H
			p.VX = 0
			p.VY = 0
			p.Invulnerable = 60
			w.Event("door")
			w.CameraX = clamp(p.X-150, 0, max(0, float64(w.Map.Width*16-ViewWidth)))
			w.CameraY = clamp(p.Y-85, 0, max(0, float64(w.Map.Height*16-ViewHeight)))
			return
		}
	}
	for y := int(p.Y / 16); y <= int((p.Y+p.H)/16); y++ {
		for x := int(p.X / 16); x <= int((p.X+p.W)/16); x++ {
			if w.info(x, y).Behavior == 99 {
				w.finish()
				return
			}
		}
	}
}
func (w *World) finish() {
	for _, e := range w.Enemies {
		if (e.Kind == 11 || e.Kind == 12) && e.HP > 0 {
			w.Notify("DEFEAT THE BOSS FIRST", 100)
			return
		}
	}
	if w.Level == 7 {
		w.Won = true
		w.Completed = 8
		w.Event("win")
		w.addScore(5000)
	} else {
		w.Transition = 90
		w.Event("level")
		w.addScore(1000)
		w.Notify("LEVEL COMPLETE", 90)
	}
}
func (w *World) killPlayer(kind int) {
	if w.Death > 0 || w.Player.Invulnerable > 0 || w.Transition > 0 {
		return
	}
	w.Death = 120
	w.DeathKind = kind
	w.Event("death")
	w.burst(w.Player.X+7, w.Player.Y+10, 20, 12)
}
func (w *World) KillForTest(kind int) { w.Player.Invulnerable = 0; w.killPlayer(kind) }
func (w *World) burst(x, y float64, count, color int) {
	for i := 0; i < count; i++ {
		w.Particles = append(w.Particles, Particle{x, y, (w.random() - 0.5) * 3, -w.random() * 3, 25 + int(w.random()*25), color})
	}
}
func (w *World) effects() {
	ps := w.Particles[:0]
	for _, p := range w.Particles {
		p.Life--
		if p.Life > 0 {
			p.X += p.VX
			p.Y += p.VY
			p.VY += 0.12
			ps = append(ps, p)
		}
	}
	w.Particles = ps
	pops := w.Popups[:0]
	for _, p := range w.Popups {
		p.Life--
		if p.Life > 0 {
			p.Y -= 0.3
			pops = append(pops, p)
		}
	}
	w.Popups = pops
}

func (w *World) updateEnemies() {
	p := &w.Player
	for i := range w.Enemies {
		e := &w.Enemies[i]
		if e.HP <= 0 {
			if e.Dead > 0 {
				e.Dead--
			}
			continue
		}
		if math.Abs(e.X-p.X) > 260 || math.Abs(e.Y-p.Y) > 210 {
			continue
		}
		e.Active = true
		e.Anim++
		e.Timer--
		dx := p.X - e.X
		switch e.Kind {
		case 3: // Ghosts drift through walls.
			e.VX = sign(dx) * 0.55
			e.VY = sign(p.Y-e.Y) * 0.28
			e.X += e.VX
			e.Y += e.VY
		case 6: // Flame patrols float horizontally.
			e.VX = float64(e.Facing) * 0.8
			e.X += e.VX
			e.Y = e.HomeY + 8 + math.Sin(float64(e.Anim)/18)*8
			if math.Abs(e.X-e.HomeX) > 64 {
				e.Facing = -e.Facing
			}
		case 12: // The Mansion Master floats above the final chamber.
			e.VX = sign(dx) * 0.35
			e.X += e.VX
			targetY := clamp(p.Y-100+math.Sin(float64(e.Anim)/45)*20, e.HomeY-100, e.HomeY+130)
			e.VY = clamp((targetY-e.Y)*0.025, -1, 1)
			e.Y += e.VY
			e.X = clamp(e.X, 48, float64(w.Map.Width*16)-e.W-48)
		case 9: // Ceiling trap drops when Dave approaches.
			if math.Abs(dx) < 24 && p.Y > e.Y {
				e.VY = math.Min(e.VY+0.3, 7)
				e.Y += e.VY
			}
		default:
			speed := 0.5
			switch e.Kind {
			case 2:
				speed = 0.7
			case 4:
				speed = 0.4
			case 5:
				speed = 0.8
			case 7:
				speed = 1.2
			case 11:
				speed = 0.9
			case 12:
				speed = 0.6
			}
			if math.Abs(p.Y-e.Y) < 48 && math.Abs(dx) < 150 {
				e.Facing = int(sign(dx))
			}
			e.VX = float64(e.Facing) * speed
			if e.Ground {
				edge := e.X - 2
				if e.Facing > 0 {
					edge = e.X + e.W + 2
				}
				if w.info(int(edge/16), int((e.Y+e.H+2)/16)).Up == 0 {
					e.Facing = -e.Facing
					e.VX = float64(e.Facing) * speed
				}
				if e.Kind == 4 && e.Timer <= 0 {
					e.VY = -3.5
					e.Timer = 110
				}
				if e.Kind == 7 && e.Timer <= 0 && math.Abs(dx) < 120 {
					e.VY = -4.3
					e.Timer = 150
				}
			}
			e.VY = math.Min(e.VY+Gravity, 6.5)
			if w.Move(&e.Body, false) {
				e.Facing = -e.Facing
			}
		}
		if e.VX < 0 {
			e.Facing = -1
		} else if e.VX > 0 {
			e.Facing = 1
		}
		if (e.Kind == 2 || e.Kind == 12) && e.Timer <= 0 && math.Abs(dx) < 230 {
			sx, sy := e.X+e.W/2, e.Y+e.H*0.3
			px, py := p.Rect().Center()
			vx, vy := px-sx, py-sy
			length := math.Hypot(vx, vy)
			if length > 0 {
				w.Shots = append(w.Shots, Shot{X: sx, Y: sy, VX: vx / length * 2, VY: vy / length * 2, Life: 180, Enemy: true, Kind: e.Kind})
			}
			e.Timer = 150
		}
		if e.Rect().Overlap(p.Rect()) {
			w.killPlayer(e.Kind)
		}
	}
}
func (w *World) updateShots() {
	out := make([]Shot, 0, len(w.Shots))
	for _, s := range w.Shots {
		s.Life--
		if s.Life <= 0 {
			continue
		}
		hit := false
		// Substeps prevent a fast shell skipping a narrow enemy or wall.
		for n := 0; n < 4 && !hit; n++ {
			s.X += s.VX / 4
			s.Y += s.VY / 4
			t := w.info(int(s.X/16), int(s.Y/16))
			if t.Up != 0 && t.Down != 0 && t.Left != 0 && t.Right != 0 {
				hit = true
				break
			}
			r := Rect{s.X - 2, s.Y - 2, 4, 4}
			if s.Enemy {
				if r.Overlap(w.Player.Rect()) {
					w.killPlayer(s.Kind)
					hit = true
				}
			} else {
				// A shell can intercept a thrown knife / energy orb.
				for j := range w.Shots {
					a := &w.Shots[j]
					if a.Enemy && a.Life > 0 && math.Hypot(a.X-s.X, a.Y-s.Y) < 7 {
						a.Life = 0
						hit = true
						break
					}
				}
				if !hit {
					for i := range w.Enemies {
						e := &w.Enemies[i]
						if e.HP <= 0 || !r.Overlap(e.Rect()) {
							continue
						}
						e.HP--
						w.burst(s.X, s.Y, 8, 12)
						hit = true
						w.Event("hit")
						if e.HP <= 0 {
							e.Dead = 24
							score := 100
							if e.Kind >= 11 {
								score = 2000
							}
							w.addScore(score)
							w.Event("enemy")
							w.Popups = append(w.Popups, Popup{e.X, e.Y, fmt.Sprint(score), 45})
							if e.Kind == 11 {
								w.Transition = 120
								w.Notify("FRANK DEFEATED", 120)
							}
							if e.Kind == 12 {
								w.Won = true
								w.Completed = 8
								w.Notify("DELBERT IS SAFE!", 300)
								w.addScore(5000)
								w.Event("win")
							}
						}
						break
					}
				}
			}
		}
		if !hit {
			out = append(out, s)
		} else {
			w.burst(s.X, s.Y, 3, 14)
		}
	}
	w.Shots = out
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
func (w *World) bodyInWall(b *Body) bool {
	for y := int(b.Y / 16); y <= int((b.Y+b.H-0.001)/16); y++ {
		for x := int(b.X / 16); x <= int((b.X+b.W-0.001)/16); x++ {
			t := w.info(x, y)
			if t.Up != 0 && t.Down != 0 && t.Left != 0 && t.Right != 0 {
				return true
			}
		}
	}
	return false
}
