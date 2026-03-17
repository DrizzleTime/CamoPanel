package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"camopanel/server/internal/model"

	"github.com/google/uuid"
)

const websiteMetadataDirName = "sites"

func (a *App) websiteMetadataDir() string {
	return filepath.Join(a.cfg.OpenRestyDataDir, websiteMetadataDirName)
}

func (a *App) ensureWebsiteMetadataDir() error {
	return os.MkdirAll(a.websiteMetadataDir(), 0o755)
}

func (a *App) websiteMetadataPath(name string) string {
	return filepath.Join(a.websiteMetadataDir(), normalizeProjectName(name)+".json")
}

func (a *App) saveWebsite(website model.Website) error {
	if err := a.ensureWebsiteMetadataDir(); err != nil {
		return err
	}

	website = a.normalizeWebsiteRecord(website)
	payload, err := json.MarshalIndent(website, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.websiteMetadataPath(website.Name), payload, 0o644)
}

func (a *App) removeWebsite(website model.Website) error {
	err := os.Remove(a.websiteMetadataPath(website.Name))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (a *App) listWebsites() ([]model.Website, error) {
	entries, err := os.ReadDir(a.websiteMetadataDir())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []model.Website{}, nil
		}
		return nil, err
	}

	websites := make([]model.Website, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		payload, err := os.ReadFile(filepath.Join(a.websiteMetadataDir(), entry.Name()))
		if err != nil {
			return nil, err
		}

		var website model.Website
		if err := json.Unmarshal(payload, &website); err != nil {
			return nil, err
		}
		websites = append(websites, a.normalizeWebsiteRecord(website))
	}

	slices.SortFunc(websites, func(left, right model.Website) int {
		switch {
		case left.CreatedAt.After(right.CreatedAt):
			return -1
		case left.CreatedAt.Before(right.CreatedAt):
			return 1
		default:
			return strings.Compare(left.Name, right.Name)
		}
	})

	return websites, nil
}

func (a *App) findWebsite(websiteID string) (model.Website, error) {
	websites, err := a.listWebsites()
	if err != nil {
		return model.Website{}, err
	}
	for _, website := range websites {
		if website.ID == websiteID {
			return website, nil
		}
	}
	return model.Website{}, errors.New("站点不存在")
}

func (a *App) normalizeWebsiteRecord(website model.Website) model.Website {
	now := time.Now().UTC()
	website.ID = strings.TrimSpace(website.ID)
	if website.ID == "" {
		website.ID = uuid.NewString()
	}
	website.Name = normalizeProjectName(website.Name)
	if website.SiteMode == "" {
		website.SiteMode = website.Type
	}
	if website.Type == "" {
		website.Type = website.SiteMode
	}
	if website.Status == "" {
		website.Status = "ready"
	}
	if website.ConfigPath == "" && website.Name != "" {
		website.ConfigPath = filepath.Join(a.cfg.OpenRestyDataDir, "conf.d", website.Name+".conf")
	}
	if website.CreatedAt.IsZero() {
		if website.UpdatedAt.IsZero() {
			website.CreatedAt = now
		} else {
			website.CreatedAt = website.UpdatedAt.UTC()
		}
	} else {
		website.CreatedAt = website.CreatedAt.UTC()
	}
	if website.UpdatedAt.IsZero() {
		website.UpdatedAt = website.CreatedAt
	} else {
		website.UpdatedAt = website.UpdatedAt.UTC()
	}
	return website
}
