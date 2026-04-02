package repo

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	websitesdomain "camopanel/server/internal/modules/websites/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WebsiteRepository struct {
	root string
}

func NewWebsiteRepository(openRestyDataDir string) *WebsiteRepository {
	return &WebsiteRepository{root: filepath.Join(openRestyDataDir, "sites")}
}

func (r *WebsiteRepository) List(_ context.Context) ([]websitesdomain.Website, error) {
	entries, err := os.ReadDir(r.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []websitesdomain.Website{}, nil
		}
		return nil, err
	}

	items := make([]websitesdomain.Website, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		payload, err := os.ReadFile(filepath.Join(r.root, entry.Name()))
		if err != nil {
			return nil, err
		}
		var record WebsiteRecord
		if err := json.Unmarshal(payload, &record); err != nil {
			return nil, err
		}
		items = append(items, toWebsiteDomain(record))
	}

	slices.SortFunc(items, func(left, right websitesdomain.Website) int {
		switch {
		case left.CreatedAt.After(right.CreatedAt):
			return -1
		case left.CreatedAt.Before(right.CreatedAt):
			return 1
		default:
			return strings.Compare(left.Name, right.Name)
		}
	})
	return items, nil
}

func (r *WebsiteRepository) FindByID(ctx context.Context, websiteID string) (websitesdomain.Website, error) {
	items, err := r.List(ctx)
	if err != nil {
		return websitesdomain.Website{}, err
	}
	for _, item := range items {
		if item.ID == websiteID {
			return item, nil
		}
	}
	return websitesdomain.Website{}, websitesdomain.ErrWebsiteNotFound
}

func (r *WebsiteRepository) Save(_ context.Context, website websitesdomain.Website) error {
	if err := os.MkdirAll(r.root, 0o755); err != nil {
		return err
	}
	record := fromWebsiteDomain(website)
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	if record.CreatedAt.IsZero() {
		now := time.Now().UTC()
		record.CreatedAt = now
		record.UpdatedAt = now
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = time.Now().UTC()
	}
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(r.root, strings.ToLower(strings.TrimSpace(record.Name))+".json"), raw, 0o644)
}

func (r *WebsiteRepository) Delete(_ context.Context, websiteID string) error {
	items, err := r.List(context.Background())
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.ID == websiteID {
			err := os.Remove(filepath.Join(r.root, strings.ToLower(strings.TrimSpace(item.Name))+".json"))
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			return nil
		}
	}
	return nil
}

type CertificateRepository struct {
	db *gorm.DB
}

func NewCertificateRepository(db *gorm.DB) *CertificateRepository {
	return &CertificateRepository{db: db}
}

func (r *CertificateRepository) List(ctx context.Context) ([]websitesdomain.Certificate, error) {
	var records []CertificateRecord
	if err := r.db.WithContext(ctx).Order("updated_at desc").Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]websitesdomain.Certificate, 0, len(records))
	for _, item := range records {
		items = append(items, toCertificateDomain(item))
	}
	return items, nil
}

func (r *CertificateRepository) FindByID(ctx context.Context, certificateID string) (websitesdomain.Certificate, error) {
	var record CertificateRecord
	if err := r.db.WithContext(ctx).First(&record, "id = ?", certificateID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return websitesdomain.Certificate{}, websitesdomain.ErrCertificateNotFound
		}
		return websitesdomain.Certificate{}, err
	}
	return toCertificateDomain(record), nil
}

func (r *CertificateRepository) FindByDomain(ctx context.Context, domain string) (websitesdomain.Certificate, error) {
	var record CertificateRecord
	if err := r.db.WithContext(ctx).First(&record, "domain = ?", domain).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return websitesdomain.Certificate{}, websitesdomain.ErrCertificateNotFound
		}
		return websitesdomain.Certificate{}, err
	}
	return toCertificateDomain(record), nil
}

func (r *CertificateRepository) Save(ctx context.Context, item websitesdomain.Certificate) error {
	return r.db.WithContext(ctx).Save(fromCertificateDomain(item)).Error
}

func (r *CertificateRepository) Delete(ctx context.Context, certificateID string) error {
	return r.db.WithContext(ctx).Delete(&CertificateRecord{}, "id = ?", certificateID).Error
}

func toWebsiteDomain(record WebsiteRecord) websitesdomain.Website {
	return websitesdomain.Website{
		ID:            record.ID,
		Name:          record.Name,
		Type:          record.Type,
		Domain:        record.Domain,
		Domains:       record.Domains,
		RootPath:      record.RootPath,
		IndexFiles:    record.IndexFiles,
		ProxyPass:     record.ProxyPass,
		PHPProjectID:  record.PHPProjectID,
		PHPPort:       record.PHPPort,
		RewriteMode:   record.RewriteMode,
		RewritePreset: record.RewritePreset,
		RewriteRules:  record.RewriteRules,
		ConfigPath:    record.ConfigPath,
		Status:        record.Status,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
	}
}

func fromWebsiteDomain(item websitesdomain.Website) WebsiteRecord {
	return WebsiteRecord{
		ID:            item.ID,
		Name:          item.Name,
		Type:          item.Type,
		Domain:        item.Domain,
		Domains:       item.Domains,
		RootPath:      item.RootPath,
		IndexFiles:    item.IndexFiles,
		ProxyPass:     item.ProxyPass,
		PHPProjectID:  item.PHPProjectID,
		PHPPort:       item.PHPPort,
		RewriteMode:   item.RewriteMode,
		RewritePreset: item.RewritePreset,
		RewriteRules:  item.RewriteRules,
		ConfigPath:    item.ConfigPath,
		Status:        item.Status,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}

func toCertificateDomain(item CertificateRecord) websitesdomain.Certificate {
	return websitesdomain.Certificate{
		ID:             item.ID,
		Domain:         item.Domain,
		Email:          item.Email,
		Provider:       item.Provider,
		Status:         item.Status,
		FullchainPath:  item.FullchainPath,
		PrivateKeyPath: item.PrivateKeyPath,
		LastError:      item.LastError,
		ExpiresAt:      item.ExpiresAt,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

func fromCertificateDomain(item websitesdomain.Certificate) CertificateRecord {
	return CertificateRecord{
		ID:             item.ID,
		Domain:         item.Domain,
		Email:          item.Email,
		Provider:       item.Provider,
		Status:         item.Status,
		FullchainPath:  item.FullchainPath,
		PrivateKeyPath: item.PrivateKeyPath,
		LastError:      item.LastError,
		ExpiresAt:      item.ExpiresAt,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}
