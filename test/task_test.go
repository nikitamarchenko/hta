package task_test

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/nikitamarchenko/hta/internal/task"
	"github.com/stretchr/testify/assert"
)

func toString(t []Task) string {
	b := strings.Builder{}
	for _, v := range t {
		b.WriteString(fmt.Sprint(v.Id))
	}
	return b.String()
}

func TestDeleteTask(t *testing.T) {

	tasks := TaskList{
		Task{Id: 1},
		Task{Id: 2},
		Task{Id: 3},
		Task{Id: 4},
		Task{Id: 5},
		Task{Id: 6},
		Task{Id: 7},
		Task{Id: 8},
		Task{Id: 9},
	}
	tasks.TaskDelete(5)
	if toString(tasks) != "12346789" {
		t.Errorf("error 5")
	}
	tasks.TaskDelete(1)
	if toString(tasks) != "2346789" {
		t.Errorf("error 1")
	}
	tasks.TaskDelete(9)
	if toString(tasks) != "234678" {
		t.Errorf("error 9")
	}
	if len(tasks) != 6 {
		t.Errorf("error len")
	}
}

func TestCanTaskBeParent(t *testing.T) {

	tasks := TaskList{
		Task{Id: 1},
		Task{Id: 2},
		Task{Id: 3},
	}

	Debug = 1

	if err := tasks.LinkTask(1, 2); err != nil {
		t.Errorf("can't link 1 and 2: %s", err)
	}

	if err := tasks.LinkTask(2, 3); err != nil {
		t.Errorf("can't link 2 and 3: %s", err)
	}

	if err := tasks.LinkTask(1, 1); err == nil {
		t.Error("can link 1 and 1")
	}

	if can, chain := tasks.CanTaskBeParent(3, 1); can {
		t.Errorf("can link 3 and 1 %v", chain)
	}

	if can, chain := tasks.CanTaskBeParent(1, 3); !can {
		t.Errorf("can't link 1 and 3 %v", chain)
	}

	if can, chain := tasks.CanTaskBeParent(3, 2); can {
		t.Errorf("can link 3 and 2 %v", chain)
	}

	if can, chain := tasks.CanTaskBeParent(2, 1); can {
		t.Errorf("can link 2 and 1 %v", chain)
	}
}

func TestGetTaskById(t *testing.T) {

	tasks := TaskList{
		Task{Id: 1},
		Task{Id: 2},
		Task{Id: 3},
	}

	if tasks.GetTaskById(10) != nil {
		t.Errorf("found not existed task")
	}

	task1 := tasks.GetTaskById(1)

	task1.Desc = "test"

	task2 := tasks.GetTaskById(1)

	if task1 != task2 {
		t.Errorf("GetTaskById return copy of object")
	}

	if task1.Desc != "test" {
		t.Errorf("GetTaskById return copy of object")
	}
}

func TestTopoSort(t *testing.T) {

	tasks := TaskList{
		Task{Id: 1},
		Task{Id: 2},
		Task{Id: 3},
		Task{Id: 4},
		Task{Id: 5},
	}

	tasks.LinkTask(1, 2)
	tasks.LinkTask(2, 3)
	tasks.LinkTask(4, 5)
	tasks.LinkTask(3, 5)

	order, items := tasks.TopoSort()
	t.Logf("%#v\n", order)
	t.Logf("%#v\n", items)

	assert.Equal(t, order, []int{5, 3, 2, 1, 4})
}
