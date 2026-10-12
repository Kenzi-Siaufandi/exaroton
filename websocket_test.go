package exaroton

import (
	"context"
	"testing"
	"time"
)

func TestWSStatus(t *testing.T) {
	serverID := requireServerID(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	server := client.Server(serverID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ws, err := server.OpenWebSocket(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	go func() {
		for err := range ws.ErrorChan {
			t.Error(err)
		}
	}()

	for msg := range ws.StatusChan {
		t.Log(msg)
	}
}

func TestWSConsole(t *testing.T) {
	serverID := requireServerID(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	server := client.Server(serverID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ws, err := server.OpenWebSocket(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	err = ws.SubscribeConsole(ctx, 3)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for err := range ws.ErrorChan {
			t.Error(err)
		}
	}()

	for line := range ws.ConsoleChan {
		t.Log(line)
	}
}

func TestWSTick(t *testing.T) {
	serverID := requireServerID(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	server := client.Server(serverID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ws, err := server.OpenWebSocket(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	err = ws.SubscribeTick(ctx)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for err := range ws.ErrorChan {
			t.Error(err)
		}
	}()

	for tick := range ws.TickChan {
		t.Log(tick)
	}
}

func TestWSStats(t *testing.T) {
	serverID := requireServerID(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	server := client.Server(serverID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ws, err := server.OpenWebSocket(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	err = ws.SubscribeStats(ctx)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for err := range ws.ErrorChan {
			t.Error(err)
		}
	}()

	for stats := range ws.StatsChan {
		t.Log(stats)
	}
}

func TestWSHeap(t *testing.T) {
	serverID := requireServerID(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	server := client.Server(serverID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ws, err := server.OpenWebSocket(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	err = ws.SubscribeHeap(ctx)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for err := range ws.ErrorChan {
			t.Error(err)
		}
	}()

	for heap := range ws.HeapChan {
		t.Log(heap)
	}
}

func TestWSManagement(t *testing.T) {
	serverID := requireServerID(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	server := client.Server(serverID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ws, err := server.OpenWebSocket(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	if err := ws.SubscribeManagement(ctx); err != nil {
		t.Fatal(err)
	}

	players, err := ws.GetAllowlist(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Old: %v", players)

	newPlayers := []Player{
		{Name: "Jiternos", UUID: "981853f1-a3f4-4f5a-808a-1a0ff47a5110"},
	}
	if newResult, err := ws.AddAllowlist(ctx, newPlayers); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("New: %v", newResult)
	}
}
