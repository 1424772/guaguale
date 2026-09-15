package main

import "testing"

func TestEnvHelpersFallback(t *testing.T) {
	t.Setenv("TEST_EMPTY_VALUE", "")
	if envString("TEST_EMPTY_VALUE", "fallback") != "fallback" {
		t.Fatal("envString did not use fallback")
	}
	t.Setenv("TEST_BAD_INT", "bad")
	if envInt("TEST_BAD_INT", 12) != 12 {
		t.Fatal("envInt did not use fallback")
	}
}
