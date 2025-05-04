package model

import (
	"time"

	"github.com/google/uuid"
)

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
	ReportID      uint
	Report        AppReport `gorm:"foreignKey:ReportID"`
}

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

type AppInteraction struct {
	ID          uint `gorm:"primarykey"`
	ExtraLabel  string
	AreaOptions string // JSON string
	Headers     string // JSON string
}

type AppReport struct {
	ID          uint `gorm:"primarykey"`
	CreatedAt   time.Time
	YesReport   string
	NoReport    string
	DefaultGoal int
}

type AppConfig struct {
	AppID          uuid.UUID
	FolderName     string
	ReportFilePath string
	DefaultGoal    int
	YesReport      string
	NoReport       string
	ExtraLabel     string
	AreaOptions    []string
	Headers        []string
}

type PresenceRecord struct {
	ID          uint `gorm:"primaryKey"`
	Date        string
	Time        string
	Response    string
	Observation string
	Area        string
}
