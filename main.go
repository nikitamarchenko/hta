package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
)

type Task struct {
	Id        int    `json:"id"`
	Desc      string `json:"desc"`
	DependsOn []int  `json:"depends_on"`
	Closed    bool   `json:"closed"`
}

type TaskList []Task

type Ctx struct {
	Tasks  *TaskList
	w      io.Writer
	prompt []string
}

// command line args
var debug int
var filename string

func saveTasks(filename string, t TaskList) error {
	b, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	err = os.WriteFile(filename, b, 0644)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

func loadTasks(filename string) (*TaskList, error) {
	b, err := os.ReadFile(filename)

	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var t TaskList
	err = json.Unmarshal(b, &t)

	if err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	return &t, nil
}

func debugF(format string, a ...any) {
	if debug > 0 {
		fmt.Fprintf(os.Stdout, format, a...)
	}
}

func debug2F(format string, a ...any) {
	if debug > 1 {
		fmt.Fprintf(os.Stdout, format, a...)
	}
}

func debug2PrintTaskList(t []*Task) {
	if debug > 1 {
		for _, v := range t {
			var dl []string
			for _, d := range v.DependsOn {
				dl = append(dl, fmt.Sprint(d))
			}
			ds := strings.Join(dl, ",")
			fmt.Printf("[%d](%s)->", v.Id, ds)
		}
		fmt.Printf("\n")
	}
}

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

func main() {
	flag.IntVar(&debug, "debug", 1, "use debug for debug")
	flag.StringVar(&filename, "filename", "./hta.json", "hta db")
	flag.Parse()

	info := infoColor()
	fmt.Printf("This %s rocks!\n", info("HTA"))

	tasks, err := loadTasks(filename)

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Printf("error on load tasks: %s\n", err)
		os.Exit(1)
	}

	if tasks == nil {
		tasks = &TaskList{}
	}

	fmt.Printf("Loaded %s tasks\n", info(len(*tasks)))

	ctx := &Ctx{
		Tasks: tasks,
		w:     os.Stdout,
	}

	ctx.addPrompt(info("❯"))

	var s string
	for {
		s = ctx.readUserInput(nil)
		debugF("usr: [%s]\n", s)
		switch s {
		case "c":
			createCmd(ctx)
		case "d":
			deleteCmd(ctx)
		case "l":
			listCmd(ctx, false)
		case "ls":
			listCmd(ctx, true)
		case "ln":
			makeLinkCmd(ctx)
		}
	}
}

func (t *TaskList) topoSort() ([]int, []Task) {

	var order []int
	var orderItems []Task
	seen := make(map[int]bool)
	var visitAll func(items []int)

	m := make(map[int]Task)

	visitAll = func(items []int) {
		for _, item := range items {
			if !seen[item] {
				seen[item] = true
				visitAll(m[item].DependsOn)
				order = append(order, item)
				orderItems = append(orderItems, m[item])
			}
		}
	}

	var keys []int
	for _, v := range *t {
		keys = append(keys, v.Id)
		m[v.Id] = v
	}

	sort.Ints(keys)
	visitAll(keys)
	return order, orderItems
}

