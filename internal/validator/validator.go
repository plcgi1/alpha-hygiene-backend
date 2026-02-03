package validator

import (
	"github.com/go-playground/validator/v10"
)

// NewValidator creates a new validator instance with custom validators
func NewValidator() (*validator.Validate, error) {
	validate := validator.New()

	// Custom validator for Ethereum addresses
	err := validate.RegisterValidation("eth_addr", func(fl validator.FieldLevel) bool {
		addr := fl.Field().String()
		if len(addr) != 42 {
			return false
		}
		if addr[:2] != "0x" {
			return false
		}
		// Check for hex characters only
		for _, char := range addr[2:] {
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
				return false
			}
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	return validate, nil
}
