package main

import (
	"fmt"
	"strings"
	"testing"
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
	taskDelete(&tasks, 5)
	if toString(tasks) != "12346789" {
		t.Errorf("error 5")
	}
	taskDelete(&tasks, 1)
	if toString(tasks) != "2346789" {
		t.Errorf("error 1")
	}
	taskDelete(&tasks, 9)
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
	
	debug = 1
	
	if err := tasks.linkTask(1, 2); err!= nil {
		t.Errorf("can't link 1 and 2: %s", err)
	}

	if err := tasks.linkTask(2, 3); err!= nil {
		t.Errorf("can't link 2 and 3: %s", err)
	}

	if err := tasks.linkTask(1, 1); err == nil {
		t.Error("can link 1 and 1")
	}

	if can, chain := tasks.canTaskBeParent(3, 1); can {
		t.Errorf("can link 3 and 1 %v", chain)
	}

	if can, chain := tasks.canTaskBeParent(1, 3); !can {
		t.Errorf("can't link 1 and 3 %v", chain)
	}

	if can, chain := tasks.canTaskBeParent(3, 2); can {
		t.Errorf("can link 3 and 2 %v", chain)
	}

	if can, chain := tasks.canTaskBeParent(2, 1); can {
		t.Errorf("can link 2 and 1 %v", chain)
	}
}

func TestGetTaskById(t *testing.T)  {
	
	tasks := TaskList{
		Task{Id: 1},
		Task{Id: 2},
		Task{Id: 3},
	}

	if tasks.getTaskById(10) != nil {
		t.Errorf("found not existed task")
	}

	task1 := tasks.getTaskById(1)

	task1.Desc = "test"

	task2 := tasks.getTaskById(1)

	if task1 != task2 {
		t.Errorf("getTaskById return copy of object")
	}

	if task1.Desc != "test" {
		t.Errorf("getTaskById return copy of object")
	}
}


func TestTopoSort(t *testing.T)  {
	
	tasks := TaskList{
		Task{Id: 1},
		Task{Id: 2},
		Task{Id: 3},
		Task{Id: 4},
		Task{Id: 5},
	}

	tasks.linkTask(1, 2)
	tasks.linkTask(2, 3)
	tasks.linkTask(4, 5)
	tasks.linkTask(3, 5)
	
	order, items := tasks.topoSort()
	t.Logf("%#v\n", order)
	t.Logf("%#v\n", items)
	
}