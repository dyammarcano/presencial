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
	"fyne.io/systray"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	layoutISO = "2006-01-02"
	layoutBR  = "02/01/2006"

	high       = 320
	highPopup  = 80
	width      = 400
	widthPopup = 200
)

type arr struct {
	ValuesArea    []string `json:"areas"`
	ValuesHeaders []string `json:"headers"`
}

// MainApp main app structure
type MainApp struct {
	*App
	db       *gorm.DB
	app      fyne.App
	win      fyne.Window
	firstRun bool
	records  []PresenceRecord
}

// NewMainApp main app structure
func NewMainApp(appName string) (*MainApp, error) {
	a := &MainApp{
		app:     newSmallFontTheme(app.New()),
		App:     &App{},
		records: []PresenceRecord{},
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
		&AppLanguage{},
		&AppInteraction{},
		&App{},
		&PresenceRecord{},
		&AppConfig{},
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

	var allRecords []PresenceRecord
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

	label := widget.NewLabel("Como você está trabalhando hoje?")

	buttonPresencial := widget.NewButton("✔ Presencial", func() {
		if len(m.records) >= m.AppConfig.DefaultGoal {
			info := dialog.NewInformation("Meta atingida",
				fmt.Sprintf("Você já atingiu a meta de %d dias presenciais neste mês!", m.AppConfig.DefaultGoal), m.win,
			)

			info.SetOnClosed(func() {
				m.showAreaPopup(observation)
			})

			info.Show()
			return
		}
		m.showAreaPopup(observation)
	})

	buttonRemoto := widget.NewButton("✔ Remoto", func() {
		if err := m.savePresenceToDB(&PresenceRecord{Response: "Remoto", Observation: observation, Area: "Remoto"}); err != nil {
			m.app.SendNotification(&fyne.Notification{
				Title:   "Erro",
				Content: err.Error(),
			})
			<-time.After(10 * time.Millisecond)
			return
		}

		m.app.SendNotification(&fyne.Notification{
			Title:   "Salvo",
			Content: "Trabalho remoto registrado com sucesso.",
		})
		<-time.After(10 * time.Millisecond)
		m.app.Quit()
	})

	buttons := container.New(
		layout.NewGridLayoutWithColumns(2),
		buttonRemoto,
		buttonPresencial,
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
	_ = json.Unmarshal([]byte(m.Interaction.AreaOptions), &area)

	selectWidget := widget.NewSelect(area.ValuesArea, func(selected string) { newArea = selected })
	selectWidget.PlaceHolder = "Selecione o local de trabalho"

	var pop dialog.Dialog

	acceptButton := widget.NewButton("✔ Aceitar", func() {
		if newArea == "" {
			dialog.ShowInformation("Erro", "Você precisa selecionar um local", m.win)
			return
		}

		if err := m.savePresenceToDB(&PresenceRecord{Response: "Presencial", Observation: observation, Area: newArea}); err != nil {
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
			Content: "Presença presencial registrada com sucesso.",
		})
		<-time.After(10 * time.Millisecond)
		m.app.Quit()
	})

	cancelButton := widget.NewButton("✖ Cancelar", func() {
		pop.Hide()
	})

	pop = dialog.NewCustomWithoutButtons("Local de Trabalho", container.NewVBox(
		widget.NewLabel("Selecione onde você está trabalhando presencialmente:"),
		selectWidget,
		container.New(
			layout.NewGridLayoutWithColumns(2),
			cancelButton,
			acceptButton,
		),
	), m.win)

	pop.Resize(fyne.NewSize(widthPopup, highPopup))
	pop.Show()
}

func (m *MainApp) savePresenceToDB(presence *PresenceRecord) error {
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

	if err := m.db.Save(&m.AppConfig).Error; err != nil {
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

	cancelBtn := widget.NewButton("✖ Cancelar", func() {
		onComplete()
	})

	buttons := container.NewHBox(layout.NewSpacer(), cancelBtn, saveBtn)

	mainForm := container.NewVBox(
		widget.NewLabel("Editar Headers:"),
		formContent,
		addBtn,
		buttons,
	)

	m.win.SetContent(container.NewVScroll(mainForm))
	m.win.Resize(fyne.NewSize(width, high))
	m.win.Show()
}

func (m *MainApp) showAreaConfigForm(onComplete func()) {
	var area arr
	if err := json.Unmarshal([]byte(m.Interaction.AreaOptions), &area); err != nil {
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
			entry.MultiLine = false
			entry.SetMinRowsVisible(2)

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

			row := container.NewBorder(nil, nil, nil, delBtn, entry)
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

		m.Interaction.AreaOptions = string(areaJSON)
		if err := m.db.Save(&m.Interaction).Error; err != nil {
			dialog.ShowError(fmt.Errorf("erro ao salvar no banco de dados: %w", err), m.win)
			return
		}

		dialog.ShowInformation("Sucesso", "Áreas atualizadas com sucesso", m.win)
		onComplete()
	})

	cancelBtn := widget.NewButton("✖ Cancelar", func() {
		onComplete()
	})

	buttons := container.NewHBox(layout.NewSpacer(), cancelBtn, saveBtn)

	content := container.NewVBox(
		widget.NewLabelWithStyle("Editar Áreas", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		formContainer,
		buttons,
	)

	refreshForm()

	m.win.SetContent(content)
	m.win.Resize(fyne.NewSize(width, high))
	m.win.Show()
}

func (m *MainApp) buildMainMenu() {
	m.win = m.app.NewWindow(m.Language.Title)
	m.win.SetFixedSize(true)

	fileMenu := fyne.NewMenu("Arquivo",
		fyne.NewMenuItem("Exportar Dados (JSON)", func() {
			dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
				if err != nil || writer == nil {
					return
				}
				defer func(writer fyne.URIWriteCloser) {
					if err := writer.Close(); err != nil {
						dialog.ShowError(err, m.win)
						return
					}
				}(writer)

				// Get the file path from the URI
				filePath := writer.URI().Path()
				if !strings.HasSuffix(filePath, ".json") {
					filePath += ".json"
				}

				if err := m.exportToJSON(filePath); err != nil {
					dialog.ShowError(err, m.win)
					return
				}

				dialog.ShowInformation("Sucesso", "Dados exportados com sucesso", m.win)
			}, m.win)
		}),
		fyne.NewMenuItem("Importar Dados (JSON)", func() {
			dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
				if err != nil || reader == nil {
					return
				}
				defer func(reader fyne.URIReadCloser) {
					if err := reader.Close(); err != nil {
						dialog.ShowError(err, m.win)
						return
					}
				}(reader)

				// Get the file path from the URI
				filePath := reader.URI().Path()

				if err := m.importFromJSON(filePath); err != nil {
					dialog.ShowError(err, m.win)
					return
				}

				dialog.ShowInformation("Sucesso", "Dados importados com sucesso", m.win)
				m.win.SetContent(m.buildMainContent())
			}, m.win)
		}),
		fyne.NewMenuItemSeparator(),
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

	if !m.firstRun {
		entryDefault.SetText(strconv.Itoa(m.AppConfig.DefaultGoal))
	}

	saveBtn := widget.NewButton("💾 Salvar", func() {
		if entryDefault.Text == "" {
			dialog.ShowError(fmt.Errorf("o valor precisa não pode estar vacio"), m.win)
			return
		}

		if err := m.updateGoal(entryDefault.Text); err != nil {
			dialog.ShowError(fmt.Errorf("erro ao atualizar meta: %w", err), m.win)
			return
		}

		dialog.ShowInformation("Salvo", "Configuração salva com sucesso", m.win)
		onComplete()
	})

	cancelBtn := widget.NewButton("✖ Cancelar", func() {
		onComplete()
	})

	buttons := container.NewHBox(layout.NewSpacer(), cancelBtn, saveBtn)

	form := container.NewVBox(
		widget.NewLabel("Configure os dias de presença:"),
		entryDefault,
		buttons,
	)

	m.win.SetContent(form)
	m.win.Resize(fyne.NewSize(width, high))
	m.win.Show()
}

