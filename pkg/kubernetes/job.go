package kubernetes

import (
	"context"
	"fmt"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/jjeffery/kv"
)

// This file contains the implementation of functions used to start and run
// task and perform other various tasks within a Kubernetes cluster

type Status struct {
	ID    string
	Level int
	Msg   kv.Error
}

func (task *Task) sendStatus(ctx context.Context, statusC chan *Status, level int, msg kv.Error) {
	select {
	case statusC <- &Status{ID: task.start.ID, Level: level, Msg: msg}:
		return
	case <-time.After(20 * time.Millisecond):
	}
	fmt.Println("ID", task.start.ID, spew.Sdump(msg))
}

func (task *Task) runJob(ctx context.Context, logger chan *Status) (err kv.Error) {
	//task.start.JobSpec
	return nil
}
