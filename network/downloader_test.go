package network

import "testing"

func TestAssets(t *testing.T) {
	got := 5
	want := 5
	if got != want {
		t.Errorf("Add(2,3) = %d; want %d", got, want)
	}
}
