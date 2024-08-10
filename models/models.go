package models

import (
	"time"

	"gorm.io/gorm"
)

type ZapPayDB struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Email     string         `gorm:"unique"`
	Amount    int
}

type Transaction struct {
	Tid           string `gorm:"primaryKey"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
	SenderEmail   string
	ReceiverEmail string
	Amount        int
	State         string
}
