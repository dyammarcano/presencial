package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/xuri/excelize/v2"
)

const (
	report   = "S"
	noReport = "N"
)

var (
	areaOptions = []string{"AG", "CT", "CEIC", "OUTRO"}
	headers     = []string{"data", "hora", "resposta", "observacao", "area"}
)

type Presence struct {
	Date        string `json:"data" csv:"data"`
	Time        string `json:"hora" csv:"hora"`
	Response    string `json:"resposta" csv:"resposta"`     // "S" ou "N"
	Observation string `json:"observacao" csv:"observacao"` // Campo livre
	Area        string `json:"area" csv:"area"`             // AG, CT, CEIC, OUTRO
}

func newPresence(report, observance, area string) *Presence {
	return &Presence{
		Date:        time.Now().Format("02/01/2006"),
		Time:        time.Now().Format("15:04:05"),
		Response:    report,
		Observation: observance,
		Area:        area,
	}
}

func (p *Presence) ToSlice() []string {
	return []string{
		p.Date,
		p.Time,
		p.Response,
		p.Observation,
		p.Area,
	}
}

type AppConfig struct {
	FolderName  string
	CsvFilename string
	DefaultGoal int
	MaxGoal     int
	ExtraLabel  string
	CsvHeaders  []string
}

var config = AppConfig{
	FolderName:  "Presencial",
	CsvFilename: "registros.xlsx",
	DefaultGoal: 8,
	MaxGoal:     31,
	ExtraLabel:  "extra",
	CsvHeaders:  []string{"data", "hora", "resposta", "observacao", "area"},
}

func getPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	dataFolder := filepath.Join(homeDir, config.FolderName)
	_ = os.MkdirAll(dataFolder, os.ModePerm)
	return filepath.Join(dataFolder, config.CsvFilename)
}

func ensureFileExists(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		file, err := os.Create(path)
		if err != nil {
			log.Fatal(err)
		}

		writer := csv.NewWriter(file)
		_ = writer.Write(config.CsvHeaders)
		writer.Flush()
		_ = file.Close()
	}
}

func savePresenceToExcel(xlsxPath string, isPresent bool, observation string, area string) error {
	fileExists := false
	if _, err := os.Stat(xlsxPath); err == nil {
		fileExists = true
	}

	var f *excelize.File
	var err error

	if fileExists {
		f, err = excelize.OpenFile(xlsxPath)
		if err != nil {
			return fmt.Errorf("failed to open existing file: %w", err)
		}
	} else {
		f = excelize.NewFile()
		sheet := f.GetSheetName(f.GetActiveSheetIndex())
		for i, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheet, cell, h)
		}
	}

	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println("error closing file:", err)
		}
	}()

	sheet := f.GetSheetName(f.GetActiveSheetIndex())
	rows, err := f.GetRows(sheet)
	if err != nil {
		return fmt.Errorf("failed to read rows: %w", err)
	}

	nextRow := len(rows) + 1
	presence := newPresence(report, observation, area)

	for i, val := range presence.ToSlice() {
		cell, _ := excelize.CoordinatesToCellName(i+1, nextRow)
		if err := f.SetCellValue(sheet, cell, val); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
	}

	if err := f.SaveAs(xlsxPath); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

func main() {
	myApp := app.New()
	myWin := myApp.NewWindow("Controle de Presença")

	filePath := getPath()
	ensureFileExists(filePath)

	area := ""
	observation := ""

	areaSelect := widget.NewSelect(areaOptions, func(value string) {
		area = value
	})

	label := widget.NewLabel("Você está presencial hoje?")
	buttonYes := widget.NewButton("Sim", func() {
		if area == "" {
			dialog.ShowInformation("Erro", "Selecione uma área", myWin)
			return
		}
		savePresenceToExcel(filePath, true, observation, area)
		dialog.ShowInformation("Salvo", "Presença registrada com sucesso.", myWin)
	})

	buttonNo := widget.NewButton("Não", func() {
		savePresenceToExcel(filePath, false, "", "")
		dialog.ShowInformation("Informativo", "Tudo bem. Hoje não será contado como presencial.", myWin)
	})

	form := container.NewVBox(
		label,
		areaSelect,
		buttonYes,
		buttonNo,
	)

	myWin.SetContent(form)
	myWin.ShowAndRun()
}
