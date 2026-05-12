package repository

import (
	"context"
	"strings"
	"testing"
)

func TestProxyRepositoryCreate(t *testing.T) {
	db := &fakeExecutor{
		row: fakeRow{values: []any{
			"proxy-id",
			"proxy1",
			"proxy.example.com",
			8000,
			"http",
			"user",
			"PROXY_01_PASSWORD",
			true,
			20,
			0,
			[]byte(`{"testRunId":"test-run"}`),
		}},
	}
	repo := NewProxyRepository(db)

	proxy, err := repo.Create(context.Background(), CreateProxyParams{
		Name:               "proxy1",
		Host:               "proxy.example.com",
		Port:               8000,
		Protocol:           "http",
		Username:           stringPtr("user"),
		PasswordSecretName: stringPtr("PROXY_01_PASSWORD"),
		Enabled:            true,
		MaxInstances:       intPtr(20),
		Metadata:           map[string]any{"testRunId": "test-run"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if proxy.ID != "proxy-id" {
		t.Fatalf("ID = %q", proxy.ID)
	}
	if proxy.Port != 8000 {
		t.Fatalf("Port = %d", proxy.Port)
	}
	if proxy.MaxInstances == nil || *proxy.MaxInstances != 20 {
		t.Fatalf("MaxInstances = %v", proxy.MaxInstances)
	}
}

func TestProxyRepositoryList(t *testing.T) {
	db := &fakeExecutor{
		rows: fakeRows{values: [][]any{
			{
				"proxy-id",
				"proxy1",
				"proxy.example.com",
				8000,
				"http",
				"user",
				"PROXY_01_PASSWORD",
				true,
				20,
				3,
				[]byte(`{"testRunId":"test-run"}`),
			},
		}},
	}
	repo := NewProxyRepository(db)

	items, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items len = %d", len(items))
	}
	if items[0].ID != "proxy-id" {
		t.Fatalf("ID = %q", items[0].ID)
	}
	if items[0].CurrentInstances != 3 {
		t.Fatalf("CurrentInstances = %d", items[0].CurrentInstances)
	}
	if !strings.Contains(db.lastSQL, "from public.proxies") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestProxyRepositoryUpsert(t *testing.T) {
	db := &fakeExecutor{
		row: fakeRow{values: []any{
			"proxy-id",
			"proxy1",
			"proxy.example.com",
			8000,
			"http",
			"user",
			"PROXY_01_PASSWORD",
			true,
			20,
			0,
			[]byte(`{"source":"fbflix"}`),
		}},
	}
	repo := NewProxyRepository(db)

	proxy, err := repo.Upsert(context.Background(), CreateProxyParams{
		Name:               "proxy1",
		Host:               "proxy.example.com",
		Port:               8000,
		Protocol:           "http",
		Username:           stringPtr("user"),
		PasswordSecretName: stringPtr("PROXY_01_PASSWORD"),
		Enabled:            true,
		MaxInstances:       intPtr(20),
		Metadata:           map[string]any{"source": "fbflix"},
	})
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	if proxy.ID != "proxy-id" {
		t.Fatalf("ID = %q", proxy.ID)
	}
	if !strings.Contains(db.lastSQL, "on conflict (host, port)") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func stringPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}
