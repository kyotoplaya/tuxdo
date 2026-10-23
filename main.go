package main

/*
todo:
1. Понять почему вне case return m, nil ломает навигацию
2. Когда будет добавлен lipgloss сделать пометку избранных задач, важных и тд (цветом)

done:
1. Понять почему если менять статус у последнего элемента курсор пропадает в никуда:
индекс курсора после удаление элемента оставался равен индексу удаленного элемента.
.CursorUp() решил проблему, но необходимо проверять не оказался ли список пустой, иначе
метод при вызове выйдет за его границы

2. Реализовать режим edit

3. Сохранение

4. Оформление


*/

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// стиль для названия
var titleStyle = lipgloss.NewStyle().PaddingLeft(1)

// стиль для important задач
var importantStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6c6c"))

// стиль для work задач
var workStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#76fb8c"))

// стиль для home задач
var homeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#e4ef67"))

type status int

type mode int

type category int

const (
	todo status = iota
	done
)

const (
	nav mode = iota
	addTitle
	addDesc
	editTitle
	editDesc
)

const (
	trivial category = iota
	important
	work
	home
)

type task struct {
	Id              int      `json:"id"`
	TaskTitle       string   `json:"title"`
	Status          status   `json:"status"`
	TaskDescription string   `json:"description"`
	Category        category `json:"category"`
}

type model struct {
	todoList list.Model
	doneList list.Model

	activeList status

	inTitle textinput.Model
	inDesc  textinput.Model

	mode mode

	ready bool

	saveFlag bool // ture - изменений нет, bool - изменения есть

	width  int
	height int
}

type listDeligate struct {
	// потом тут поля для стилей
}

func (d listDeligate) Height() int { return 1 }

func (d listDeligate) Spacing() int { return 0 }

func (d listDeligate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d listDeligate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	t, ok := item.(task)
	if !ok {
		return
	}

	// выделен ли этот элемент
	isSelected := index == m.Index()

	// "> " для выделенного, "  " для остальных
	prefix := "  "
	if isSelected {
		prefix = "> "
	}

	var row string
	switch t.Category {
	case trivial:
		row = t.Title()
	case important:
		row = fmt.Sprintf("%s", importantStyle.Render(t.Title()))
	case work:
		row = fmt.Sprintf("%s", workStyle.Render(t.Title()))
	case home:
		row = fmt.Sprintf("%s", homeStyle.Render(t.Title()))
	}

	if isSelected {
		// для выделенного: жирный + акцентный цвет
		titleStyle = titleStyle.Bold(true).Foreground(lipgloss.Color("#ae6fff"))

		descStr := item.(task).Description()
		if descStr != "" {
			descStr = lipgloss.NewStyle().Render(descStr)

			if len(descStr) > m.Width()-1 {
				descStr = descStr[:m.Width()-4] + "..."
			}

			fmt.Fprintf(w, "%s%s: %s", prefix, titleStyle.Render(t.Title()), descStr)
		} else {
			fmt.Fprintf(w, "%s%s", prefix, titleStyle.Render(t.Title()))
		}

	} else {
		fmt.Fprintf(w, "%s%s", prefix, row)
	}

}

func (t task) FilterValue() string {
	return t.TaskTitle
}

func (t task) Title() string {
	return t.TaskTitle
}

func (t task) Description() string {
	return t.TaskDescription
}

func loadTasks() (todo, done []list.Item) {
	jsonBytes, err := os.ReadFile("tasks.json")
	if err != nil {
		if err != os.ErrNotExist { // Если файла просто нет - не считаем это ошибкой, он создаться при сохранении
			fmt.Println("Error while reading file:", err) // в ином случае выводим ошибку
		}

		return
	}

	rawJson := []task{}
	err = json.Unmarshal(jsonBytes, &rawJson)
	if err != nil {
		fmt.Println("Error while deserialization tasks:", err)
		return
	}

	todo = []list.Item{}
	done = []list.Item{}

	for _, t := range rawJson {
		if t.Status == 0 {
			todo = append(todo, t)
		} else {
			done = append(done, t)
		}
	}

	return todo, done

}

func saveTasksToFile(tasks []list.Item) {

	jsonData, err := json.Marshal(tasks)

	if err != nil {
		fmt.Println("Serialization error:", err)
		return
	}

	err = os.WriteFile("tasks.json", jsonData, 0644)
	if err != nil {
		fmt.Println("Error while writing done to file:", err)
		return
	}
}

func splitTaskLists(todoList, doneList []list.Item) []list.Item {
	tasks := append(todoList, doneList...)
	return tasks
}

func (m model) switchList() model {
	switch m.activeList {
	case todo:
		m.activeList = done
	case done:
		m.activeList = todo
	}

	return m
}

