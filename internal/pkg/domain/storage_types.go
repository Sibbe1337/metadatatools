package domain

// SignedURLOperation represents the type of operation for a signed URL
type SignedURLOperation string

const (
	SignedURLUpload   SignedURLOperation = "upload"
	SignedURLDownload SignedURLOperation = "download"
)

// StoragePathType represents the type of storage path
type StoragePathType string

const (
	StoragePathTemp StoragePathType = "temp"
	StoragePathPerm StoragePathType = "perm"
)

// String returns the string representation of the storage path type
func (s StoragePathType) String() string {
	return string(s)
}

// StorageError represents a storage-specific error
type StorageError struct {
	Code    string
	Message string
	Op      string
	Err     error
}

// Error implements the error interface
func (e *StorageError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *StorageError) Unwrap() error {
	return e.Err
}

// NewStorageError creates a new storage error
func NewStorageError(code, message, op string, err error) *StorageError {
	return &StorageError{
		Code:    code,
		Message: message,
		Op:      op,
		Err:     err,
	}
}
