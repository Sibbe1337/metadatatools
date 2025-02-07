package storage

import (
	"fmt"
	"path/filepath"
	"time"

	"metadatatool/internal/pkg/domain"
)

// generateKey generates a storage key
func generateKey(pathType domain.StoragePathType, filename string) string {
	ext := filepath.Ext(filename)
	timestamp := time.Now().UTC().Format("20060102150405")
	return fmt.Sprintf("%s/%s/%s%s", pathType.String(), timestamp[:8], timestamp, ext)
}
