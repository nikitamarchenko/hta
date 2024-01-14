package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/nikitamarchenko/hta/internal/task"
	"github.com/nikitamarchenko/hta/internal/util"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
)

type Ctx struct {
	Tasks  *task.TaskList
	w      io.Writer
	prompt []string
}

// command line args
var debug int
var filename string

func Run() {
	flag.IntVar(&debug, "debug", 0, "use debug for debug")
	flag.StringVar(&filename, "filename", "./hta.json", "hta db")
	flag.Parse()

	task.Debug = debug
	util.Debug = debug

	info := infoColor()

	tasks, err := task.LoadTasks(filename)

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Printf("error on load tasks: %s\n", err)
		os.Exit(1)
	}

	if tasks == nil {
		tasks = &task.TaskList{}
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
		util.DebugF("usr: [%s]\n", s)
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
		case "rn":
			renameCmd(ctx)
		}
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

		if !ctx.Tasks.TaskExists(int(pId64)) {
			return 0, 0, errors.New("p not found")
		}
		pId := int(pId64)

		dId64, err := strconv.ParseInt(ids[1], 10, 64)
		if err != nil {
			return 0, 0, errors.New("d a number")
		}

		if !ctx.Tasks.TaskExists(int(dId64)) {
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

			if can, chain := ctx.Tasks.CanTaskBeParent(pId, dId); !can {
				return errors.Join(
					fmt.Errorf("dep error chain %s", task.FormatTasksForError(*chain)), err)
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
		p := ctx.Tasks.GetTaskById(pId)
		d := ctx.Tasks.GetTaskById(dId)

		var text string
		if ctx.Tasks.IsTaskHasDeps(pId, dId) {
			p.RemoveDep(dId)
			text = fmt.Sprintf("%s %s not depends on %s %s",
				idColor(p.Id), p.Desc,
				idColor(d.Id), d.Desc)

		} else {
			ctx.Tasks.LinkTask(pId, dId)
			text = fmt.Sprintf("%s %s depends on %s %s",
				idColor(p.Id), p.Desc,
				idColor(d.Id), d.Desc)
		}
		ctx.printLn(info("ok"), text)
		task.SaveTasks(filename, *ctx.Tasks)
	}
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

	t := task.Task{Id: newId, Desc: s, DependsOn: []int{}}
	*ctx.Tasks = append(*ctx.Tasks, t)
	task.SaveTasks(filename, *ctx.Tasks)
	ctx.print("%s add %s: %s\n", addColor(""), idColor(t.Id), t.Desc)
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
			if !ctx.Tasks.TaskExists(int(id64)) {
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

		ctx.Tasks.TaskDelete(id)
		ctx.printLn(info("ok"))
		task.SaveTasks(filename, *ctx.Tasks)
	}
}

func listCmd(ctx *Ctx, sorted bool) {
	idColor := idColor()

	var items []task.Task
	if sorted {
		_, items = ctx.Tasks.TopoSort()
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
		tt := fmt.Sprintf("%3d", v.Id)
		ctx.print("%s: %s %s\n", idColor(tt), v.Desc, d)
	}
}

func renameCmd(ctx *Ctx) {
	info := infoColor()
	add := addColor()
	help := helpColor()
	idColor := idColor()
	ctx.addPrompt(add("rename") + info("❯"))
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
			if !ctx.Tasks.TaskExists(int(id64)) {
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
		t := ctx.Tasks.GetTaskById(id)
		ctx.addPrompt(idColor(id) + info("❯"))
		defer ctx.popPrompt()
		s := ctx.readUserInputWithDefault(nil, t.Desc)

		if s == "" {
			ctx.print("abort\n")
			return
		}
		t.Desc = s
		task.SaveTasks(filename, *ctx.Tasks)
		ctx.printLn(info("ok"))
		return
	}
}

func (ctx *Ctx) readUserInput(validate func(input string) error) string {
	return ctx.readUserInputWithDefault(validate, "")
}

func (ctx *Ctx) readUserInputWithDefault(
	validate func(input string) error, def string) string {
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
		Default:   def,
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
