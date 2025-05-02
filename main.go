package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/xuri/excelize/v2"
)

const (
	yesReport = "S"
	noReport  = "N"
)

var (
	areaOptions = []string{"CT", "CEIC", "AG", "OUTRO"}
	headers     = []string{"data", "hora", "resposta", "observacao", "area"}
)

type JsonConfig struct {
	DefaultGoal int `json:"Goal"`
}

type Presence struct {
	Date        string `json:"data" csv:"data"`
	Time        string `json:"hora" csv:"hora"`
	Response    string `json:"resposta" csv:"resposta"`
	Observation string `json:"observacao" csv:"observacao"`
	Area        string `json:"area" csv:"area"`
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
	return []string{p.Date, p.Time, p.Response, p.Observation, p.Area}
}

type AppConfig struct {
	FolderName  string
	Filename    string
	DefaultGoal int
	ExtraLabel  string
	Headers     []string
}

var config = AppConfig{
	FolderName:  "Presencial",
	Filename:    "registros1.xlsx",
	DefaultGoal: 8,
	ExtraLabel:  "extra",
	Headers:     []string{"data", "hora", "resposta", "observacao", "area"},
}

func saveConfigToJSON(path string, cfg JsonConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao serializar config.json: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func loadConfigFromJSON(configPath string) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("Config.json não encontrado ou erro ao ler: %v. Usando valor padrão.", err)
		return
	}

	var fileConfig struct {
		DefaultGoal int `json:"defaultGoal"`
	}

	if err := json.Unmarshal(data, &fileConfig); err != nil {
		log.Printf("Erro ao interpretar config.json: %v. Usando padrão.", err)
		return
	}

	if fileConfig.DefaultGoal > 0 {
		config.DefaultGoal = fileConfig.DefaultGoal
	}
}

func getPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Erro ao obter diretório do usuário: %v", err)
	}
	dataFolder := filepath.Join(homeDir, config.FolderName)
	if err := os.MkdirAll(dataFolder, os.ModePerm); err != nil {
		log.Fatalf("Erro ao criar pasta: %v", err)
	}
	return filepath.Join(dataFolder, config.Filename)
}

func ensureFileExists(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Println("Arquivo não existe, criando novo .xlsx válido com cabeçalhos...")

		f := excelize.NewFile()

		sheet := f.GetSheetName(f.GetActiveSheetIndex())

		for i, header := range config.Headers {
			cell, err := excelize.CoordinatesToCellName(i+1, 1)
			if err != nil {
				log.Fatalf("Erro ao calcular nome da célula: %v", err)
			}
			if err := f.SetCellValue(sheet, cell, header); err != nil {
				log.Fatalf("Erro ao definir valor na célula %s: %v", cell, err)
			}
		}

		if err := f.SaveAs(path); err != nil {
			log.Fatalf("Erro ao salvar arquivo .xlsx: %v", err)
		}

		log.Printf("Arquivo %s criado com sucesso.", path)
	}
}

func savePresenceToExcel(xlsxPath, report, observation string, area string) error {
	fileExists := false
	if _, err := os.Stat(xlsxPath); err == nil {
		fileExists = true
	}

	var f *excelize.File
	var err error

	if fileExists {
		f, err = excelize.OpenFile(xlsxPath)
		if err != nil {
			log.Printf("Erro ao abrir arquivo existente: %v", err)
			return fmt.Errorf("falha ao abrir arquivo existente: %w", err)
		}
		log.Println("Arquivo Excel existente aberto.")
	} else {
		f = excelize.NewFile()
		sheet := f.GetSheetName(f.GetActiveSheetIndex())
		for i, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			if err := f.SetCellValue(sheet, cell, h); err != nil {
				log.Printf("Erro ao escrever cabeçalho: %v", err)
			}
		}
		log.Println("Novo arquivo Excel criado com cabeçalhos.")
	}

	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("Erro ao fechar arquivo: %v", err)
		}
	}()

	sheet := f.GetSheetName(f.GetActiveSheetIndex())
	rows, err := f.GetRows(sheet)
	if err != nil {
		log.Printf("Erro ao ler linhas do Excel: %v", err)
		return fmt.Errorf("falha ao ler linhas: %w", err)
	}

	nextRow := len(rows) + 1
	presence := newPresence(report, observation, area)

	for i, val := range presence.ToSlice() {
		cell, _ := excelize.CoordinatesToCellName(i+1, nextRow)
		if err := f.SetCellValue(sheet, cell, val); err != nil {
			log.Printf("Erro ao definir valor da célula %s: %v", cell, err)
			return fmt.Errorf("falha ao escrever célula %s: %w", cell, err)
		}
	}

	if err := f.SaveAs(xlsxPath); err != nil {
		log.Printf("Erro ao salvar arquivo Excel: %v", err)
		return fmt.Errorf("falha ao salvar: %w", err)
	}

	log.Printf("Presença salva com sucesso em: %s", xlsxPath)
	return nil
}

