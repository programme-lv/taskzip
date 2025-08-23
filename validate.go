package main

import (
	"github.com/programme-lv/taskzip/common/etrace"
	"github.com/programme-lv/taskzip/taskfs"
)

func validate(src string) error {
	info("running validate on %s", src)
	dir, cleanup, err := extractToTmpIfZip(src)
	if err != nil {
		return err
	}
	defer cleanup()

	task, err := taskfs.Read(dir)
	if err != nil {
		return etrace.Wrap("read task", err)
	}
	info("read task without errors")
	printTaskOverview(task)
	if err := task.Validate(); err != nil {
		return err
	}
	info("all good")
	return nil
}
