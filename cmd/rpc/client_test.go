package rpc

import "testing"

func TestClientURLEmptyAdminFlag(t *testing.T) {
	client := NewClient("http://query", "http://admin")
	emptyAdmin := []bool{}

	got := client.url(LogsRouteName, "", emptyAdmin...)
	want := "http://query" + routePaths[LogsRouteName].Path
	if got != want {
		t.Fatalf("url with empty admin flag = %q, want %q", got, want)
	}
}

func TestClientURLAdminFlag(t *testing.T) {
	client := NewClient("http://query", "http://admin")

	got := client.url(LogsRouteName, "", true)
	want := "http://admin" + routePaths[LogsRouteName].Path
	if got != want {
		t.Fatalf("url with admin flag = %q, want %q", got, want)
	}
}
