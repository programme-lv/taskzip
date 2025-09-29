package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/programme-lv/taskzip/common/etrace"
	"github.com/programme-lv/taskzip/external/lio/lio2023"
	"github.com/programme-lv/taskzip/external/lio/lio2024"
	"github.com/programme-lv/taskzip/taskfs"
)

func transform(src string, dst string, format string, zipOut bool) error {
	info("running transform\n\t- src: %s\n\t- dst parent dir: %s\n\t- format: %s", src, dst, format)

	srcDir, cleanup, err := extractToTmpIfZip(src)
	if err != nil {
		return etrace.Wrap("prepare src", err)
	}
	defer cleanup()

	var task taskfs.Task
	switch format {
	case "lio2023":
		task, err = lio2023.ParseLio2023TaskDir(srcDir)
	case "lio2024":
		task, err = lio2024.ParseLio2024TaskDir(srcDir)
	default:
		msg := fmt.Sprintf("unsupported task format: %s", format)
		return etrace.NewError(msg)
	}
	if err != nil {
		return etrace.Wrap("parsing task in transform cmd", err)
	}

	fmt.Printf("task: %+v\n", task.Origin)
	if err := task.Validate(); err != nil {
		if etrace.IsCritical(err) {
			fmt.Printf("err: %+v\n", err)
			msg := "validate task parsed"
			return etrace.Wrap(msg, err)
		}
		fmt.Printf("not critical err: %+v\n", err)
	}

	if zipOut {
		return writeTaskToZip(task, dst)
	}
	return writeTaskToDir(task, dst)
}

func writeTaskToDir(task taskfs.Task, dst string) error {
	path := filepath.Join(dst, task.ShortID)

	if dirExists(path) {
		ok, err := promptEraseExistingDir(path)
		if err != nil {
			return etrace.Wrap("prompt erase", err)
		}
		if ok {
			if err := os.RemoveAll(path); err != nil {
				return etrace.Wrap("remove dir", err)
			}
		}
	}

	if err := taskfs.Write(task, path); err != nil {
		return etrace.Wrap("write task", err)
	}

	info("successfully transformed task")
	printTaskOverview(task)
	readmePath := filepath.Join(path, "readme.md")
	if fileInfo, err := os.Stat(readmePath); err == nil && fileInfo.Size() > 0 {
		hint("check out %s\n", readmePath)
	}
	return nil
}

func writeTaskToZip(task taskfs.Task, dst string) error {
	zipPath := filepath.Join(dst, task.ShortID+".zip")
	if fileExists(zipPath) {
		ok, err := promptEraseExistingFile(zipPath)
		if err != nil {
			return etrace.Wrap("prompt erase", err)
		}
		if ok {
			if err := os.Remove(zipPath); err != nil {
				return etrace.Wrap("remove file", err)
			}
		}
	}

	if err := taskfs.WriteZip(task, zipPath); err != nil {
		return etrace.Wrap("write zip", err)
	}
	info("success; zip at %s", zipPath)
	hint("check out readme.md inside the .zip")
	printTaskOverview(task)
	return nil
}

func promptEraseExistingDir(dirPath string) (bool, error) {
	warn("dst dir %s already exists", dirPath)
	ask("delete it recursively and continue? [y/N]")
	answer, err := readAnswer()
	if err != nil {
		return false, etrace.Wrap("read answer", err)
	}
	implication := "continue without deleting"
	if answer == "y" || answer == "yes" {
		implication = "delete and continue"
	}
	info("received answer: %s (%s)", answer, implication)
	return answer == "y" || answer == "yes", nil
}

func promptEraseExistingFile(filePath string) (bool, error) {
	warn("dest file %s already exists", filePath)
	ask("delete it and continue? [y/N]")
	answer, err := readAnswer()
	if err != nil {
		return false, etrace.Wrap("read answer", err)
	}
	implication := "continue without deleting"
	if answer == "y" || answer == "yes" {
		implication = "delete and continue"
	}
	info("received answer: %s (%s)", answer, implication)
	return answer == "y" || answer == "yes", nil
}

func dirExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
