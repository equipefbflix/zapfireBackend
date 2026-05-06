package repository

import (
	"context"
	"strings"
	"testing"
)

func TestEvolutionServerRepositoryCreate(t *testing.T) {
	db := &fakeExecutor{
		row: fakeRow{values: []any{
			"server-id",
			"evo1",
			"https://evo.example.com",
			"EVOLUTION_EVO1_API_KEY",
			true,
			1,
			5,
			"healthy",
			[]byte(`{"testRunId":"test-run"}`),
		}},
	}
	repo := NewEvolutionServerRepository(db)

	server, err := repo.Create(context.Background(), CreateEvolutionServerParams{
		Name:              "evo1",
		BaseURL:           "https://evo.example.com",
		APIKeySecretName:  "EVOLUTION_EVO1_API_KEY",
		Enabled:           true,
		Weight:            1,
		MaxConcurrentJobs: 5,
		Metadata:          map[string]any{"testRunId": "test-run"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if server.ID != "server-id" {
		t.Fatalf("ID = %q", server.ID)
	}
	if server.HealthStatus != "healthy" {
		t.Fatalf("HealthStatus = %q", server.HealthStatus)
	}
}

func TestEvolutionServerRepositoryList(t *testing.T) {
	db := &fakeExecutor{
		rows: fakeRows{values: [][]any{
			{"server-id", "evo1", "https://evo.example.com", "EVOLUTION_EVO1_API_KEY", true, 2, 7, "healthy", []byte(`{"testRunId":"test-run"}`)},
		}},
	}
	repo := NewEvolutionServerRepository(db)

	servers, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(servers) != 1 {
		t.Fatalf("servers len = %d", len(servers))
	}
	if servers[0].Weight != 2 {
		t.Fatalf("Weight = %d", servers[0].Weight)
	}
	if !strings.Contains(db.lastSQL, "from public.evolution_servers") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestEvolutionServerRepositoryGetByID(t *testing.T) {
	db := &fakeExecutor{
		row: fakeRow{values: []any{
			"server-id",
			"evo1",
			"https://evo.example.com",
			"EVOLUTION_EVO1_API_KEY",
			true,
			2,
			7,
			"healthy",
			[]byte(`{"testRunId":"test-run"}`),
		}},
	}
	repo := NewEvolutionServerRepository(db)

	server, err := repo.GetByID(context.Background(), "server-id")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if server.ID != "server-id" {
		t.Fatalf("ID = %q", server.ID)
	}
	if db.lastArgs[0] != "server-id" {
		t.Fatalf("id arg = %v", db.lastArgs[0])
	}
	if !strings.Contains(db.lastSQL, "where id = $1") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestEvolutionServerRepositoryListEnabled(t *testing.T) {
	db := &fakeExecutor{
		rows: fakeRows{values: [][]any{
			{"server-id", "evo1", "https://evo.example.com", "EVOLUTION_EVO1_API_KEY", true, 1, 5, "healthy", []byte(`{}`)},
		}},
	}
	repo := NewEvolutionServerRepository(db)

	servers, err := repo.ListEnabled(context.Background())
	if err != nil {
		t.Fatalf("ListEnabled() error = %v", err)
	}

	if len(servers) != 1 {
		t.Fatalf("servers len = %d", len(servers))
	}
	if servers[0].Name != "evo1" {
		t.Fatalf("server name = %q", servers[0].Name)
	}
}