func loadMonthlyReport(xlsxPath string) string {
	f, err := excelize.OpenFile(xlsxPath)
	if err != nil {
		log.Printf("Erro ao abrir arquivo Excel para leitura de relatório: %v", err)
		return "Erro ao carregar relatório."
	}
	defer f.Close()

	sheet := f.GetSheetName(f.GetActiveSheetIndex())
	rows, err := f.GetRows(sheet)
	if err != nil {
		log.Printf("Erro ao ler linhas do relatório: %v", err)
		return "Erro ao carregar relatório."
	}

	var report string
	now := time.Now()
	count := 0

	for i, row := range rows {
		if i == 0 || len(row) < 3 {
			continue
		}

		dateStr := row[0]
		resp := row[2]
		area := "N/A"
		if len(row) >= 5 {
			area = row[4]
		}

		date, err := time.Parse("02/01/2006", dateStr)
		if err != nil {
			continue
		}

		if date.Month() == now.Month() && date.Year() == now.Year() && resp == yesReport {
			report += fmt.Sprintf("* %s - %s\n", dateStr, area)
			count++
		}
	}

	header := fmt.Sprintf("Você marcou presença %d vez(es) neste mês:\n\n", count)
	if count == 0 {
		return "Nenhuma presença registrada este mês."
	}
	return fmt.Sprintf("%s,%s", header, report)
}

func countMonthlyPresence(xlsxPath string) (int, error) {
	f, err := excelize.OpenFile(xlsxPath)
	if err != nil {
		return 0, fmt.Errorf("falha ao abrir arquivo: %w", err)
	}
	defer f.Close()

	sheet := f.GetSheetName(f.GetActiveSheetIndex())
	rows, err := f.GetRows(sheet)
	if err != nil {
		return 0, fmt.Errorf("falha ao ler linhas: %w", err)
	}

	count := 0
	now := time.Now()
	for i, row := range rows {
		if i == 0 || len(row) < 3 {
			continue // Skip headers or malformed rows
		}
		dateStr, resposta := row[0], row[2]
		parsedDate, err := time.Parse("02/01/2006", dateStr)
		if err != nil {
			continue
		}
		if parsedDate.Month() == now.Month() && parsedDate.Year() == now.Year() && resposta == yesReport {
			count++
		}
	}
	return count, nil
}

func showConfigForm(app fyne.App, win fyne.Window, configPath string, onComplete func(JsonConfig)) {
	var defaultGoal int

	entryDefault := widget.NewEntry()
	entryDefault.SetPlaceHolder("Dias presenciais (ex: 8)")

	entryMax := widget.NewEntry()
	entryMax.SetPlaceHolder("Limite máximo (ex: 31)")

	saveBtn := widget.NewButton("Salvar", func() {
		dg, err := strconv.Atoi(entryDefault.Text)

		if err != nil || dg < 1 || dg > 31 {
			dialog.ShowError(fmt.Errorf("valores inválidos"), win)
			return
		}

		cfg := JsonConfig{
			DefaultGoal: dg,
		}
		if err := saveConfigToJSON(configPath, cfg); err != nil {
			dialog.ShowError(err, win)
			return
		}

		dialog.ShowInformation("Salvo", "Configuração salva com sucesso", win)
		onComplete(cfg)
	})

	form := container.NewVBox(
		widget.NewLabel("Primeiro uso - configure os limites de presença:"),
		entryDefault,
		entryMax,
		saveBtn,
	)

	win.SetContent(form)
	win.Resize(fyne.NewSize(300, 200))
	win.Show()
}

