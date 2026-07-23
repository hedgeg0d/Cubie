package main

import (
	"fmt"
	"sort"
	"strconv"
)


func itoa(n int) string { return strconv.Itoa(n) }

const timesFile = "times.json"

type SolveEvent struct {
	T    int64  `json:"t"`
	Kind string `json:"kind"`
	Val  string `json:"val"`
}

type Solve struct {
	Ms       int64        `json:"ms"`
	Scramble string       `json:"scramble"`
	Penalty  string       `json:"penalty"`
	At       int64        `json:"at"`
	Events   []SolveEvent `json:"events,omitempty"`
}

func (s Solve) moves() []string {
	out := []string{}
	for _, e := range s.Events {
		if e.Kind == "move" {
			out = append(out, e.Val)
		}
	}
	return out
}

func (s Solve) rotations() []string {
	out := []string{}
	for _, e := range s.Events {
		if e.Kind == "rot" {
			out = append(out, e.Val)
		}
	}
	return out
}

func (s Solve) effectiveMs() (int64, bool) {
	switch s.Penalty {
	case "DNF":
		return 0, false
	case "+2":
		return s.Ms + 2000, true
	default:
		return s.Ms, true
	}
}

func loadSolves() []Solve {
	var solves []Solve
	readJSON(timesFile, &solves)
	return solves
}

func saveSolves(solves []Solve) error {
	return writeJSON(timesFile, solves)
}

func formatMs(ms int64) string {
	if ms < 0 {
		return "--"
	}
	minutes := ms / 60000
	seconds := (ms % 60000) / 1000
	centis := (ms % 1000) / 10
	if minutes > 0 {
		return fmt.Sprintf("%d:%02d.%02d", minutes, seconds, centis)
	}
	return fmt.Sprintf("%d.%02d", seconds, centis)
}

func best(solves []Solve) int64 {
	b := int64(-1)
	for _, s := range solves {
		ms, ok := s.effectiveMs()
		if !ok {
			continue
		}
		if b < 0 || ms < b {
			b = ms
		}
	}
	return b
}

func mean(solves []Solve) int64 {
	var sum, count int64
	for _, s := range solves {
		ms, ok := s.effectiveMs()
		if !ok {
			return -1
		}
		sum += ms
		count++
	}
	if count == 0 {
		return -1
	}
	return sum / count
}

const sessionsFile = "sessions.json"

type TimerSessions struct {
	Active   string            `json:"active"`
	Sessions map[string][]Solve `json:"sessions"`
}

func (s TimerSessions) names() []string {
	out := make([]string, 0, len(s.Sessions))
	for k := range s.Sessions {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func loadSessions() TimerSessions {
	s := TimerSessions{Sessions: map[string][]Solve{}}
	if err := readJSON(sessionsFile, &s); err != nil {
		var solves []Solve
		if readJSON(timesFile, &solves) == nil && len(solves) > 0 {
			s.Sessions["Default"] = solves
			s.Active = "Default"
			saveSessions(s)
		}
		return s
	}
	if s.Sessions == nil {
		s.Sessions = map[string][]Solve{}
	}
	if len(s.Sessions) == 0 {
		s.Sessions["Default"] = nil
		s.Active = "Default"
		saveSessions(s)
	}
	if _, ok := s.Sessions[s.Active]; !ok {
		for name := range s.Sessions {
			s.Active = name
			break
		}
	}
	return s
}

func saveSessions(sessions TimerSessions) error {
	return writeJSON(sessionsFile, sessions)
}

func averageOf(solves []Solve, n int) int64 {
	if len(solves) < n {
		return -1
	}
	window := solves[len(solves)-n:]

	values := make([]int64, 0, n)
	dnf := 0
	for _, s := range window {
		ms, ok := s.effectiveMs()
		if !ok {
			dnf++
			values = append(values, 1<<62)
			continue
		}
		values = append(values, ms)
	}
	if dnf >= 2 {
		return -1
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	trimmed := values[1 : n-1]
	var sum int64
	for _, v := range trimmed {
		sum += v
	}
	return sum / int64(len(trimmed))
}
