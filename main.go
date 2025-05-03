package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/dyammarcano/presencial/config"
	"github.com/xuri/excelize/v2"
)

func presence(report, observance, area string) []string {
	p := &struct {
		Date        string `json:"data"`
		Time        string `json:"hora"`
		Response    string `json:"resposta"`
		Observation string `json:"observacao"`
		Area        string `json:"area"`
	}{
		Date:        time.Now().Format("02/01/2006"),
		Time:        time.Now().Format("15:04:05"),
		Response:    report,
		Observation: observance,
		Area:        area,
	}

	return []string{p.Date, p.Time, p.Response, p.Observation, p.Area}
}

func ensureFileExists(cfg *config.AppConfig) {
	if _, err := os.Stat(cfg.ReportFilePath); os.IsNotExist(err) {
		log.Println("Arquivo não existe, criando novo .xlsx válido com cabeçalhos...")
		f := excelize.NewFile()
		sheet := f.GetSheetName(f.GetActiveSheetIndex())
		for i, header := range cfg.Headers {
			cell, err := excelize.CoordinatesToCellName(i+1, 1)
			if err != nil {
				log.Fatalf("Erro ao calcular nome da célula: %v", err)
			}
			if err := f.SetCellValue(sheet, cell, header); err != nil {
				log.Fatalf("Erro ao definir valor na célula %s: %v", cell, err)
			}
		}
		if err := f.SaveAs(cfg.ReportFilePath); err != nil {
			log.Fatalf("Erro ao salvar arquivo .xlsx: %v", err)
		}
		log.Printf("Arquivo %s criado com sucesso.", cfg.ReportFilePath)
	}
}

func showConfigForm(win fyne.Window, cfg *config.AppConfig, onComplete func()) {
	entryDefault := widget.NewEntry()
	entryDefault.SetPlaceHolder("Dias presenciais (ex: 8)")

	saveBtn := widget.NewButton("Salvar", func() {
		dg, err := strconv.Atoi(entryDefault.Text)
		if err != nil || dg < 1 || dg > 24 {
			dialog.ShowError(fmt.Errorf("valores inválidos"), win)
			return
		}

		cfg.DefaultGoal = dg

		if err := cfg.SetConfig(cfg); err != nil {
			dialog.ShowError(err, win)
		}

		dialog.ShowInformation("Salvo", "Configuração salva com sucesso", win)
		onComplete()
	})

	form := container.NewVBox(
		widget.NewLabel("Primeiro uso - configure os limites de presença:"),
		entryDefault,
		saveBtn,
	)

	win.SetContent(form)
	win.Resize(fyne.NewSize(300, 200))
	win.Show()
}

func savePresenceToExcel(report, observation string, area string, cfg *config.AppConfig) error {
	fileExists := false
	if _, err := os.Stat(cfg.ReportFilePath); err == nil {
		fileExists = true
	}

	var f *excelize.File
	var err error

	if fileExists {
		f, err = excelize.OpenFile(cfg.ReportFilePath)
		if err != nil {
			log.Printf("Erro ao abrir arquivo existente: %v", err)
			return fmt.Errorf("falha ao abrir arquivo existente: %w", err)
		}
		log.Println("Arquivo Excel existente aberto.")
	} else {
		f = excelize.NewFile()
		sheet := f.GetSheetName(f.GetActiveSheetIndex())
		for i, h := range cfg.Headers {
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
	for i, val := range presence(report, observation, area) {
		cell, _ := excelize.CoordinatesToCellName(i+1, nextRow)
		if err := f.SetCellValue(sheet, cell, val); err != nil {
			log.Printf("Erro ao definir valor da célula %s: %v", cell, err)
			return fmt.Errorf("falha ao escrever célula %s: %w", cell, err)
		}
	}

	if err := f.SaveAs(cfg.ReportFilePath); err != nil {
		log.Printf("Erro ao salvar arquivo Excel: %v", err)
		return fmt.Errorf("falha ao salvar: %w", err)
	}

	log.Printf("Presença salva com sucesso em: %s", cfg.ReportFilePath)
	return nil
}

func loadMonthlyReport(cfg *config.AppConfig) string {
	f, err := excelize.OpenFile(cfg.ReportFilePath)
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

		if date.Month() == now.Month() && date.Year() == now.Year() && resp == cfg.YesReport {
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

func countMonthlyPresence(cfg *config.AppConfig) (int, error) {
	f, err := excelize.OpenFile(cfg.ReportFilePath)
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
		if parsedDate.Month() == now.Month() && parsedDate.Year() == now.Year() && resposta == cfg.YesReport {
			count++
		}
	}
	return count, nil
}

func showAreaPopup(myApp fyne.App, myWin fyne.Window, cfg *config.AppConfig, observation string) {
	newArea := ""
	selectWidget := widget.NewSelect(cfg.AreaOptions, func(selected string) {
		newArea = selected
	})
	selectWidget.PlaceHolder = "Selecione a área"

	var pop dialog.Dialog

	acceptButton := widget.NewButton("✔ Aceitar", func() {
		if newArea == "" {
			dialog.ShowInformation("Erro", "Você precisa selecionar uma área", myWin)
			return
		}
		if err := savePresenceToExcel(cfg.YesReport, observation, newArea, cfg); err != nil {
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

func buildMainContent(myApp fyne.App, myWin fyne.Window, cfg *config.AppConfig) fyne.CanvasObject {
	reportLabel := widget.NewLabel(loadMonthlyReport(cfg))
	reportLabel.Wrapping = fyne.TextWrapWord

	observation := ""

	label := widget.NewLabel("Você está presencial hoje?")

	buttonYes := widget.NewButton("✔ Sim", func() {
		count, err := countMonthlyPresence(cfg)
		if err != nil {
			dialog.ShowError(err, myWin)
			return
		}
		if count >= cfg.DefaultGoal {
			dialog.ShowInformation("Meta atingida",
				fmt.Sprintf("Você já atingiu a meta de %d dias presenciais neste mês!", cfg.DefaultGoal),
				myWin,
			)
		}
		showAreaPopup(myApp, myWin, cfg, observation)
	})

	buttonNo := widget.NewButton("✖ Não", func() {
		if err := savePresenceToExcel(cfg.NoReport, "", "", cfg); err != nil {
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

	return form
}

func main() {
	myApp := app.New()
	myWin := myApp.NewWindow("Controle de Presença")
	myWin.SetFixedSize(true)

	cfg, ok, err := config.GetConfig("config.json")
	if err != nil {
		log.Fatal(err)
	}

	ensureFileExists(cfg)

	if !ok {
		showConfigForm(myWin, cfg, func() {
			myWin.SetContent(buildMainContent(myApp, myWin, cfg))
		})
	} else {
		myWin.SetContent(buildMainContent(myApp, myWin, cfg))
	}

	myWin.Resize(fyne.NewSize(400, 250))
	myWin.ShowAndRun()
}
