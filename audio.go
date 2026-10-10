package main

import (
	"encoding/binary"

	"math"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// Short synthesized effects keep the game independent of DOS audio hardware.
// Original music / PC-speaker recordings are not embedded.
type soundBank struct {
	ctx     *audio.Context
	enabled bool
	players map[string][]*audio.Player
	next    map[string]int
}

func newSoundBank(enabled bool) *soundBank {
	s := &soundBank{enabled: enabled, players: map[string][]*audio.Player{}, next: map[string]int{}}
	if !enabled {
		return s
	}
	s.initialize()
	return s
}
func (s *soundBank) initialize() {
	if s.ctx != nil {
		return
	}
	s.ctx = audio.NewContext(44100)
	specs := map[string][3]float64{"shoot": {160, 40, .13}, "reload": {620, 360, .05}, "jump": {250, 650, .08}, "loot": {600, 1100, .10}, "life": {400, 1400, .28}, "death": {400, 40, .6}, "hit": {180, 50, .07}, "enemy": {250, 70, .12}, "door": {120, 360, .16}, "level": {400, 1000, .4}, "win": {500, 1600, .7}, "gameover": {300, 60, .5}}
	for name, v := range specs {
		pcm := effect(v[0], v[1], v[2], name == "shoot" || name == "hit")
		for n := 0; n < 3; n++ {
			p := s.ctx.NewPlayerFromBytes(pcm)
			p.SetVolume(.24)
			s.players[name] = append(s.players[name], p)
		}
	}
}
func (s *soundBank) toggle() {
	s.enabled = !s.enabled
	if s.enabled {
		s.initialize()
	} else {
		for _, ps := range s.players {
			for _, p := range ps {
				p.Pause()
			}
		}
	}
}
func (s *soundBank) play(name string) {
	if !s.enabled {
		return
	}
	ps := s.players[name]
	if len(ps) == 0 {
		return
	}
	i := s.next[name] % len(ps)
	s.next[name]++
	p := ps[i]
	p.Pause()
	if e := p.Rewind(); e == nil {
		p.Play()
	}
}
func effect(start, end, duration float64, noise bool) []byte {
	count := int(duration * 44100)
	data := make([]byte, count*4)
	phase := 0.0
	seed := uint32(42)
	for i := 0; i < count; i++ {
		t := float64(i) / float64(count)
		phase += (start + (end-start)*t) / 44100
		v := .4
		if math.Mod(phase, 1) > .5 {
			v = -.4
		}
		if noise {
			seed = seed*1664525 + 1013904223
			v = v*.3 + (float64(seed>>16)/65535-.5)*.7
		}
		v *= math.Pow(1-t, 1.6)
		sample := int16(v * 32767)
		binary.LittleEndian.PutUint16(data[i*4:], uint16(sample))
		binary.LittleEndian.PutUint16(data[i*4+2:], uint16(sample))
	}
	return data
}
