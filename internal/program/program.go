package program

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/dyammarcano/presencial/internal/model"
	"github.com/dyammarcano/presencial/internal/theme"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type MainApp struct {
	db *gorm.DB

	app fyne.App
	win fyne.Window

	firstRun bool

	*model.App
	*model.AppConfig
}

func NewMainApp(appName string) (*MainApp, error) {
	a := &MainApp{
		app: theme.NewSmallFontTheme(app.New()),
	}

	if err := a.setupDatabase(appName); err != nil {
		return nil, err
	}

	if err := a.initApp(); err != nil {
		return nil, err
	}

	return a, nil
}

func (m *MainApp) getAppDataFolder(appName string) string {
	var base string

	switch runtime.GOOS {
	case "windows":
		base = os.Getenv("AppData") // e.g., C:\Users\<User>\AppData\Roaming
		if base == "" {
			base = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
	case "darwin":
		base = filepath.Join(os.Getenv("HOME"), "Library", "Application Support")
	default: // Linux and others
		base = os.Getenv("XDG_DATA_HOME")
		if base == "" {
			base = filepath.Join(os.Getenv("HOME"), ".local", "share")
		}
	}

	base = filepath.Join(base, appName)
	if err := os.MkdirAll(base, 0755); err != nil {
		log.Printf("erro ao criar diretório de dados: %v", err)
		return ""
	}
	return base
}

func (m *MainApp) setupDatabase(appName string) error {
	dbPath := filepath.Join(m.getAppDataFolder(appName), "application.db")

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		m.firstRun = true
	}

	var err error
	m.db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("erro ao conectar no banco de dados: %v", err)
	}

	if err = m.db.AutoMigrate(
		&model.AppLanguage{},
		&model.AppInteraction{},
		&model.AppReport{},
		&model.App{},
		&model.PresenceRecord{},
	); err != nil {
		return fmt.Errorf("erro ao migrar estruturas: %v", err)
	}

	if m.firstRun {
		if err := m.createDefaultApp(); err != nil {
			return fmt.Errorf("erro ao criar dados padrão: %v", err)
		}
	}
	return nil
}

func (m *MainApp) initApp() error {
	if err := m.loadConfigFromDB(); err != nil {
		return err
	}

	m.buildMainMenu()

	if m.firstRun {
		m.showConfigForm(func() {
			m.win.SetContent(m.buildMainContent())
		})
	} else {
		m.win.SetContent(m.buildMainContent())
	}

	return nil
}

func (m *MainApp) loadConfigFromDB() error {
	var app model.App
	if err := m.db.Preload("Language").Preload("Interaction").Preload("Report").First(&app).Error; err != nil {
		return fmt.Errorf("erro ao carregar configuração do app: %v", err)
	}

	m.App = &app

	type arr struct {
		Values []string `json:"areas"`
	}

	var area arr
	_ = json.Unmarshal([]byte(app.Interaction.AreaOptions), &area)

	if err := json.Unmarshal([]byte(app.Interaction.AreaOptions), &area); err != nil {
		return fmt.Errorf("erro ao interpretar AreaOptions: %w", err)
	}

	var head arr
	if err := json.Unmarshal([]byte(app.Interaction.Headers), &head); err != nil {
		return fmt.Errorf("erro ao interpretar HeadersOptions: %w", err)
	}

	m.AppConfig = &model.AppConfig{

		AppID:          app.AppID,
		FolderName:     "Presencial",
		ReportFilePath: "",
		DefaultGoal:    app.Report.DefaultGoal,
		YesReport:      app.Report.YesReport,
		NoReport:       app.Report.NoReport,
		ExtraLabel:     app.Interaction.ExtraLabel,
		AreaOptions:    area.Values,
		Headers:        head.Values,
	}

	return nil
}