func makeLinkCmd(ctx *Ctx) {

	info := infoColor()
	warn := warnColor()
	help := helpColor()
	idColor := idColor()

	ctx.addPrompt(warn("ln") + info("❯"))
	defer ctx.popPrompt()

	parse := func(input string) (int, int, error) {
		ids := strings.Split(input, " ")
		if len(ids) != 2 {
			return 0, 0, errors.New("can't split values p d")
		}

		pId64, err := strconv.ParseInt(ids[0], 10, 64)
		if err != nil {
			return 0, 0, errors.New("p a number")
		}

		if !ctx.Tasks.taskExists(int(pId64)) {
			return 0, 0, errors.New("p not found")
		}
		pId := int(pId64)

		dId64, err := strconv.ParseInt(ids[1], 10, 64)
		if err != nil {
			return 0, 0, errors.New("d a number")
		}

		if !ctx.Tasks.taskExists(int(dId64)) {
			return 0, 0, errors.New("p not found")
		}
		dId := int(dId64)
		return pId, dId, nil
	}

	var s string
	for {
		s = ctx.readUserInput(func(input string) error {
			switch input {
			case "", "e", "l", "ls":
				return nil
			}
			pId, dId, err := parse(input)
			if err != nil {
				return err
			}

			if can, chain := ctx.Tasks.canTaskBeParent(pId, dId); !can {
				return errors.Join(
					fmt.Errorf("dep error chain %s", formatTasksForError(*chain)), err)
			}
			return nil
		})

		if s == "e" {
			break
		}

		if s == "l" {
			listCmd(ctx, false)
			continue
		}

		if s == "ls" {
			listCmd(ctx, true)
			continue
		}

		if s == "" {
			listCmd(ctx, false)
			ctx.printLn(help("  'e' for exit"))
			ctx.printLn(help("  'p d' p dep on d"))
			continue
		}

		pId, dId, _ := parse(s)
		p := ctx.Tasks.getTaskById(pId)
		d := ctx.Tasks.getTaskById(dId)

		var text string
		if ctx.Tasks.isTaskHasDeps(pId, dId) {
			p.removeDep(dId)
			text = fmt.Sprintf("%s %s not depends on %s %s",
				idColor(p.Id), p.Desc,
				idColor(d.Id), d.Desc)

		} else {
			ctx.Tasks.linkTask(pId, dId)
			text = fmt.Sprintf("%s %s depends on %s %s",
				idColor(p.Id), p.Desc,
				idColor(d.Id), d.Desc)
		}
		ctx.printLn(info("ok"), text)
		saveTasks(filename, *ctx.Tasks)
	}
}

func (t *TaskList) getTaskById(id int) *Task {
	for i, v := range *t {
		if v.Id == id {
			return &(*t)[i]
		}
	}
	return nil
}

func (t *TaskList) isTaskHasDeps(p int, d int) bool {
	for _, v := range *t {
		if p == v.Id {
			for _, vv := range v.DependsOn {
				if vv == d {
					return true
				}
			}
		}
	}
	return false
}

func (t *Task) removeDep(d int) {
	f := -1
	for i, vv := range t.DependsOn {
		if vv == d {
			f = i
		}
	}
	if f >= 0 {
		copy(t.DependsOn[f:], t.DependsOn[f+1:])
		t.DependsOn = t.DependsOn[:len(t.DependsOn)-1]
	}
}

func (t *TaskList) canTaskBeParent(pId, dId int) (bool, *[]*Task) {
	chain := []*Task{}
	chainResult := []*Task{}
	debug2F("check %d %d\n", pId, dId)
	var visit func(task *Task)
	visit = func(task *Task) {
		debug2F("on %d\n", task.Id)
		chain = append(chain, task)
		debug2F("def->")
		debug2PrintTaskList(chain)

		defer func() {
			debug2F("def-<")
			debug2PrintTaskList(chain)
			chain = chain[:len(chain)-1]
		}()

		if task.Id == pId {
			chainResult = make([]*Task, len(chain))
			copy(chainResult, chain)
			debug2F("FOUND\n")
			debug2F("%v %v", chainResult, chain)
			return
		}
		for _, v := range task.DependsOn {
			visit(t.getTaskById(v))
			if len(chainResult) > 0 {
				return
			}
		}
	}
	if d := t.getTaskById(dId); d != nil {
		visit(d)
	}

	if len(chainResult) > 0 {
		return false, &chainResult
	}

	if p := t.getTaskById(pId); p != nil {
		for _, pp := range p.DependsOn {
			visit(t.getTaskById(pp))
		}
	}

	return len(chainResult) == 0, &chainResult
}

func (t *TaskList) linkTask(pId, dId int) error {

	p := t.getTaskById(pId)
	d := t.getTaskById(dId)

	if p == nil {
		return fmt.Errorf("parent %d not found", pId)
	}

	if d == nil {
		return fmt.Errorf("dependent %d not found", dId)
	}

	if p == d {
		return fmt.Errorf("p and d the same task %d", pId)
	}

	for _, v := range p.DependsOn {
		if v == dId {
			return nil
		}
	}

	if can, chain := t.canTaskBeParent(pId, dId); !can {
		return fmt.Errorf("can't link %d with %d\n %v", pId, dId, chain)
	}

	p.DependsOn = append(p.DependsOn, d.Id)

	return nil
}

