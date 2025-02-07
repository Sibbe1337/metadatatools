package domain

// ValidationResult represents the result of validating track metadata.
type ValidationResult struct {
	// Valid indicates if the metadata is valid
	Valid bool

	// Confidence is the AI model's confidence in the validation (0-1)
	Confidence float64

	// Issues contains any validation issues found
	Issues []ValidationIssue
}

// ValidationIssue represents a single validation issue found.
type ValidationIssue struct {
	// Field is the name of the field with the issue
	Field string

	// Message describes the validation issue
	Message string

	// Severity indicates how serious the issue is (1-5)
	Severity int
}
