// Package program provides the core functionality for the attendance tracking application
package program

import (
	"time"

	"github.com/google/uuid"
)

// App represents the main application entity with configuration and localization settings
type App struct {
	ID            uint `gorm:"primarykey"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	AppID         uuid.UUID
	Name          string
	Theme         string
	LanguageID    uint
	Language      AppLanguage `gorm:"foreignKey:LanguageID"`
	InteractionID uint
	Interaction   AppInteraction `gorm:"foreignKey:InteractionID"`
	AppConfigID   uint
	AppConfig     AppConfig `gorm:"foreignKey:AppConfigID"`
}

// AppLanguage contains all the localized text strings used in the application interface
type AppLanguage struct {
	ID          uint `gorm:"primarykey"`
	CreatedAt   time.Time
	WindowName  string
	Title       string
	Welcome     string
	Goal        string
	Report      string
	Observation string
	Area        string
	Save        string
	Cancel      string
	Yes         string
	No          string
	Close       string
	Error       string
	Success     string
	SuccessMsg  string
	ErrorMsg    string
	WarningMsg  string
	Warning     string
	Info        string
}

// AppInteraction defines the interaction elements and options for the application
type AppInteraction struct {
	ID          uint `gorm:"primarykey"`
	ExtraLabel  string
	AreaOptions string
	Headers     string
}

// AppConfig stores configuration settings for the application
type AppConfig struct {
	ID          uint `gorm:"primarykey"`
	DefaultGoal int
}

// PresenceRecord to hold records
type PresenceRecord struct {
	ID          uint `gorm:"primaryKey"`
	Date        string
	Time        string
	Response    string
	Observation string
	Area        string
}
