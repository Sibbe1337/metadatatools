package validator

import (
	"metadatatool/internal/pkg/domain"

	"github.com/go-playground/validator/v10"
)

type Validator struct {
	validate *validator.Validate
}

func NewValidator() *Validator {
	return &Validator{
		validate: validator.New(),
	}
}

func (v *Validator) Validate(track *domain.Track) domain.ValidationResult {
	err := v.validate.Struct(track)
	if err == nil {
		return domain.ValidationResult{IsValid: true}
	}

	var errors []domain.ValidationError
	for _, err := range err.(validator.ValidationErrors) {
		errors = append(errors, domain.ValidationError{
			Field:   err.Field(),
			Message: err.Error(),
		})
	}

	return domain.ValidationResult{
		IsValid: false,
		Errors:  errors,
	}
}

func (v *Validator) Struct(s interface{}) error {
	return v.validate.Struct(s)
}

func (v *Validator) Var(field interface{}, tag string) error {
	return v.validate.Var(field, tag)
}

func (v *Validator) RegisterValidation(tag string, fn validator.Func) error {
	return v.validate.RegisterValidation(tag, fn)
}
