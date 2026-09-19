package database

import (
	"testing"
	"time"

	"aunefyren/poenskelisten/models"

	"github.com/google/uuid"
)

func createTestNews(t *testing.T, title string, date time.Time) models.News {
	t.Helper()

	news := models.News{
		Title:   title,
		Body:    "Body of " + title,
		Enabled: true,
		Date:    date,
	}
	news.ID = uuid.New()

	created, err := CreateNewsPostInDB(news)
	if err != nil {
		t.Fatalf("failed to create news post: %v", err)
	}

	return created
}

func TestNewsPostsOrderedByDateDesc(t *testing.T) {
	setupTestDB(t)

	older := createTestNews(t, "Older", time.Now().AddDate(0, 0, -2))
	newer := createTestNews(t, "Newer", time.Now())

	posts, err := GetNewsPosts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("expected 2 news posts, got %d", len(posts))
	}
	// Newest first.
	if posts[0].ID != newer.ID || posts[1].ID != older.ID {
		t.Fatalf("expected news ordered newest-first, got %q then %q", posts[0].Title, posts[1].Title)
	}
}

func TestGetUpdateDeleteNewsPost(t *testing.T) {
	setupTestDB(t)

	news := createTestNews(t, "Announcement", time.Now())

	got, err := GetNewsPostByNewsID(news.ID)
	if err != nil {
		t.Fatalf("GetNewsPostByNewsID returned error: %v", err)
	}
	if got.Title != "Announcement" {
		t.Fatalf("expected title 'Announcement', got %q", got.Title)
	}

	got.Title = "Updated"
	if _, err := UpdateNewsPostInDB(got); err != nil {
		t.Fatalf("UpdateNewsPostInDB returned error: %v", err)
	}
	reread, err := GetNewsPostByNewsID(news.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reread.Title != "Updated" {
		t.Fatalf("expected updated title, got %q", reread.Title)
	}

	if err := DeleteNewsPost(news.ID); err != nil {
		t.Fatalf("DeleteNewsPost returned error: %v", err)
	}
	if _, err := GetNewsPostByNewsID(news.ID); err == nil {
		t.Fatalf("expected error looking up disabled news post, got nil")
	}

	// The enabled listing should now be empty.
	posts, err := GetNewsPosts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("expected no news posts after delete, got %d", len(posts))
	}
}

func TestNewsQueriesFailOnClosedDB(t *testing.T) {
	runClosedDBCases(t, map[string]func() error{
		"DeleteNewsPost": func() error {
			return DeleteNewsPost(uuid.New())
		},
		"GetNewsPosts": func() error {
			_, err := GetNewsPosts()
			return err
		},
		"GetNewsPostByNewsID": func() error {
			_, err := GetNewsPostByNewsID(uuid.New())
			return err
		},
		"UpdateNewsPostInDB": func() error {
			_, err := UpdateNewsPostInDB(models.News{})
			return err
		},
		"CreateNewsPostInDB": func() error {
			_, err := CreateNewsPostInDB(models.News{})
			return err
		},
	})
}
