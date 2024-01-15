package utils

import "testing"

func TestGetRandomStringConsecutively(t *testing.T) {
	random1 := GetRandomString(32)
	random2 := GetRandomString(32)

	if random1 == random2 {
		t.Fatal("random string not randomized between consecutive calls")
	}
	if len(random1) != 32 {
		t.Fatalf("random string has wrong length: Expected 32 got %v", len(random1))
	}
	if len(random2) != 32 {
		t.Fatalf("random string has wrong length: Expected 32 got %v", len(random2))
	}
}
