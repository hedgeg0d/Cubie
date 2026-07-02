package cube

import "testing"

func packMoves(codes [5]int) []byte {
	var v uint32
	for i, c := range codes {
		v |= uint32(c) << (27 - 5*i)
	}
	b := make([]byte, 20)
	b[0] = 0xA5
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
