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


*/

import (
	"fmt"
	"os"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type status int

type mode int

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

type task struct {
	id          int
	title       string
	status      status
	description string
}

type model struct {
	todoList list.Model
	doneList list.Model

	activeList status

	inTitle textinput.Model
	inDesc  textinput.Model

	mode mode

	ready bool
}

func (t task) FilterValue() string {
	return t.title
}

func (t task) Title() string {
	return t.title
}

func (t task) Description() string {
	return t.description
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
		t.status = done
		m.doneList.InsertItem(len(m.doneList.Items()), t)
		m.todoList.RemoveItem(m.todoList.GlobalIndex())

		m.todoList.CursorUp()

	case done:
		if len(m.doneList.Items()) == 0 {
			return m
		}

		t := m.doneList.SelectedItem().(task)
		t.status = todo
		m.todoList.InsertItem(len(m.todoList.Items()), t)
		m.doneList.RemoveItem(m.doneList.GlobalIndex())

		m.doneList.CursorUp()

	}

	return m
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

	return model{inTitle: inTitle, inDesc: inDesc}
}

func (m model) initLists(width, height int) model {

	m.todoList = list.New([]list.Item{}, list.NewDefaultDelegate(), width, height)
	m.doneList = list.New([]list.Item{}, list.NewDefaultDelegate(), width, height)

	m.todoList.Title = "To Do"
	m.todoList.SetItems([]list.Item{
		task{status: todo, title: "Доделать агента СОМа", description: "очередь готова"},
		task{status: todo, title: "Заказать суши", description: "рили хочется"},
		task{status: todo, title: "Сходить к Гале", description: "к 19:00"},
	})

	m.doneList.Title = "Done"

	m.todoList.SetShowHelp(false)
	m.todoList.SetShowPagination(false)
	m.todoList.SetShowStatusBar(false)

	m.doneList.SetShowHelp(false)
	m.doneList.SetShowPagination(false)
	m.doneList.SetShowStatusBar(false)

	m.ready = true

	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) saveTask() model {

	title := m.inTitle.Value()
	desc := m.inDesc.Value()

	if desc == "" {
		desc = title
	}

	t := task{title: title, description: desc}
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
				return m, tea.Quit
			case "enter":
				m = m.switchStatus()
				return m, nil

			case "tab":
				m = m.switchList()
				return m, nil

			case "a", "ф":
				m.mode = addTitle
				return m, nil
			case "e", "у":
				switch m.activeList {
				case todo:
					m.inTitle.SetValue(m.todoList.SelectedItem().(task).title)
					m.inDesc.SetValue(m.todoList.SelectedItem().(task).description)
				case done:
					m.inTitle.SetValue(m.doneList.SelectedItem().(task).title)
					m.inDesc.SetValue(m.doneList.SelectedItem().(task).description)
				}

				m.mode = editTitle
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

					switch m.activeList {
					case todo:
						t := task{title: title, description: desc}

						m.todoList.SetItem(m.todoList.Cursor(), t)

					case done:
						t := task{title: title, description: desc}

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
		colWidth := msg.Width
		colHeight := msg.Height - 2 // Заголовок - Тело - Футер

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

	switch m.activeList {
	case todo:
		bodyStr = m.todoList.View()
	case done:
		bodyStr = m.doneList.View()
	}

	switch m.mode {
	case addTitle, editTitle:
		footerStr = m.inTitle.View()
	case addDesc, editDesc:
		footerStr = m.inDesc.View()

	default:
		footerStr = "(a) - add (e) - edit (d) - delete (tab) - todo\\done (s) - save (q) - quit"

	}

	vStr := lipgloss.JoinVertical(lipgloss.Top, bodyStr, footerStr)

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
