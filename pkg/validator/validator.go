package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// CustomValidator envuelve la instancia de go-playground/validator.
type CustomValidator struct {
	validate *validator.Validate
}

// New crea e inicializa un validador con configuraciones por defecto.
func New() *CustomValidator {
	return &CustomValidator{
		validate: validator.New(),
	}
}

// ValidateStruct valida cualquier struct y formatea los errores de validación de forma legible.
func (v *CustomValidator) ValidateStruct(s interface{}) error {
	if err := v.validate.Struct(s); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var errMsgs []string
			for _, fieldErr := range validationErrors {
				errMsgs = append(errMsgs, formatFieldError(fieldErr))
			}
			return fmt.Errorf("%s", strings.Join(errMsgs, "; "))
		}
		return err
	}
	return nil
}

func formatFieldError(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("el campo '%s' es obligatorio", fe.Field())
	case "min":
		return fmt.Sprintf("el campo '%s' debe tener al menos %s caracteres", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("el campo '%s' no puede exceder %s caracteres", fe.Field(), fe.Param())
	case "oneof":
		return fmt.Sprintf("el campo '%s' debe ser uno de: [%s]", fe.Field(), fe.Param())
	default:
		return fmt.Sprintf("el campo '%s' falló la validación '%s'", fe.Field(), fe.Tag())
	}
}
