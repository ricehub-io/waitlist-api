package handlers

import (
	"fmt"
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func InitCustomValidation() error {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("ricetitle", func(fl validator.FieldLevel) bool {
			re := regexp.MustCompile(`^[a-zA-Z0-9 '_-]+$`)
			return re.MatchString(fl.Field().String())
		}); err != nil {
			return fmt.Errorf("ricetitle validation: %w", err)
		}
	}
	return nil
}
