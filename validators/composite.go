package validators

import (
	"github.com/ha1tch/queryfy"
)

// AndValidator combines multiple validators with AND logic.
// All validators must pass for the value to be valid.
type AndValidator struct {
	validators []queryfy.ValidatorFunc
}

// NewAndValidator creates a validator that requires all sub-validators to pass.
func NewAndValidator(validators ...queryfy.ValidatorFunc) *AndValidator {
	return &AndValidator{validators: validators}
}

// Validate runs all validators and collects all errors.
func (v *AndValidator) Validate(value interface{}) error {
	var errors []queryfy.FieldError

	for _, validator := range v.validators {
		if err := validator(value); err != nil {
			if fieldErr, ok := err.(*queryfy.FieldError); ok {
				errors = append(errors, *fieldErr)
			} else if validationErr, ok := err.(*queryfy.ValidationError); ok {
				errors = append(errors, validationErr.Errors...)
			} else {
				errors = append(errors, queryfy.FieldError{
					Message: err.Error(),
					Value:   value,
				})
			}
		}
	}

	if len(errors) > 0 {
		return &queryfy.ValidationError{Errors: errors}
	}

	return nil
}

// OrValidator combines multiple validators with OR logic.
// At least one validator must pass for the value to be valid.
type OrValidator struct {
	validators []queryfy.ValidatorFunc
}

// NewOrValidator creates a validator that requires at least one sub-validator to pass.
func NewOrValidator(validators ...queryfy.ValidatorFunc) *OrValidator {
	return &OrValidator{validators: validators}
}

// Validate runs all validators and succeeds if any pass.
func (v *OrValidator) Validate(value interface{}) error {
	if len(v.validators) == 0 {
		return nil
	}

	var lastError error

	for _, validator := range v.validators {
		if err := validator(value); err == nil {
			return nil // At least one validator passed
		} else {
			lastError = err
		}
	}

	// All validators failed, return the last error
	if lastError != nil {
		return &queryfy.FieldError{
			Message: "none of the validators passed",
			Value:   value,
		}
	}

	return nil
}

// NotValidator inverts the result of another validator.
type NotValidator struct {
	validator queryfy.ValidatorFunc
}

// NewNotValidator creates a validator that inverts another validator's result.
func NewNotValidator(validator queryfy.ValidatorFunc) *NotValidator {
	return &NotValidator{validator: validator}
}

// Validate succeeds if the wrapped validator fails.
func (v *NotValidator) Validate(value interface{}) error {
	if err := v.validator(value); err == nil {
		return &queryfy.FieldError{
			Message: "value must not match the validation",
			Value:   value,
		}
	}
	return nil
}
