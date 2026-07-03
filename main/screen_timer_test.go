package main

import "testing"

func TestScrambleStepSingle(t *testing.T) {
	half, wrong, adv := scrambleStep("U", "U", "", nil)
	if !adv || half != "" || len(wrong) != 0 {
		t.Fatalf("correct single should advance clean, got half=%q wrong=%v adv=%v", half, wrong, adv)
	}
	half, wrong, adv = scrambleStep("U", "R", "", nil)
	if adv || len(wrong) != 1 || wrong[0] != "R" {
		t.Fatalf("wrong single should push, got half=%q wrong=%v adv=%v", half, wrong, adv)
	}
}

func TestScrambleStepDoubleClockwise(t *testing.T) {
	half, wrong, adv := scrambleStep("U2", "U", "", nil)
	if adv || half != "U" || len(wrong) != 0 {
		t.Fatalf("first quarter should record half, got half=%q adv=%v", half, adv)
	}
	half, wrong, adv = scrambleStep("U2", "U", half, wrong)
	if !adv || half != "" {
		t.Fatalf("second same-direction quarter should advance, got half=%q adv=%v", half, adv)
	}
}

func TestScrambleStepDoubleCounter(t *testing.T) {
	half, wrong, adv := scrambleStep("U2", "U'", "", nil)
	if adv || half != "U'" {
		t.Fatalf("counter first quarter should record U', got half=%q adv=%v", half, adv)
	}
	_, _, adv = scrambleStep("U2", "U'", half, wrong)
	if !adv {
		t.Fatalf("second U' should complete the double")
	}
}

func TestScrambleStepDoubleUndoHalf(t *testing.T) {
	half, wrong, _ := scrambleStep("U2", "U", "", nil)
	half, wrong, adv := scrambleStep("U2", "U'", half, wrong)
	if adv || half != "" || len(wrong) != 0 {
		t.Fatalf("reversing the first quarter should reset to full double, got half=%q wrong=%v adv=%v", half, wrong, adv)
	}
}

func TestScrambleStepDoubleWrongThenUndo(t *testing.T) {
	half, wrong, _ := scrambleStep("U2", "U", "", nil)
	half, wrong, _ = scrambleStep("U2", "R", half, wrong)
	if half != "U" || len(wrong) != 1 {
		t.Fatalf("wrong move mid-double should push and keep half, got half=%q wrong=%v", half, wrong)
	}
	half, wrong, adv := scrambleStep("U2", "R'", half, wrong)
	if adv || half != "U" || len(wrong) != 0 {
		t.Fatalf("undoing wrong move should keep half and clear stack, got half=%q wrong=%v", half, wrong)
	}
}
