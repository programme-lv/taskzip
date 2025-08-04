package taskfs

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

func Write(task Task, dirPath string) error {
	dirAbsPath, err := filepath.Abs(dirPath)
	if err != nil {
		msg := fmt.Sprintf("get abs path of %s", dirPath)
		return wrap(msg, err)
	}

	if doesDirExist(dirAbsPath) {
		msg := fmt.Sprintf("dir %s already exists", dirAbsPath)
		return wrap(msg)
	}

	parentDir := filepath.Dir(dirAbsPath)
	if !doesDirExist(parentDir) {
		msg := fmt.Sprintf("parent dir %s does not exist", parentDir)
		return wrap(msg)
	}

	err = os.Mkdir(dirAbsPath, 0755)
	if err != nil {
		msg := fmt.Sprintf("create dir %s", dirAbsPath)
		return wrap(msg, err)
	}

	writer, err := NewTaskWriter(dirAbsPath, task)
	if err != nil {
		msg := fmt.Sprintf("init task dir writer %s", dirAbsPath)
		return wrap(msg, err)
	}

	return writer.WriteTask()
}

type TaskWriter struct {
	path string // absolute path to the task directory
	task *Task
}

func NewTaskWriter(
	taskDirAbsPath string,
	taskToWrite Task,
) (TaskWriter, error) {
	// check that the dir is empty
	files, err := os.ReadDir(taskDirAbsPath)
	if err != nil {
		msg := fmt.Sprintf("read task dir %s", taskDirAbsPath)
		return TaskWriter{}, wrap(msg, err)
	}
	if len(files) > 0 {
		msg := fmt.Sprintf("dir %s is not empty", taskDirAbsPath)
		return TaskWriter{}, wrap(msg)
	}

	// TODO: validate task

	return TaskWriter{
		path: taskDirAbsPath,
		task: &taskToWrite,
	}, nil
}

func (w TaskWriter) WriteTask() error {
	err := w.TaskToml()
	if err != nil {
		return err
	}
	err = w.Testlib()
	if err != nil {
		return err
	}
	err = w.Tests()
	if err != nil {
		return err
	}
	err = w.Solutions()
	if err != nil {
		return err
	}
	err = w.Readme()
	if err != nil {
		return err
	}
	err = w.Examples()
	if err != nil {
		return err
	}
	return nil
}

func (w TaskWriter) Readme() error {
	err := w.WriteFile("readme.md", []byte(w.task.ReadMe))
	if err != nil {
		msg := "write readme.md"
		return wrap(msg, err)
	}
	return nil
}

func (w TaskWriter) WriteFile(path string, content []byte) error {
	absPath := filepath.Join(w.path, path)
	err := os.WriteFile(absPath, content, 0755)
	if err != nil {
		msg := fmt.Sprintf("write file %s", path)
		return wrap(msg, err)
	}
	return nil
}

func (w TaskWriter) CreateDir(path string) error {
	absPath := filepath.Join(w.path, path)
	err := os.Mkdir(absPath, 0755)
	if err != nil {
		msg := fmt.Sprintf("create dir %s", path)
		return wrap(msg, err)
	}
	return nil
}

func (w TaskWriter) TaskToml() error {
	tomlPath := "task.toml"
	taskToml := NewTaskToml(w.task)

	buf := bytes.NewBuffer(nil)
	enc := toml.NewEncoder(buf)
	enc.SetTablesInline(false)
	enc.SetIndentTables(true)

	err := enc.Encode(taskToml)
	if err != nil {
		msg := fmt.Sprintf("encode task.toml file %s", tomlPath)
		return wrap(msg, err)
	}
	err = w.WriteFile(tomlPath, buf.Bytes())
	if err != nil {
		msg := fmt.Sprintf("write task.toml file %s", tomlPath)
		return wrap(msg, err)
	}
	return nil
}

func (w TaskWriter) Testlib() error {
	err := w.CreateDir("testlib")
	if err != nil {
		msg := "create testlib directory"
		return wrap(msg, err)
	}

	if w.task.Testing.Checker != "" {
		err := w.WriteFile("testlib/checker.cpp", []byte(w.task.Testing.Checker))
		if err != nil {
			msg := "write non-empty checker.cpp"
			return wrap(msg, err)
		}
	}

	if w.task.Testing.Interactor != "" {
		err := w.WriteFile("testlib/interactor.cpp", []byte(w.task.Testing.Interactor))
		if err != nil {
			msg := "write non-empty interactor.cpp"
			return wrap(msg, err)
		}
	}
	return nil
}

func (w TaskWriter) Tests() error {
	err := w.CreateDir("tests")
	if err != nil {
		msg := "create tests directory"
		return wrap(msg, err)
	}

	for i, test := range w.task.Testing.Tests {
		inPath := fmt.Sprintf("tests/%03di.txt", i+1)
		outPath := fmt.Sprintf("tests/%03do.txt", i+1)
		err := w.WriteFile(inPath, []byte(test.Input))
		if err != nil {
			msg := fmt.Sprintf("write test %d input", i)
			return wrap(msg, err)
		}
		err = w.WriteFile(outPath, []byte(test.Answer))
		if err != nil {
			msg := fmt.Sprintf("write test %d output", i)
			return wrap(msg, err)
		}
	}

	return nil
}

func (w TaskWriter) Solutions() error {
	solutionsDir := "solutions"
	err := w.CreateDir("solutions")
	if err != nil {
		msg := "create solutions directory"
		return wrap(msg, err)
	}

	for i, sol := range w.task.Solutions {
		solPath := filepath.Join(solutionsDir, sol.Fname)
		err := w.WriteFile(solPath, []byte(sol.Content))
		if err != nil {
			msg := fmt.Sprintf("write solution %d", i)
			return wrap(msg, err)
		}
	}
	return nil
}

func (w TaskWriter) Examples() error {
	examplesDir := "examples"
	err := w.CreateDir("examples")
	if err != nil {
		msg := "create examples directory"
		return wrap(msg, err)
	}

	for i, example := range w.task.Statement.Examples {
		inPath := filepath.Join(examplesDir, fmt.Sprintf("%03di.txt", i+1))
		outPath := filepath.Join(examplesDir, fmt.Sprintf("%03do.txt", i+1))
		notePath := filepath.Join(examplesDir, fmt.Sprintf("%03d.md", i+1))
		err := w.WriteFile(inPath, []byte(example.Input))
		if err != nil {
			msg := fmt.Sprintf("write example %d input", i)
			return wrap(msg, err)
		}
		err = w.WriteFile(outPath, []byte(example.Output))
		if err != nil {
			msg := fmt.Sprintf("write example %d output", i)
			return wrap(msg, err)
		}
		mdNoteContent := ""
		for lang, note := range example.MdNote {
			mdNoteContent += fmt.Sprintf("%s\n---\n%s\n", lang, note)
		}
		err = w.WriteFile(notePath, []byte(mdNoteContent))
		if err != nil {
			msg := fmt.Sprintf("write example %d note", i)
			return wrap(msg, err)
		}
	}
	return nil
}

func (w TaskWriter) TestGroups() error {
	filePath := "testgroups.txt"
	

}
