package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var defaultValues = &AppConfig{
	FolderName:     "Presencial",
	ReportFilePath: "registros.xlsx",
	DefaultGoal:    8,
	ExtraLabel:     "adicional",
	AreaOptions:    []string{"CT", "CEIC", "AG", "OUTRO"},
	Headers:        []string{"data", "hora", "resposta", "observacao", "area"},
	YesReport:      "S",
	NoReport:       "N",
}

type AppConfig struct {
	homeDir        string
	configPath     string
	FolderName     string   `json:"folderName"`
	ReportFilePath string   `json:"reportfilepath"`
	ExtraLabel     string   `json:"extraLabel"`
	AreaOptions    []string `json:"areaOptions"`
	Headers        []string `json:"headers"`
	DefaultGoal    int      `json:"meta"`
	YesReport      string   `json:"respostaSim"`
	NoReport       string   `json:"respostaNao"`
}

func (a *AppConfig) SetConfig(cfg *AppConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.configPath, data, 0644)
}

func GetConfig(filename string) (*AppConfig, bool, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, false, fmt.Errorf("erro ao obter diretório do usuário: %v", err)
	}

	dataFolder := filepath.Join(homeDir, defaultValues.FolderName)
	configPath := filepath.Join(dataFolder, filename)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(dataFolder, os.ModePerm); err != nil {
			return nil, false, fmt.Errorf("erro ao criar pasta: %v", err)
		}

		defaultValues.homeDir = dataFolder
		defaultValues.configPath = configPath
		defaultValues.ReportFilePath = filepath.Join(dataFolder, defaultValues.ReportFilePath)

		return defaultValues, false, nil
	}

	cfg := &AppConfig{}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, false, fmt.Errorf("erro ao ler config.json: %v", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, false, fmt.Errorf("erro ao interpretar config.json: %v", err)
	}

	cfg.homeDir = dataFolder
	cfg.configPath = configPath
	cfg.ReportFilePath = filepath.Join(dataFolder, defaultValues.ReportFilePath)

	return cfg, true, nil
}
