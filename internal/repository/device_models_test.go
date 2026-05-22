package repository

import (
	"context"
	"testing"
)

func TestDeviceModelRepositoryCreatePersistsModelWithImageAndOrder(t *testing.T) {
	repo := NewDeviceModelRepository(&fakeExecutor{
		row: fakeRow{values: []any{
			"model-1",
			"iPhone 15",
			"ios",
			"iOS",
			"17.4",
			"https://example.com/iphone15.png",
			1,
			true,
			[]byte(`{"color":"black"}`),
		}},
	})

	model, err := repo.Create(context.Background(), CreateDeviceModelParams{
		Name:         "iPhone 15",
		OS:           "ios",
		SystemLabel:  "iOS",
		VersionLabel: "17.4",
		ImageURL:     "https://example.com/iphone15.png",
		SortOrder:    1,
		Enabled:      true,
		Metadata:     map[string]any{"color": "black"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if model.ID != "model-1" {
		t.Fatalf("model.ID = %q", model.ID)
	}
	if model.ImageURL != "https://example.com/iphone15.png" {
		t.Fatalf("model.ImageURL = %q", model.ImageURL)
	}
	if model.SortOrder != 1 {
		t.Fatalf("model.SortOrder = %d", model.SortOrder)
	}
}

func TestDeviceModelRepositoryListEnabledReturnsOnlyEnabledModels(t *testing.T) {
	repo := NewDeviceModelRepository(&fakeExecutor{
		rows: fakeRows{values: [][]any{
			{"model-1", "iPhone 15", "ios", "iOS", "17.4", "https://example.com/iphone15.png", 1, true, []byte(`{}`)},
			{"model-2", "Galaxy S24", "android", "Android", "14", "https://example.com/s24.png", 2, true, []byte(`{}`)},
		}},
	})

	items, err := repo.ListEnabled(context.Background())
	if err != nil {
		t.Fatalf("ListEnabled() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d", len(items))
	}
	if items[0].Name != "iPhone 15" {
		t.Fatalf("items[0].Name = %q", items[0].Name)
	}
}

func TestDeviceModelRepositoryUpdateTogglesEnabledAndSortOrder(t *testing.T) {
	repo := NewDeviceModelRepository(&fakeExecutor{
		row: fakeRow{values: []any{
			"model-1",
			"iPhone 15",
			"ios",
			"iOS",
			"17.4",
			"https://example.com/iphone15-v2.png",
			4,
			false,
			[]byte(`{"color":"silver"}`),
		}},
	})

	enabled := false
	sortOrder := 4
	imageURL := "https://example.com/iphone15-v2.png"
	model, err := repo.Update(context.Background(), "model-1", UpdateDeviceModelParams{
		Enabled:   &enabled,
		SortOrder: &sortOrder,
		ImageURL:  &imageURL,
		Metadata:  map[string]any{"color": "silver"},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if model.Enabled {
		t.Fatalf("model.Enabled = %v", model.Enabled)
	}
	if model.SortOrder != 4 {
		t.Fatalf("model.SortOrder = %d", model.SortOrder)
	}
}