func (m model) switchStatus() model {
	switch m.activeList {
	case todo:
		if len(m.todoList.Items()) == 0 {
			return m
		}

		t := m.todoList.SelectedItem().(task)
		t.Status = done
		m.doneList.InsertItem(len(m.doneList.Items()), t)
		m.todoList.RemoveItem(m.todoList.GlobalIndex())

		m.todoList.CursorUp()

	case done:
		if len(m.doneList.Items()) == 0 {
			return m
		}

		t := m.doneList.SelectedItem().(task)
		t.Status = todo
		m.todoList.InsertItem(len(m.todoList.Items()), t)
		m.doneList.RemoveItem(m.doneList.GlobalIndex())

		m.doneList.CursorUp()

	}

	return m
}

func nextCategory(c category) category {
	switch c {
	case trivial:
		return important
	case important:
		return work
	case work:
		return home
	case home:
		return trivial
	}

	return trivial
}

func strCategory(c category) string {
	switch c {
	case trivial:
		return "trivial"
	case important:
		return "important"
	case work:
		return "work"
	case home:
		return "home"
	}

	return ""
}

func (m model) nextMode() mode {
	switch m.mode {
	case addTitle:
		return addDesc
	case addDesc:
		return nav
	case editTitle:
		return editDesc
	case editDesc:
		return nav
	}

	return nav
}

func initialModel() model {
	inTitle := textinput.New()
	inTitle.Placeholder = "Task title..."
	inTitle.SetWidth(40)

	inDesc := textinput.New()
	inDesc.Placeholder = "Task description..."
	inDesc.SetWidth(100)

	return model{
		inTitle:  inTitle,
		inDesc:   inDesc,
		saveFlag: true, // при загрузке задач изменений нет
	}
}

func (m model) initLists(width, height int) model {
	d := listDeligate{}

	m.todoList = list.New([]list.Item{}, d, width, height)
	m.doneList = list.New([]list.Item{}, d, width, height)

	// m.todoList.SetItems([]list.Item{
	// 	task{Status: todo, TaskTitle: "Доделать агента СОМа", TaskDescription: "очередь готова"},
	// 	task{Status: todo, TaskTitle: "Заказать суши", TaskDescription: "рили хочется"},
	// 	task{Status: todo, TaskTitle: "Сходить к Гале", TaskDescription: "к 19:00"},
	// })

	m.todoList.SetShowTitle(false)
	m.todoList.SetShowHelp(false)
	m.todoList.SetShowPagination(false)
	m.todoList.SetShowStatusBar(false)

	m.doneList.SetShowTitle(false)
	m.doneList.SetShowHelp(false)
	m.doneList.SetShowPagination(false)
	m.doneList.SetShowStatusBar(false)

	todoList, doneList := loadTasks()
	m.todoList.SetItems(todoList)
	m.doneList.SetItems(doneList)

	m.ready = true

	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) saveTask() model {

	title := m.inTitle.Value()
	desc := m.inDesc.Value()

	t := task{TaskTitle: title, TaskDescription: desc}
	m.todoList.InsertItem(len(m.todoList.Items()), t)

	m.inTitle.Reset()
	m.inDesc.Reset()

	return m
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:

		switch m.mode {
		case nav:
			switch msg.String() {
			case "q", "й":
				tasks := splitTaskLists(m.todoList.Items(), m.doneList.Items())
				saveTasksToFile(tasks)
				return m, tea.Quit
			case "enter":
				m = m.switchStatus()
				return m, nil

			case "tab":
				m = m.switchList()
				return m, nil

			case "d", "в":
				switch m.activeList {
				case todo:
					if len(m.todoList.Items()) > 0 {
						m.todoList.RemoveItem(m.todoList.GlobalIndex())
					}
					if len(m.todoList.Items()) > 0 {
						m.todoList.CursorUp()
					}
				case done:
					if len(m.doneList.Items()) > 0 {
						m.doneList.RemoveItem(m.todoList.GlobalIndex())
					}
					if len(m.doneList.Items()) > 0 {
						m.doneList.CursorUp()
					}
				}

				m.saveFlag = false

			case "a", "ф":
				m.mode = addTitle
				m.saveFlag = false
				return m, nil
			case "e", "у":
				switch m.activeList {
				case todo:
					m.inTitle.SetValue(m.todoList.SelectedItem().(task).TaskTitle)
					m.inDesc.SetValue(m.todoList.SelectedItem().(task).TaskDescription)
				case done:
					m.inTitle.SetValue(m.doneList.SelectedItem().(task).TaskTitle)
					m.inDesc.SetValue(m.doneList.SelectedItem().(task).TaskDescription)
				}

				m.mode = editTitle
				m.saveFlag = false
				return m, nil

			case "s", "ы":
				tasks := splitTaskLists(m.todoList.Items(), m.doneList.Items())
				saveTasksToFile(tasks)

				m.saveFlag = true

			case "c", "с":
				switch m.activeList {
				case todo:
					t := m.todoList.SelectedItem().(task)
					t.Category = nextCategory(t.Category)
					m.todoList.SetItem(m.todoList.Cursor(), t)
				case done:
					t := m.doneList.SelectedItem().(task)
					t.Category = nextCategory(t.Category)
					m.doneList.SetItem(m.doneList.Cursor(), t)
				}

				m.saveFlag = false

				return m, nil
			}

			// return m, nil - понять почему так не работает

		case addTitle, editTitle:
			m.inTitle.Focus()

			switch msg.String() {
			case "esc":
				m.inTitle.Reset()
				m.inTitle.Blur()
				m.mode = nav
				return m, nil

			case "enter":
				title := m.inTitle.Value()
				if title == "" {
					m.inTitle, cmd = m.inTitle.Update(msg)
					return m, cmd
				}
				m.inTitle.Blur()

				m.mode = m.nextMode()

				return m, nil
			}

			m.inTitle, cmd = m.inTitle.Update(msg)
			return m, cmd

		case addDesc, editDesc:
			m.inDesc.Focus()

			switch msg.String() {
			case "esc":
				m.inDesc.Reset()
				m.inDesc.Blur()
				m.mode = nav

				return m, nil

			case "enter":
				m.inDesc.Blur()

				switch m.mode {
				case addDesc:
					m = m.saveTask()
				case editDesc:
					title := m.inTitle.Value()
					desc := m.inDesc.Value()

					m.inTitle.Reset()
					m.inDesc.Reset()

					switch m.activeList {
					case todo:
						t := task{TaskTitle: title, TaskDescription: desc}

						m.todoList.SetItem(m.todoList.Cursor(), t)

					case done:
						t := task{TaskTitle: title, TaskDescription: desc}

						m.doneList.SetItem(m.todoList.Cursor(), t)
					}
				}

				m.mode = m.nextMode()

				return m, nil
			}

			m.inDesc, cmd = m.inDesc.Update(msg)
			return m, cmd

		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		colWidth := msg.Width
		colHeight := msg.Height - 5 // Заголовок - Тело - Футер - рамка (2)

		if !m.ready {
			m = m.initLists(colWidth, colHeight)
		} else {
			m.todoList.SetSize(colWidth, colHeight)
			m.doneList.SetSize(colWidth, colHeight)
		}

		return m, nil
	}

	if !m.ready {
		return m, nil
	}

	switch m.activeList {
	case todo:
		m.todoList, cmd = m.todoList.Update(msg)
	case done:
		m.doneList, cmd = m.doneList.Update(msg)
	}

	return m, cmd
}