func createCmd(ctx *Ctx) {
	info := infoColor()
	idColor := idColor()
	addColor := addColor()
	ctx.addPrompt("new" + info("❯"))
	defer ctx.popPrompt()
	s := ctx.readUserInput(nil)

	if s == "" {
		ctx.print("abort\n")
		return
	}

	newId := 1
	for _, v := range *ctx.Tasks {
		if newId <= v.Id {
			newId = v.Id + 1
		}
	}

	t := Task{Id: newId, Desc: s, DependsOn: []int{}}
	*ctx.Tasks = append(*ctx.Tasks, t)
	saveTasks(filename, *ctx.Tasks)
	ctx.print("%s add %s: %s\n", addColor(""), idColor(t.Id), t.Desc)
}

func (t *TaskList) taskExists(id int) bool {
	for _, v := range *t {
		if id == v.Id {
			return true
		}
	}
	return false
}

func (t *TaskList) taskDelete(id int) {
	f := -1
	for i, v := range *t {
		if id == v.Id {
			f = i
		}
		t.getTaskById(v.Id).removeDep(id)
	}
	if f < 0 {
		return
	}
	copy((*t)[f:], (*t)[f+1:])
	*t = (*t)[:len(*t)-1]
}

func deleteCmd(ctx *Ctx) {
	info := infoColor()
	warn := warnColor()
	help := helpColor()
	ctx.addPrompt(warn("delete") + info("❯"))
	defer ctx.popPrompt()
	var s string
	for {
		var id int
		s = ctx.readUserInput(func(input string) error {
			if input == "" || input == "e" {
				return nil
			}
			id64, err := strconv.ParseInt(input, 10, 64)
			if err != nil {
				return errors.New("not a number")
			}
			if !ctx.Tasks.taskExists(int(id64)) {
				return errors.New("not found")
			}
			id = int(id64)
			return nil
		})

		if s == "e" {
			break
		}

		if s == "" {
			listCmd(ctx, false)
			ctx.printLn(help("  'e' for exit"))
			continue
		}

		ctx.Tasks.taskDelete(id)
		ctx.printLn(info("ok"))
		saveTasks(filename, *ctx.Tasks)
	}
}

func listCmd(ctx *Ctx, sorted bool) {
	idColor := idColor()

	var items []Task
	if sorted {
		_, items = ctx.Tasks.topoSort()
	} else {
		items = *ctx.Tasks
	}

	for _, v := range items {
		var d string
		if len(v.DependsOn) > 0 {
			var b []string
			for _, i := range v.DependsOn {
				b = append(b, idColor(i))
			}
			d = fmt.Sprintf("󰜴[%s]", strings.Join(b, ", "))
		}
		ctx.print(" %s : %s %s\n", idColor(v.Id), v.Desc, d)
	}
}

func (ctx *Ctx) readUserInput(validate func(input string) error) string {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	prompt := promptui.Prompt{
		Label:     ctx.createPrompt(),
		Templates: templates,
		Validate:  validate,
	}

	result, err := prompt.Run()

	if err != nil {
		fmt.Printf("exit\n")
		os.Exit(0)
	}

	return strings.TrimSpace(result)
}

func (ctx *Ctx) print(format string, a ...any) {
	fmt.Fprintf(ctx.w, format, a...)
}

func (ctx *Ctx) printLn(a ...any) {
	fmt.Fprintln(ctx.w, a...)
}

func (ctx *Ctx) createPrompt() string {
	return strings.Join(ctx.prompt, "")
}

func (ctx *Ctx) addPrompt(s string) {
	ctx.prompt = append(ctx.prompt, s)
}

func (ctx *Ctx) popPrompt() {
	ctx.prompt = ctx.prompt[:len(ctx.prompt)-1]
}

func formatTasksForError(tl []*Task) string {
	var b []string
	for _, t := range tl {
		b = append(b, fmt.Sprintf("[%d]%s", t.Id, t.Desc))
	}
	return strings.Join(b, "->")
}
