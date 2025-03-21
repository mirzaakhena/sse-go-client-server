package model

import "gorm.io/gorm"

// TODO completing this later
type Client struct {
	gorm.Model
	ClientID string
}
