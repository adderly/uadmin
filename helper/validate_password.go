package helper

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/muesli/crunchy"
)

const (
	minPasswordLengths = 14
	minPasswordRate    = 80
	dictionaryPath     = "PASS_DICTIONARY_PATH"
)

var (
	ErrLength = fmt.Errorf("password is too short, please enter at least %d characters", minPasswordLengths)
	ErrUpper  = errors.New("password does not contain at least one uppercase letter")
	ErrLow    = errors.New("password does not contain at least one lowercase letter")
	ErrWeak   = errors.New("password is too weak")

	uppercasePattern = regexp.MustCompile(`[A-Z]`)
	lowercasePattern = regexp.MustCompile(`[a-z]`)
)

type PasswordValidator interface {
	Validate(string) error
}

var validator *passwordValidator

type passwordValidator struct {
	validator *crunchy.Validator
}

func NewPasswordValidator() PasswordValidator {
	if validator != nil {
		return validator
	}
	path := os.Getenv(dictionaryPath) // path from user
	validator := &passwordValidator{
		validator: crunchy.NewValidatorWithOpts(crunchy.Options{
			DictionaryPath:    path, // if the path is empty, crunchy will use the default value  "/usr/share/dict"
			MinLength:         minPasswordLengths,
			MustContainDigit:  true,
			MustContainSymbol: true,
		}),
	}
	return validator
}

func (p *passwordValidator) Validate(pass string) error {
	rate, err := p.validator.Rate(pass)
	if err != nil {
		if errors.Is(err, crunchy.ErrTooShort) {
			return ErrLength // error with expected pass lengths
		}
		return err
	}
	// crunchy doesn't return err if the pass doesn't contain upper/lower case letter
	// we need to check the pass to return a more user-friendly message
	if rate < minPasswordRate {
		return handleLowPwdRate(pass)
	}
	return nil
}

func handleLowPwdRate(pass string) error {
	if !uppercasePattern.MatchString(pass) {
		return ErrUpper
	}
	if !lowercasePattern.MatchString(pass) {
		return ErrLow
	}
	return ErrWeak
}
