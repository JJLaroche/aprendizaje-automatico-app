package clock

import "time"

// Clock provee una interfaz para consultar el tiempo actual de forma determinista y mockeable.
type Clock interface {
	Now() time.Time
}

// RealClock implementa Clock utilizando la hora del sistema en UTC.
type RealClock struct{}

// NewRealClock crea una nueva instancia de RealClock.
func NewRealClock() Clock {
	return &RealClock{}
}

// Now retorna el tiempo actual del sistema en UTC.
func (r *RealClock) Now() time.Time {
	return time.Now().UTC()
}
