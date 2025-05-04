package program

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
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

const layoutISO = "2006-01-02"
const layoutBR = "02/01/2006"

type arr struct {
	ValuesArea    []string `json:"areas"`
	ValuesHeaders []string `json:"headers"`
}

type MainApp struct {
	*model.App
	db       *gorm.DB
	app      fyne.App
	win      fyne.Window
	firstRun bool
	records  []model.PresenceRecord
}

func NewMainApp(appName string) (*MainApp, error) {
	a := &MainApp{
		app:     theme.NewSmallFontTheme(app.New()),
		App:     &model.App{},
		records: []model.PresenceRecord{},
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
		base = os.Getenv("AppData")
		if base == "" {
			base = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
	case "darwin":
		base = filepath.Join(os.Getenv("HOME"), "Library", "Application Support")
	default:
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
		&model.App{},
		&model.PresenceRecord{},
		&model.AppConfig{},
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
	if !m.firstRun {
		if err := m.loadConfigFromDB(); err != nil {
			return err
		}
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
	if err := m.db.
		Preload("Language").
		Preload("Interaction").
		Preload("AppConfig").
		First(&m.App).Error; err != nil {
		return fmt.Errorf("erro ao carregar dados do app: %w", err)
	}

	var allRecords []model.PresenceRecord
	if err := m.db.Order("date DESC, time DESC").Find(&allRecords).Error; err != nil {
		log.Printf("erro ao carregar registros anteriores: %v", err)
	}

	now := time.Now()
	for _, r := range allRecords {
		t, err := time.Parse(layoutISO, r.Date)
		if err != nil {
			continue
		}

		if t.Month() == now.Month() && t.Year() == now.Year() {
			m.records = append(m.records, r)
		}
	}

	return nil
}

func (m *MainApp) buildMainContent() fyne.CanvasObject {
	reportLabel := widget.NewLabel(m.loadMonthlyReport())
	reportLabel.Wrapping = fyne.TextWrapWord

	observation := ""

	label := widget.NewLabel("Você está presencial hoje?")

	buttonYes := widget.NewButton("✔ Sim", func() {
		if len(m.records) >= m.App.AppConfig.DefaultGoal {
			info := dialog.NewInformation("Meta atingida",
				fmt.Sprintf("Você já atingiu a meta de %d dias presenciais neste mês!", m.App.AppConfig.DefaultGoal), m.win,
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

func (m *MainApp) showAreaPopup(observation string) {
	newArea := ""

	var area arr
	_ = json.Unmarshal([]byte(m.App.Interaction.AreaOptions), &area)

	selectWidget := widget.NewSelect(area.ValuesArea, func(selected string) { newArea = selected })
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
			<-time.After(10 * time.Millisecond)
			return
		}

		pop.Hide()

		m.app.SendNotification(&fyne.Notification{
			Title:   "Salvo",
			Content: "Presença registrada com sucesso.",
		})
		<-time.After(10 * time.Millisecond)
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
	presence.Date = time.Now().Format(layoutISO)
	presence.Time = time.Now().Format("15:04:05")
	return m.db.Create(presence).Error
}

func (m *MainApp) updateGoal(text string) error {
	dg, err := strconv.Atoi(text)
	if err != nil || dg < 1 || dg > 24 {
		dialog.ShowError(fmt.Errorf("valores inválidos"), m.win)
		return err
	}

	if err := m.db.Preload("AppConfig").Where("app_id = ?", m.App).Error; err != nil {
		dialog.ShowError(fmt.Errorf("erro ao buscar a: %w", err), m.win)
		return err
	}

	m.AppConfig.DefaultGoal = dg

	if err := m.db.Save(&m.App.AppConfig).Error; err != nil {
		dialog.ShowError(fmt.Errorf("erro ao salvar config: %w", err), m.win)
		return err
	}
	return nil
}

func (m *MainApp) showHeaderConfigForm(onComplete func()) {
	var current arr
	if err := json.Unmarshal([]byte(m.Interaction.Headers), &current); err != nil {
		dialog.ShowError(fmt.Errorf("erro ao carregar headers: %w", err), m.win)
		return
	}

	var headerEntries []*widget.Entry
	var headerContainers []fyne.CanvasObject

	buildHeaderList := func() []fyne.CanvasObject {
		return headerContainers
	}

	formContent := container.NewVBox()

	for _, header := range current.ValuesHeaders {
		entry := widget.NewEntry()
		entry.SetText(header)

		removeBtn := widget.NewButton("🗑", func(e *widget.Entry) func() {
			return func() {
				index := -1
				for i, en := range headerEntries {
					if en == e {
						index = i
						break
					}
				}
				if index >= 0 {
					headerEntries = append(headerEntries[:index], headerEntries[index+1:]...)
					headerContainers = append(headerContainers[:index], headerContainers[index+1:]...)
					formContent.Objects = buildHeaderList()
					formContent.Refresh()
				}
			}
		}(entry))

		row := container.NewBorder(nil, nil, nil, removeBtn, entry)
		headerEntries = append(headerEntries, entry)
		headerContainers = append(headerContainers, row)
	}

	formContent.Objects = buildHeaderList()

	addBtn := widget.NewButton("➕ Novo Header", func() {
		entry := widget.NewEntry()
		entry.SetPlaceHolder("Novo header")

		removeBtn := widget.NewButton("🗑", func(e *widget.Entry) func() {
			return func() {
				index := -1
				for i, en := range headerEntries {
					if en == e {
						index = i
						break
					}
				}
				if index >= 0 {
					headerEntries = append(headerEntries[:index], headerEntries[index+1:]...)
					headerContainers = append(headerContainers[:index], headerContainers[index+1:]...)
					formContent.Objects = buildHeaderList()
					formContent.Refresh()
				}
			}
		}(entry))

		row := container.NewBorder(nil, nil, nil, removeBtn, entry)
		headerEntries = append(headerEntries, entry)
		headerContainers = append(headerContainers, row)
		formContent.Objects = buildHeaderList()
		formContent.Refresh()
	})

	saveBtn := widget.NewButton("💾 Salvar", func() {
		var newHeaders []string
		for _, e := range headerEntries {
			txt := strings.TrimSpace(e.Text)
			if txt != "" {
				newHeaders = append(newHeaders, txt)
			}
		}

		newData, err := json.Marshal(map[string][]string{"headers": newHeaders})
		if err != nil {
			dialog.ShowError(fmt.Errorf("erro ao serializar headers: %w", err), m.win)
			return
		}

		m.Interaction.Headers = string(newData)
		if err := m.db.Save(&m.Interaction).Error; err != nil {
			dialog.ShowError(fmt.Errorf("erro ao salvar no banco: %w", err), m.win)
			return
		}

		dialog.ShowInformation("Sucesso", "Headers atualizados!", m.win)
		onComplete()
	})

	mainForm := container.NewVBox(
		widget.NewLabel("Editar Headers:"),
		addBtn,
		formContent,
		saveBtn,
	)

	m.win.SetContent(container.NewVScroll(mainForm))
	m.win.Resize(fyne.NewSize(400, 400))
	m.win.Show()
}

func (m *MainApp) showAreaConfigForm(onComplete func()) {
	var area arr
	if err := json.Unmarshal([]byte(m.App.Interaction.AreaOptions), &area); err != nil {
		dialog.ShowError(fmt.Errorf("erro ao carregar áreas: %w", err), m.win)
		return
	}

	var entries []*widget.Entry
	formContainer := container.NewVBox()

	var refreshForm func()
	refreshForm = func() {
		formContainer.Objects = nil
		entries = []*widget.Entry{}

		for _, val := range area.ValuesArea {
			entry := widget.NewEntry()
			entry.SetText(val)
			entries = append(entries, entry)

			delBtn := widget.NewButton("🗑", func(e *widget.Entry) func() {
				return func() {
					for i, en := range entries {
						if en == e {
							area.ValuesArea = append(area.ValuesArea[:i], area.ValuesArea[i+1:]...)
							refreshForm()
							return
						}
					}
				}
			}(entry))

			row := container.NewHBox(entry, delBtn)
			formContainer.Add(row)
		}

		addBtn := widget.NewButton("➕ Adicionar nova área", func() {
			area.ValuesArea = append(area.ValuesArea, "")
			refreshForm()
		})
		formContainer.Add(addBtn)
	}

	saveBtn := widget.NewButton("💾 Salvar", func() {
		var newAreas []string
		for _, e := range entries {
			val := e.Text
			if val != "" {
				newAreas = append(newAreas, val)
			}
		}
		area.ValuesArea = newAreas

		areaJSON, err := json.Marshal(area)
		if err != nil {
			dialog.ShowError(fmt.Errorf("erro ao serializar áreas: %w", err), m.win)
			return
		}

		m.App.Interaction.AreaOptions = string(areaJSON)
		if err := m.db.Save(&m.App.Interaction).Error; err != nil {
			dialog.ShowError(fmt.Errorf("erro ao salvar no banco de dados: %w", err), m.win)
			return
		}

		dialog.ShowInformation("Sucesso", "Áreas atualizadas com sucesso", m.win)
		onComplete()
	})

	refreshForm()

	content := container.NewBorder(nil, saveBtn, nil, nil, formContainer)

	m.win.SetContent(content)
	m.win.Resize(fyne.NewSize(400, 400))
	m.win.Show()
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
		fyne.NewMenuItem("Configurar Meta de Dias", func() {
			m.showConfigForm(func() {
				m.win.SetContent(m.buildMainContent())
			})
		}),
		fyne.NewMenuItem("Editar Headers", func() {
			m.showHeaderConfigForm(func() {
				m.win.SetContent(m.buildMainContent())
			})
		}),

		fyne.NewMenuItem("Editar Áreas", func() {
			m.showAreaConfigForm(func() {
				m.win.SetContent(m.buildMainContent())
			})
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
	entryDefault.SetPlaceHolder(fmt.Sprintf("Dias presenciais (ex: %d)", m.AppConfig.DefaultGoal))

	saveBtn := widget.NewButton("Salvar", func() {
		if err := m.updateGoal(entryDefault.Text); err != nil {
			dialog.ShowError(fmt.Errorf("erro ao atualizar meta: %w", err), m.win)
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

	m.Language = model.AppLanguage{
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

	if err := m.db.Create(&m.Language).Error; err != nil {
		return err
	}

	m.Interaction = model.AppInteraction{
		ExtraLabel:  "adicional",
		AreaOptions: `{"areas": ["CT", "CEIC", "AG", "OUTRO"]}`,
		Headers:     `{"headers": ["data", "hora", "resposta", "observacao", "area"]}`,
	}

	if err := m.db.Create(&m.Interaction).Error; err != nil {
		return err
	}

	m.AppConfig = model.AppConfig{
		DefaultGoal: 4,
	}

	if err := m.db.Create(&m.AppConfig).Error; err != nil {
		return err
	}

	m.App = &model.App{
		AppID:         uuid.New(),
		Name:          "PresencialApp",
		Theme:         "light",
		LanguageID:    m.Language.ID,
		InteractionID: m.Interaction.ID,
		AppConfigID:   m.AppConfig.ID,
	}

	return m.db.Create(&m.App).Error
}

func (m *MainApp) loadMonthlyReport() string {
	var report string
	for _, r := range m.records {
		t, err := time.Parse(layoutISO, r.Date)
		if err != nil {
			continue
		}

		report += fmt.Sprintf("☑️ %s - %s\n", t.Format(layoutBR), r.Area)
	}

	for i := len(m.records); i < m.AppConfig.DefaultGoal; i++ {
		report += "🔲 (pendente)\n"
	}

	return fmt.Sprintf("Você marcou presença %d vez(es) neste mês:\n\n%s", len(m.records), report)
}

func (m *MainApp) RunApp() {
	m.win.Resize(fyne.NewSize(400, 250))
	m.win.CenterOnScreen()
	m.win.ShowAndRun()
}
