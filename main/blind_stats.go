package main

const attemptsFile = "attempts.json"

type BlindAttempt struct {
	MemoMs   int64  `json:"memo_ms"`
	ExecMs   int64  `json:"exec_ms"`
	Success  bool   `json:"success"`
	Scramble string `json:"scramble"`
	At       int64  `json:"at"`
}

func loadAttempts() []BlindAttempt {
	var attempts []BlindAttempt
	readJSON(attemptsFile, &attempts)
	return attempts
}

func saveAttempts(attempts []BlindAttempt) error {
	return writeJSON(attemptsFile, attempts)
}

func successRate(attempts []BlindAttempt) int {
	if len(attempts) == 0 {
		return 0
	}
	ok := 0
	for _, a := range attempts {
		if a.Success {
			ok++
		}
	}
	return ok * 100 / len(attempts)
}

func successCount(attempts []BlindAttempt) int {
	ok := 0
	for _, a := range attempts {
		if a.Success {
			ok++
		}
	}
	return ok
}

func bestTotal(attempts []BlindAttempt) int64 {
	b := int64(-1)
	for _, a := range attempts {
		if !a.Success {
			continue
		}
		total := a.MemoMs + a.ExecMs
		if b < 0 || total < b {
			b = total
		}
	}
	return b
}