func (m *MainApp) buildMainContent() fyne.CanvasObject {
	reportLabel := widget.NewLabel(m.loadMonthlyReport())
	reportLabel.Wrapping = fyne.TextWrapWord

	observation := ""

	label := widget.NewLabel("Você está presencial hoje?")

	buttonYes := widget.NewButton("✔ Sim", func() {
		count, err := m.countMonthlyPresence()
		if err != nil {
			m.app.SendNotification(&fyne.Notification{
				Title:   "Erro",
				Content: err.Error(),
			})
			return
		}

		if count >= m.DefaultGoal {
			info := dialog.NewInformation("Meta atingida",
				fmt.Sprintf("Você já atingiu a meta de %d dias presenciais neste mês!", m.DefaultGoal), m.win,
			)

			info.SetOnClosed(func() {
				m.showAreaPopup(observation)
			})

			info.Show()
			return
		}
		m.showAreaPopup(observation)
	})

	buttonNo := widget.NewButton("✖ Não", func() {
		m.app.SendNotification(&fyne.Notification{
			Title:   "Informativo",
			Content: "Tudo bem. Hoje não será contado como presencial.",
		})
		m.app.Quit()
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

func (m *MainApp) countMonthlyPresence() (int, error) {
	now := time.Now()
	var count int
	err := m.db.Raw("SELECT COUNT(*) FROM presence_records WHERE strftime('%m', date) = ? AND strftime('%Y', date) = ? AND response = ?", now.Format("01"), now.Format("2006"), m.Language.Yes).Scan(&count).Error
	return count, err
}

func (m *MainApp) showAreaPopup(observation string) {
	newArea := ""
	selectWidget := widget.NewSelect(m.AreaOptions, func(selected string) { newArea = selected })
	selectWidget.PlaceHolder = "Selecione a área"

	var pop dialog.Dialog

	acceptButton := widget.NewButton("✔ Aceitar", func() {
		if newArea == "" {
			dialog.ShowInformation("Erro", "Você precisa selecionar uma área", m.win)
			return
		}

		if err := m.savePresenceToDB(&model.PresenceRecord{Response: m.Language.Yes, Observation: observation, Area: newArea}); err != nil {
			m.app.SendNotification(&fyne.Notification{
				Title:   "Erro",
				Content: err.Error(),
			})
			return
		}

		pop.Hide()

		m.app.SendNotification(&fyne.Notification{
			Title:   "Salvo",
			Content: "Presença registrada com sucesso.",
		})
		m.app.Quit()
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
	), m.win)

	pop.Resize(fyne.NewSize(340, 180))
	pop.Show()
}

func (m *MainApp) savePresenceToDB(presence *model.PresenceRecord) error {
	query := fmt.Sprintf("INSERT INTO presence_records (date, time, response, observation, area) VALUES (?, ?, ?, ?, ?)")
	return m.db.Exec(query, time.Now().Format("02/01/2006"), time.Now().Format("15:04:05"), presence.Response, presence.Observation, presence.Area).Error
}

func (m *MainApp) buildMainMenu() {
	m.win = m.app.NewWindow(m.Language.Title)
	m.win.SetFixedSize(true)

	fileMenu := fyne.NewMenu("Arquivo",
		fyne.NewMenuItem("Sair", func() {
			m.app.Quit()
		}),
	)

	editMenu := fyne.NewMenu("Editar",
		fyne.NewMenuItem("Configuração", func() {
			dialog.ShowInformation("Configuração", "Função de configuração futura", m.win)
		}),
	)

	helpMenu := fyne.NewMenu("Ajuda",
		fyne.NewMenuItem("Documentação", func() {
			dialog.ShowInformation("Ajuda", "Visite github.com/dyammarcano/presencial", m.win)
		}),
	)

	aboutMenu := fyne.NewMenu("Sobre",
		fyne.NewMenuItem("Sobre o App", func() {
			dialog.ShowInformation("Sobre", "Controle de Presença v1.0\nCriado por Dyam", m.win)
		}),
	)

	m.win.SetMainMenu(fyne.NewMainMenu(fileMenu, editMenu, helpMenu, aboutMenu))
}

func (m *MainApp) showConfigForm(onComplete func()) {
	entryDefault := widget.NewEntry()
	entryDefault.SetPlaceHolder("Dias presenciais (ex: 8)")

	saveBtn := widget.NewButton("Salvar", func() {
		dg, err := strconv.Atoi(entryDefault.Text)
		if err != nil || dg < 1 || dg > 24 {
			dialog.ShowError(fmt.Errorf("valores inválidos"), m.win)
			return
		}

		m.DefaultGoal = dg

		var app model.App
		if err := m.db.Preload("Report").Where("app_id = ?", m.ID).First(&app).Error; err != nil {
			dialog.ShowError(fmt.Errorf("erro ao buscar app: %w", err), m.win)
			return
		}
		app.Report.DefaultGoal = dg
		if err := m.db.Save(&app.Report).Error; err != nil {
			dialog.ShowError(fmt.Errorf("erro ao salvar config: %w", err), m.win)
			return
		}

		dialog.ShowInformation("Salvo", "Configuração salva com sucesso", m.win)
		onComplete()
	})

	form := container.NewVBox(
		widget.NewLabel("Primeiro uso - configure os limites de presença:"),
		entryDefault,
		saveBtn,
	)

	m.win.SetContent(form)
	m.win.Resize(fyne.NewSize(300, 200))
	m.win.Show()
}

func (m *MainApp) createDefaultApp() error {
	var count int64
	m.db.Model(&model.App{}).Count(&count)
	if count > 0 {
		return nil
	}

	modelLang := model.AppLanguage{
		WindowName:  "Controle de Presença",
		Title:       "Controle de Presença",
		Welcome:     "Bem-vindo",
		Goal:        "Meta de dias presenciais",
		Report:      "Relatório de Presença",
		Observation: "Observação",
		Area:        "Área",
		Save:        "Salvar",
		Cancel:      "Cancelar",
		Yes:         "Sim",
		No:          "Não",
		Close:       "Fechar",
		Error:       "Erro",
		Success:     "Sucesso",
		SuccessMsg:  "Presença registrada com sucesso",
		ErrorMsg:    "Erro ao registrar presença",
		Warning:     "Aviso",
		WarningMsg:  "Meta já atingida",
		Info:        "Informação",
	}

	if err := m.db.Create(&modelLang).Error; err != nil {
		return err
	}

	modelInteract := model.AppInteraction{
		ExtraLabel:  "adicional",
		AreaOptions: `{"areas": ["CT", "CEIC", "AG", "OUTRO"]}`,
		Headers:     `{"headers": ["data", "hora", "resposta", "observacao", "area"]}`,
	}

	if err := m.db.Create(&modelInteract).Error; err != nil {
		return err
	}

	modelReport := model.AppReport{
		YesReport:   "S",
		NoReport:    "N",
		DefaultGoal: 8,
	}

	if err := m.db.Create(&modelReport).Error; err != nil {
		return err
	}

	modelApp := model.App{
		AppID:         uuid.New(),
		Name:          "PresencialApp",
		Theme:         "light",
		LanguageID:    modelLang.ID,
		InteractionID: modelInteract.ID,
		ReportID:      modelReport.ID,
	}
	return m.db.Create(&modelApp).Error
}

func (m *MainApp) loadMonthlyReport() string {
	now := time.Now()
	rows, err := m.db.Raw("SELECT date, area FROM presence_records WHERE strftime('%m', date) = ? AND strftime('%Y', date) = ? AND response = ?", now.Format("01"), now.Format("2006"), m.Language.Yes).Rows()
	if err != nil {
		return "Erro ao carregar relatório."
	}
	defer rows.Close()

	var report string
	count := 0
	for rows.Next() {
		var date, area string
		if err := rows.Scan(&date, &area); err == nil {
			report += fmt.Sprintf("* %s - %s\n", date, area)
			count++
		}
	}

	if count == 0 {
		return "Nenhuma presença registrada este mês."
	}
	return fmt.Sprintf("Você marcou presença %d vez(es) neste mês:\n\n%s", count, report)
}

func (m *MainApp) RunApp() {
	m.win.Resize(fyne.NewSize(400, 250))
	m.win.CenterOnScreen()
	m.win.ShowAndRun()
}
