package main

import "testing"

func TestPositionParsesUnixFormat(t *testing.T) {
	file, n, ok := position("service/live.go:42:7: [var-naming] type fakeIdP should be fakeIDP")
	if !ok || file != "service/live.go" || n != 42 {
		t.Fatalf("got %q %d %v", file, n, ok)
	}
	if _, _, ok := position("not a finding"); ok {
		t.Fatal("a line without a position must be skipped")
	}
}

func TestWithinCountsEveryLineOfANewFile(t *testing.T) {
	if !within(nil, 10) {
		t.Fatal("a nil span list means the whole file is new")
	}
	spans := []span{{3, 4}, {10, 10}}
	for n, want := range map[int]bool{2: false, 3: true, 4: true, 5: false, 10: true, 11: false} {
		if got := within(spans, n); got != want {
			t.Errorf("line %d: got %v, want %v", n, got, want)
		}
	}
}
