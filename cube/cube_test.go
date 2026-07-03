package cube

import "testing"

func packMoves(codes [5]int) []byte {
	return packMovesC(codes, 0)
}

func packMovesC(codes [5]int, counter byte) []byte {
	var v uint32
	for i, c := range codes {
		v |= uint32(c) << (27 - 5*i)
	}
	b := make([]byte, 20)
	b[0] = 0xA5
	b[11] = counter
	b[12] = byte(v >> 24)
	b[13] = byte(v >> 16)
	b[14] = byte(v >> 8)
	b[15] = byte(v)
	return b
}

func TestHandleMovesParsing(t *testing.T) {
	c := New(WeilongV10AI)
	var got string
	c.OnMove = func(move string) { got = move }

	c.handleNotification(packMoves([5]int{10, 4, 0, 8, 6}))

	if got != "R" {
		t.Errorf("OnMove got %q, want R", got)
	}
	want := [5]string{"R", "U", "F", "L", "D"}
	if c.lastMoves != want {
		t.Errorf("lastMoves = %v, want %v", c.lastMoves, want)
	}
}

func TestHandleMovesRecoversDropped(t *testing.T) {
	c := New(WeilongV10AI)
	var got []string
	c.OnMove = func(m string) { got = append(got, m) }

	c.handleNotification(packMovesC([5]int{4, 0, 0, 0, 0}, 10))
	if len(got) != 1 || got[0] != "U" {
		t.Fatalf("first move: got %v, want [U]", got)
	}

	got = nil
	c.handleNotification(packMovesC([5]int{10, 8, 4, 0, 0}, 12))
	if len(got) != 2 || got[0] != "L" || got[1] != "R" {
		t.Fatalf("dropped-move recovery: got %v, want [L R]", got)
	}
}

func TestHandleMovesIgnoresDuplicate(t *testing.T) {
	c := New(WeilongV10AI)
	var count int
	c.OnMove = func(string) { count++ }

	c.handleNotification(packMovesC([5]int{4, 0, 0, 0, 0}, 7))
	c.handleNotification(packMovesC([5]int{4, 0, 0, 0, 0}, 7))
	if count != 1 {
		t.Fatalf("duplicate packet applied %d moves, want 1", count)
	}
}

func TestHandleMovesCounterWrap(t *testing.T) {
	c := New(WeilongV10AI)
	var count int
	c.OnMove = func(string) { count++ }

	c.handleNotification(packMovesC([5]int{4, 0, 0, 0, 0}, 255))
	count = 0
	c.handleNotification(packMovesC([5]int{4, 4, 0, 0, 0}, 1))
	if count != 2 {
		t.Fatalf("counter wrap 255->1 should apply 2 moves, got %d", count)
	}
}

func TestLastMovesReversed(t *testing.T) {
	c := New(WeilongV10AI)
	c.handleNotification(packMoves([5]int{10, 4, 0, 8, 6}))
	if got := c.LastMoves(); got != "D L F U R " {
		t.Errorf("LastMoves() = %q, want %q", got, "D L F U R ")
	}
}

func TestIsSolved(t *testing.T) {
	c := New(WeilongV10AI)
	if c.IsSolved() {
		t.Error("fresh cube reported solved")
	}
	c.state = weilongSolvedState
	if !c.IsSolved() {
		t.Error("solved state not detected")
	}
}

func TestTrySendNonBlocking(t *testing.T) {
	ch := make(chan int, 1)
	trySend(ch, 1)
	trySend(ch, 2)
	if v := <-ch; v != 1 {
		t.Errorf("got %d, want 1 (first value kept)", v)
	}
	select {
	case v := <-ch:
		t.Errorf("channel should be empty, got %d", v)
	default:
	}
}

func TestDrain(t *testing.T) {
	ch := make(chan int, 1)
	ch <- 42
	drain(ch)
	select {
	case v := <-ch:
		t.Errorf("drain left value %d", v)
	default:
	}
	drain(ch)
}
