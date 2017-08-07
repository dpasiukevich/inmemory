package inmemory

import (
	"testing"
)

func BenchmarkGet(b *testing.B) {
	b.StopTimer()
	client := setupTestClient()

	_, err := client.Exec("set", []string{"x", "15"})

	if err != nil {
		b.Fatal(err)
	}
	client.cmd = "GET"
	client.args = []string{"x"}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		Get(client)
	}
	b.StopTimer()
}

func BenchmarkSet(b *testing.B) {
	b.StopTimer()

	client := setupTestClient()
	client.cmd = "SET"
	client.args = []string{"x", ""}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		Set(client)
	}
	b.StopTimer()
}

func BenchmarkLPush(b *testing.B) {
	b.StopTimer()

	client := setupTestClient()
	client.cmd = "LPUSH"
	client.args = []string{"x", ""}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		LPush(client)
	}
	b.StopTimer()
}

func BenchmarkLGet(b *testing.B) {
	b.StopTimer()
	client := setupTestClient()

	_, err := client.Exec("lpush", []string{"x", ""})

	if err != nil {
		b.Fatal(err)
	}
	client.cmd = "LGET"
	client.args = []string{"x", "0"}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		LGet(client)
	}
	b.StopTimer()
}

func BenchmarkHSet(b *testing.B) {
	b.StopTimer()
	client := setupTestClient()

	client.cmd = "HSET"
	client.args = []string{"x", "", ""}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		HSet(client)
	}
	b.StopTimer()
}

func BenchmarkHGet(b *testing.B) {
	b.StopTimer()
	client := setupTestClient()

	_, err := client.Exec("hset", []string{"x", "", ""})

	if err != nil {
		b.Fatal(err)
	}
	client.cmd = "HGET"
	client.args = []string{"x", ""}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		HGet(client)
	}
	b.StopTimer()
}
