package main

import "testing"

func TestParseCommand(t *testing.T) {
	path, err := parseCommand([]string{"services", "exporter", "serve", "--config", "host.toml"})
	if err != nil {
		t.Fatalf("parseCommand() error=%v", err)
	}
	if path != "host.toml" {
		t.Fatalf("path=%q, want host.toml", path)
	}
}

func TestParseCommandRejectsWrongOrder(t *testing.T) {
	_, err := parseCommand([]string{"exporter", "services", "serve"})
	if err == nil {
		t.Fatal("parseCommand() error=nil, want command error")
	}
}
