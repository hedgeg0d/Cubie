package main

import (
	"image/color"
	"sort"
	"strconv"
	"strings"

	"cubie/cubestate"
)

const letteringProfilesFile = "lettering_profiles.json"

var letterFaces = []string{"U", "L", "F", "R", "B", "D"}

var faceFill = map[string]color.Color{
	"U": color.RGBA{0xEC, 0xEC, 0xF0, 0xFF},
	"L": color.RGBA{0xF0, 0x8A, 0x24, 0xFF},
	"F": color.RGBA{0x2E, 0xA0, 0x44, 0xFF},
	"R": color.RGBA{0xC0, 0x2A, 0x2A, 0xFF},
	"B": color.RGBA{0x22, 0x5A, 0xC0, 0xFF},
	"D": color.RGBA{0xE8, 0xD0, 0x30, 0xFF},
}

func faceTextColor(face string) color.Color {
	switch face {
	case "U", "D":
		return color.RGBA{0x12, 0x14, 0x1C, 0xFF}
	default:
		return color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
	}
}

func stickerKey(face string, idx int) string { return face + strconv.Itoa(idx) }

type LetteringScheme map[string]string

func defaultScheme() LetteringScheme {
	base := map[string]byte{"U": 'A', "L": 'E', "F": 'I', "R": 'M', "B": 'Q', "D": 'U'}
	corners := []int{0, 2, 8, 6}
	edges := []int{1, 5, 7, 3}
	s := LetteringScheme{}
	for _, f := range letterFaces {
		for k, idx := range corners {
			s[stickerKey(f, idx)] = string(base[f] + byte(k))
		}
		for k, idx := range edges {
			s[stickerKey(f, idx)] = string(base[f] + byte(k))
		}
	}
	return s
}

type LetteringProfiles struct {
	Active   string                      `json:"active"`
	Profiles map[string]LetteringScheme `json:"profiles"`
}

func loadLetteringProfiles() LetteringProfiles {
	p := LetteringProfiles{Profiles: map[string]LetteringScheme{}}
	readJSON(letteringProfilesFile, &p)
	if p.Profiles == nil {
		p.Profiles = map[string]LetteringScheme{}
	}
	if len(p.Profiles) == 0 {
		p.Profiles["Speffz"] = defaultScheme()
		p.Active = "Speffz"
	}
	for name, s := range p.Profiles {
		if s == nil {
			p.Profiles[name] = LetteringScheme{}
		}
	}
	if _, ok := p.Profiles[p.Active]; !ok {
		for name := range p.Profiles {
			p.Active = name
			break
		}
	}
	return p
}

func (s LetteringScheme) symbol(f cubestate.Facelet) string {
	v := s[f.Face+strconv.Itoa(f.Idx)]
	if v == "" {
		return "·"
	}
	return v
}

func memoText(scheme LetteringScheme, scramble []string) (string, string) {
	r := cubestate.Memo(scramble)
	corners := make([]string, len(r.Corners))
	for i, f := range r.Corners {
		corners[i] = scheme.symbol(f)
	}
	edges := make([]string, len(r.Edges))
	for i, f := range r.Edges {
		edges[i] = scheme.symbol(f)
	}
	return strings.Join(corners, " "), strings.Join(edges, " ")
}

func (p LetteringProfiles) names() []string {
	out := make([]string, 0, len(p.Profiles))
	for k := range p.Profiles {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