func (m *MainApp) createDefaultApp() error {
	var count int64
	m.db.Model(&App{}).Count(&count)
	if count > 0 {
		return nil
	}

	m.Language = AppLanguage{
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

	m.Interaction = AppInteraction{
		ExtraLabel:  "adicional",
		AreaOptions: `{"areas": ["CT", "CEIC", "AG", "OUTRO"]}`,
		Headers:     `{"headers": ["data", "hora", "resposta", "observacao", "area"]}`,
	}

	if err := m.db.Create(&m.Interaction).Error; err != nil {
		return err
	}

	m.AppConfig = AppConfig{
		DefaultGoal: 4,
	}

	if err := m.db.Create(&m.AppConfig).Error; err != nil {
		return err
	}

	m.App = &App{
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
	var presencialCount int

	for _, r := range m.records {
		t, err := time.Parse(layoutISO, r.Date)
		if err != nil {
			continue
		}

		switch r.Response {
		case "Presencial":
			report += fmt.Sprintf("🏢 %s - %s (Presencial)\n", t.Format(layoutBR), r.Area)
			presencialCount++
		case "Remoto":
			report += fmt.Sprintf("🏠 %s - Trabalho Remoto\n", t.Format(layoutBR))
		default:
			report += fmt.Sprintf("☑️ %s - %s\n", t.Format(layoutBR), r.Area)
		}
	}

	// Only show pending for presencial goal
	for i := presencialCount; i < m.AppConfig.DefaultGoal; i++ {
		report += "🔲 (presencial pendente)\n"
	}

	return fmt.Sprintf("Você registrou %d dia(s) presencial(is) neste mês:\n\n%s", presencialCount, report)
}

// exportToJSON exports all presence records to a JSON file
func (m *MainApp) exportToJSON(filePath string) error {
	var allRecords []PresenceRecord
	if err := m.db.Order("date DESC, time DESC").Find(&allRecords).Error; err != nil {
		return fmt.Errorf("erro ao carregar registros: %w", err)
	}

	data, err := json.MarshalIndent(allRecords, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao serializar dados: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("erro ao salvar arquivo: %w", err)
	}

	return nil
}

// importFromJSON imports presence records from a JSON file
func (m *MainApp) importFromJSON(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("erro ao ler arquivo: %w", err)
	}

	var records []PresenceRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("erro ao processar JSON: %w", err)
	}

	// Validate records
	for i, record := range records {
		if record.Date == "" || record.Time == "" || record.Response == "" {
			return fmt.Errorf("registro inválido na posição %d: campos obrigatórios ausentes", i)
		}

		// Validate date format
		if _, err := time.Parse(layoutISO, record.Date); err != nil {
			return fmt.Errorf("formato de data inválido no registro %d: %s", i, record.Date)
		}
	}

	// Begin transaction
	tx := m.db.Begin()

	// Import records
	for _, record := range records {
		// Check if record already exists
		var count int64
		tx.Model(&PresenceRecord{}).
			Where("date = ? AND time = ?", record.Date, record.Time).
			Count(&count)

		if count == 0 {
			if err := tx.Create(&record).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("erro ao importar registro: %w", err)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("erro ao finalizar importação: %w", err)
	}

	// Reload current month records
	return m.loadConfigFromDB()
}

// setupTrayIcon configures the system tray icon and menu
func (m *MainApp) setupTrayIcon() {
	// Set up the systray icon
	if m.app != nil && m.app.Icon() != nil {
		systray.SetIcon(m.app.Icon().Content())
	} else {
		// Use a default icon from assets
		iconPath := filepath.Join("assets", "Bokehlicia-Captiva-Baobab-stats.64.png")
		if iconData, err := os.ReadFile(iconPath); err == nil {
			systray.SetIcon(iconData)
		} else {
			log.Printf("Failed to load icon: %v", err)
		}
	}
	systray.SetTitle("Presencial")
	systray.SetTooltip("Controle de Presença")

	// Create menu items
	mShow := systray.AddMenuItem("Mostrar Janela", "Mostrar a janela principal")
	systray.AddSeparator()
	mExport := systray.AddMenuItem("Exportar Dados (JSON)", "Exportar registros para JSON")
	mImport := systray.AddMenuItem("Importar Dados (JSON)", "Importar registros de JSON")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Sair", "Fechar o aplicativo")

	// Handle menu item clicks in a goroutine
	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				// Show the window from the main thread
				go func() {
					m.win.Show()
				}()
			case <-mExport.ClickedCh:
				// Show export dialog from the main thread
				go func() {
					// Create a temporary file path for export
					tempDir := m.getAppDataFolder("presencial")
					tempFile := filepath.Join(tempDir, "export_"+time.Now().Format("20060102_150405")+".json")

					// Export data to the temporary file
					if err := m.exportToJSON(tempFile); err != nil {
						m.app.SendNotification(&fyne.Notification{
							Title:   "Erro",
							Content: "Falha ao exportar dados: " + err.Error(),
						})
						return
					}

					// Show success notification
					m.app.SendNotification(&fyne.Notification{
						Title:   "Sucesso",
						Content: "Dados exportados para: " + tempFile,
					})
				}()
			case <-mImport.ClickedCh:
				// Show the window to allow user to use the import menu option
				go func() {
					m.win.Show()
					m.app.SendNotification(&fyne.Notification{
						Title:   "Importar Dados",
						Content: "Use o menu Arquivo > Importar Dados para selecionar um arquivo",
					})
				}()
			case <-mQuit.ClickedCh:
				systray.Quit()
				m.app.Quit()
				return
			}
		}
	}()
}

// RunApp start point to run the application logic
func (m *MainApp) RunApp() {
	// Initialize the system tray
	go func() {
		systray.Run(m.setupTrayIcon, func() {
			// Cleanup when systray exits
		})
	}()

	m.win.Resize(fyne.NewSize(width, high))
	m.win.CenterOnScreen()

	// Set window close handler to minimize to tray instead of quitting
	m.win.SetCloseIntercept(func() {
		m.win.Hide()
	})

	m.win.ShowAndRun()
}
