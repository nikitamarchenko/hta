package task

import (
	"encoding/json"
	"fmt"
	"github.com/nikitamarchenko/hta/internal/util"
	"os"
	"sort"
	"strings"
)

type Task struct {
	Id        int    `json:"id"`
	Desc      string `json:"desc"`
	DependsOn []int  `json:"depends_on"`
	Closed    bool   `json:"closed"`
}

type TaskList []Task

var Debug int

func SaveTasks(filename string, t TaskList) error {
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

func LoadTasks(filename string) (*TaskList, error) {
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

func debug2PrintTaskList(t []*Task) {
	if Debug > 1 {
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

func FormatTasksForError(tl []*Task) string {
	var b []string
	for _, t := range tl {
		b = append(b, fmt.Sprintf("[%d]%s", t.Id, t.Desc))
	}
	return strings.Join(b, "->")
}

func (t *TaskList) TopoSort() ([]int, []Task) {

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

func (t *TaskList) GetTaskById(id int) *Task {
	for i, v := range *t {
		if v.Id == id {
			return &(*t)[i]
		}
	}
	return nil
}

func (t *TaskList) IsTaskHasDeps(p int, d int) bool {
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

func (t *TaskList) GetDependsFrom(p int) []int {
	chain := []int{}
	for _, v := range *t {
		for _, vv := range v.DependsOn {
			if vv == p {
				chain = append(chain, v.Id)
			}
		}
	}
	return chain
}

func (t *Task) RemoveDep(d int) {
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

func (t *TaskList) CanTaskBeParent(pId, dId int) (bool, *[]*Task) {
	chain := []*Task{}
	chainResult := []*Task{}
	util.Debug2F("check %d %d\n", pId, dId)
	var visit func(task *Task)
	visit = func(task *Task) {
		util.Debug2F("on %d\n", task.Id)
		chain = append(chain, task)
		util.Debug2F("def->")
		debug2PrintTaskList(chain)

		defer func() {
			util.Debug2F("def-<")
			debug2PrintTaskList(chain)
			chain = chain[:len(chain)-1]
		}()

		if task.Id == pId {
			chainResult = make([]*Task, len(chain))
			copy(chainResult, chain)
			util.Debug2F("FOUND\n")
			util.Debug2F("%v %v", chainResult, chain)
			return
		}
		for _, v := range task.DependsOn {
			visit(t.GetTaskById(v))
			if len(chainResult) > 0 {
				return
			}
		}
	}
	if d := t.GetTaskById(dId); d != nil {
		visit(d)
	}

	if len(chainResult) > 0 {
		return false, &chainResult
	}

	if p := t.GetTaskById(pId); p != nil {
		for _, pp := range p.DependsOn {
			visit(t.GetTaskById(pp))
		}
	}

	return len(chainResult) == 0, &chainResult
}

func (t *TaskList) LinkTask(pId, dId int) error {

	p := t.GetTaskById(pId)
	d := t.GetTaskById(dId)

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

	if can, chain := t.CanTaskBeParent(pId, dId); !can {
		return fmt.Errorf("can't link %d with %d\n %v", pId, dId, chain)
	}

	p.DependsOn = append(p.DependsOn, d.Id)

	return nil
}

func (t *TaskList) TaskExists(id int) bool {
	for _, v := range *t {
		if id == v.Id {
			return true
		}
	}
	return false
}

func (t *TaskList) TaskDelete(id int) {
	f := -1
	for i, v := range *t {
		if id == v.Id {
			f = i
		}
		t.GetTaskById(v.Id).RemoveDep(id)
	}
	if f < 0 {
		return
	}
	copy((*t)[f:], (*t)[f+1:])
	*t = (*t)[:len(*t)-1]
}
