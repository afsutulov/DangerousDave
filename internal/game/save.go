package game

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

// Save stores the complete simulation state at the current point, not just a level number.
type Save struct {
	Version int
	World   World
}

func (w *World) Save(path string) error {
	if w.Death > 0 || w.GameOver || w.Won || w.Transition > 0 {
		return fmt.Errorf("save during active gameplay")
	}
	copy := *w
	copy.Data = nil
	copy.Events = nil
	b, e := json.Marshal(Save{1, copy})
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(path), 0755); e != nil {
		return e
	}
	tmp, e := os.CreateTemp(filepath.Dir(path), "save-*.tmp")
	if e != nil {
		return e
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, e = tmp.Write(b); e != nil {
		tmp.Close()
		return e
	}
	if e = tmp.Close(); e != nil {
		return e
	}
	// Keep the existing slot intact if replacement fails.
	return os.Rename(name, path)
}
func ReadSave(d *Data, path string) (*World, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	if len(b) > 2<<20 {
		return nil, fmt.Errorf("save is too large")
	}
	var s Save
	if e = json.Unmarshal(b, &s); e != nil {
		return nil, e
	}
	w := &s.World
	if s.Version != 1 || w.Level < 0 || w.Level >= len(d.Levels) || w.Lives < 1 || w.NextLife < 1 || w.Score < 0 {
		return nil, fmt.Errorf("invalid save")
	}
	src := d.Levels[w.Level]
	if w.Map.Width != src.Width || w.Map.Height != src.Height || len(w.Map.Tiles) != len(src.Tiles) || len(w.Map.Objects) != len(src.Objects) {
		return nil, fmt.Errorf("invalid saved map")
	}
	for _, t := range w.Map.Tiles {
		if t < 0 || t >= len(d.Tiles) {
			return nil, fmt.Errorf("invalid saved tile")
		}
	}
	p := w.Player
	if math.IsNaN(p.X) || math.IsNaN(p.Y) || p.X < 0 || p.Y < 0 || p.X > float64(src.Width*16) || p.Y > float64(src.Height*16) || p.W != PlayerWidth || p.H != PlayerHeight || p.Ammo < 0 || p.Ammo > 8 {
		return nil, fmt.Errorf("invalid player state")
	}
	if len(w.Enemies) > 500 || len(w.Shots) > 500 || len(w.Particles) > 5000 || len(w.Doors) > 500 {
		return nil, fmt.Errorf("invalid saved entities")
	}
	for _, e := range w.Enemies {
		if e.Kind < 1 || e.Kind > 12 || e.MaxHP < 1 || e.HP > e.MaxHP {
			return nil, fmt.Errorf("invalid saved enemy")
		}
	}
	for _, door := range w.Doors {
		if door.Index < 0 || door.Index >= len(src.Objects) || door.Tag != src.Objects[door.Index] {
			return nil, fmt.Errorf("invalid saved door")
		}
	}
	w.Data = d
	w.seed = 0x1971
	w.Events = nil
	w.Notify("GAME LOADED", 90)
	return w, nil
}