func (m model) View() tea.View {
	var bodyStr, footerStr string
	var savedStrString string
	var titleStr string
	var t task
	var hasTask bool

	switch m.activeList {
	case todo:
		titleStr = fmt.Sprintf("To Do (%d)", len(m.todoList.Items()))
		bodyStr = lipgloss.NewStyle().Width(m.width).Border(lipgloss.NormalBorder()).Foreground(lipgloss.Color("#FFFFFF")).Render(m.todoList.View())
		if item := m.todoList.SelectedItem(); item != nil {
			t, hasTask = item.(task)
		}
	case done:
		titleStr = fmt.Sprintf("Done (%d)", len(m.doneList.Items()))
		bodyStr = lipgloss.NewStyle().Width(m.width).Border(lipgloss.NormalBorder()).Foreground(lipgloss.Color("#FFFFFF")).Render(m.doneList.View())
		if item := m.doneList.SelectedItem(); item != nil {
			t, hasTask = item.(task)
		}
	}

	titleStr = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, titleStr)                                                                // размещаем по центру
	titleStr = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#5f5fd7")).Render(titleStr) // красим-форматируем текст                                                             // размещаем его по центру
	titleStr = lipgloss.NewStyle().Width(m.width).Render(titleStr)                                                                         // растягиваем форматирование на всю строку

	switch m.mode {
	case addTitle, editTitle:
		footerStr = m.inTitle.View()
	case addDesc, editDesc:
		footerStr = m.inDesc.View()

	default:
		footerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Width(m.width)

		catStr := "-"
		if hasTask {
			catStr = strCategory(t.Category)
			switch t.Category {
			case important:
				catStr = importantStyle.Render(catStr)
			case work:
				catStr = workStyle.Render(catStr)
			case home:
				catStr = homeStyle.Render(catStr)
			}
		}

		footerStr = fmt.Sprintf("(a) - add (e) - edit (d) - delete (tab) - todo\\done (с) - cat %s (s) - save (q) - quit", catStr)
		footerStr = footerStyle.Render(footerStr)

		switch m.saveFlag {
		case true:
			savedStrString = lipgloss.NewStyle().Foreground(lipgloss.Color("#11ff19")).Bold(true).Render("Saved")
		case false:
			savedStrString = lipgloss.NewStyle().Foreground(lipgloss.Yellow).Bold(true).Render("Unsaved")

		}

	}

	vStr := lipgloss.JoinVertical(lipgloss.Top, titleStr, bodyStr, footerStr, savedStrString)

	v := tea.NewView(vStr)
	v.AltScreen = true
	return v
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
