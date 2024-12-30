package model

import (
	"time"

	"gorm.io/gorm"
)

type Employee struct {
	gorm.Model
	Name       string    `json:"name"`
	Position   string    `json:"position"`
	Salary     uint      `json:"salary"`
	Hired_date time.Time `json:"hired_date"`
}
