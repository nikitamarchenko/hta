package cmd_v2

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
	"github.com/nikitamarchenko/hta/internal/task"
)

func infoColor() func(a ...interface{}) string {
	return color.New(color.FgHiGreen).SprintFunc()
}

func warnColor() func(a ...interface{}) string {
	return color.New(color.FgRed).SprintFunc()
}

func idColor() func(a ...interface{}) string {
	return color.New(color.FgHiBlue).SprintFunc()
}

func addColor() func(a ...interface{}) string {
	return color.New(color.FgHiYellow).SprintFunc()
}

func helpColor() func(a ...interface{}) string {
	return color.New(color.FgHiBlack).SprintFunc()
}

type Model struct {
	cursor   int
	selected int
	Tasks    *task.TaskList
	sorted   bool
	lines    []int
	ti       *textinput.Model
}

func (m Model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	var cmd tea.Cmd
	if m.ti != nil {
		*m.ti, cmd = m.ti.Update(msg)
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter", "esc":
				m.Tasks.GetTaskById(m.lines[m.cursor]).Desc = m.ti.Value()
				m.ti = nil
			}
		}
		return m, cmd
	}

	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q", "ctrl+d":
			return m, tea.Quit

		case "s":
			m.sorted = !m.sorted

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.lines)-1 {
				m.cursor++
			}

		case "enter":
			m.Tasks.GetTaskById(m.lines[m.cursor]).Desc = m.ti.Value()
			m.ti = nil

		case "i":
			ti := textinput.New()
			ti.SetValue(m.Tasks.GetTaskById(m.lines[m.cursor]).Desc)
			ti.Focus()
			ti.CharLimit = 80
			ti.Width = 40
			m.ti = &ti
		}
	}

	var items []task.Task
	if m.sorted {
		_, items = m.Tasks.TopoSort()
	} else {
		items = *m.Tasks
	}

	m.lines = make([]int, len(items))
	for i, v := range items {
		m.lines[i] = v.Id
	}

	return m, cmd
}

func (m Model) View() string {
	s := "\n   Task list:\n\n"

	idColor := idColor()

	var items []task.Task
	if m.sorted {
		_, items = m.Tasks.TopoSort()
	} else {
		items = *m.Tasks
	}
	for _, v := range items {

		selectedLine := m.lines[m.cursor] == v.Id
		if selectedLine && m.ti != nil {
			s += "    " + m.ti.View() + "\n"
		} else {
			var d string
			if len(v.DependsOn) > 0 {
				var b []string
				for _, vv := range v.DependsOn {
					b = append(b, idColor(vv))
				}
				d = fmt.Sprintf(" > %s", strings.Join(b, " "))
			}

			var dfv []string
			for _, vvv := range m.Tasks.GetDependsFrom(v.Id) {
				dfv = append(dfv, idColor(vvv))
			}
			var df string
			if len(dfv) > 0 {
				df = fmt.Sprintf(" < %s", strings.Join(dfv, " "))
			}

			selected := " "
			if selectedLine {
				selected = ">"
			}
			tt := fmt.Sprintf("%3d", v.Id)
			s += fmt.Sprintf("%s%s: %s%s%s\n", selected, idColor(tt), v.Desc, d, df)
		}
	}
	// The footer
	s += "\n   Press q to quit.\n"

	return s
}

func Run(d int, f string) {

	tasks, err := task.LoadTasks(f)

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Printf("error on load tasks: %s\n", err)
		os.Exit(1)
	}

	m := Model{
		selected: 0,
		Tasks:    tasks,
		lines:    make([]int, len(*tasks)),
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