func showAreaPopup(myApp fyne.App, myWin fyne.Window, filePath string, observation string) {
	newArea := ""
	selectWidget := widget.NewSelect(areaOptions, func(selected string) {
		newArea = selected
	})
	selectWidget.PlaceHolder = "Selecione a área"

	var pop dialog.Dialog

	acceptButton := widget.NewButton("✔ Aceitar", func() {
		if newArea == "" {
			dialog.ShowInformation("Erro", "Você precisa selecionar uma área", myWin)
			return
		}
		if err := savePresenceToExcel(filePath, yesReport, observation, newArea); err != nil {
			dialog.ShowError(err, myWin)
			return
		}
		pop.Hide()
		dialog.ShowInformation("Salvo", "Presença registrada com sucesso.", myWin)
		myApp.Quit()
	})

	cancelButton := widget.NewButton("✖ Cancelar", func() {
		pop.Hide()
	})

	pop = dialog.NewCustomWithoutButtons("Confirmação", container.NewVBox(
		widget.NewLabel("Confirme a área selecionada:"),
		selectWidget,
		container.New(
			layout.NewGridLayoutWithColumns(2),
			cancelButton,
			acceptButton,
		),
	), myWin)

	pop.Resize(fyne.NewSize(340, 180))
	pop.Show()
}

func buildMainWindow(myApp fyne.App, filePath string) fyne.Window {
	myWin := myApp.NewWindow("Controle de Presença")
	myWin.Resize(fyne.NewSize(400, 250))
	myWin.SetFixedSize(true)

	reportLabel := widget.NewLabel(loadMonthlyReport(filePath))
	reportLabel.Wrapping = fyne.TextWrapWord

	observation := ""

	label := widget.NewLabel("Você está presencial hoje?")

	buttonYes := widget.NewButton("✔ Sim", func() {
		count, err := countMonthlyPresence(filePath)
		if err != nil {
			dialog.ShowError(err, myWin)
			return
		}
		if count >= config.DefaultGoal {
			dialog.ShowInformation("Meta atingida",
				fmt.Sprintf("Você já atingiu a meta de %d dias presenciais neste mês!", config.DefaultGoal),
				myWin,
			)
		}

		showAreaPopup(myApp, myWin, filePath, observation)
	})

	buttonNo := widget.NewButton("✖ Não", func() {
		if err := savePresenceToExcel(filePath, noReport, "", ""); err != nil {
			dialog.ShowError(err, myWin)
			return
		}
		dialog.ShowInformation("Informativo", "Tudo bem. Hoje não será contado como presencial.", myWin)
		myApp.Quit()
	})

	buttons := container.New(
		layout.NewGridLayoutWithColumns(2),
		buttonNo,
		buttonYes,
	)

	form := container.NewVBox(
		reportLabel,
		label,
		buttons,
	)

	myWin.SetContent(form)
	return myWin
}

func main() {
	myApp := app.New()
	myWin := myApp.NewWindow("Controle de Presença")
	myWin.SetFixedSize(true)

	filePath := getPath()
	configPath := filepath.Join(filepath.Dir(filePath), "config.json")
	ensureFileExists(filePath)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		showConfigForm(myApp, myWin, configPath, func(cfg JsonConfig) {
			config.DefaultGoal = cfg.DefaultGoal
			myApp.Quit()
		})
	}

	loadConfigFromJSON(configPath)

	myWin = buildMainWindow(myApp, filePath)
	myWin.ShowAndRun()
}
