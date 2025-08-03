package taskfs

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pelletier/go-toml/v2"
	_ "github.com/pelletier/go-toml/v2"
)

type TaskDirReader struct {
	dirAbsPath string
	readPaths  map[string]bool // map of files that have been read
	allPaths   []string        // list of all files in the task directory
}

func NewTaskDir(dirAbsPath string) (TaskDirReader, error) {
	dirAbsPath, err := filepath.Abs(dirAbsPath)
	if err != nil {
		msg := "get absolute path"
		return TaskDirReader{}, wrap(msg, err)
	}
	allPaths, err := ReadAllPathsInDir(dirAbsPath)
	if err != nil {
		msg := "read all files"
		return TaskDirReader{}, wrap(msg, err)
	}
	return TaskDirReader{
		dirAbsPath: dirAbsPath,
		readPaths:  make(map[string]bool),
		allPaths:   allPaths,
	}, nil
}

func ReadAllPathsInDir(dirAbsPath string) ([]string, error) {
	dir, err := os.Open(dirAbsPath)
	if err != nil {
		msg := "open directory"
		return nil, wrap(msg, err)
	}
	defer dir.Close()
	files, err := dir.Readdir(0)
	if err != nil {
		msg := "list files"
		return nil, wrap(msg, err)
	}
	allPaths := make([]string, len(files))
	for i, file := range files {
		allPaths[i] = filepath.Join(dirAbsPath, file.Name())
		allPaths[i] = filepath.Clean(allPaths[i])
	}
	return allPaths, nil
}

func (dir TaskDirReader) ReadFile(relPath string) ([]byte, error) {
	joined := filepath.Join(dir.dirAbsPath, relPath)
	clean := filepath.Clean(joined)

	// ensure the path is within the task directory
	filePathRel, err := filepath.Rel(dir.dirAbsPath, clean)
	if err != nil {
		msg := "get relative path"
		return nil, wrap(msg, err)
	}
	if strings.Contains(filePathRel, "..") {
		msg := "path %s attempts to leave task directory"
		return nil, fmt.Errorf(msg, relPath)
	}

	bytes, err := os.ReadFile(clean)
	if err != nil {
		msg := "read file"
		return nil, wrap(msg, err)
	}
	return bytes, nil
}

func (dir TaskDirReader) TaskToml() (TaskToml, error) {
	content, err := dir.ReadFile("task.toml")
	if err != nil {
		msg := "read task.toml"
		return TaskToml{}, wrap(msg, err)
	}
	taskToml := TaskToml{}
	d := toml.NewDecoder(bytes.NewReader(content))
	d.DisallowUnknownFields()
	err = d.Decode(&taskToml)
	if err != nil {
		var details *toml.StrictMissingError
		if errors.As(err, &details) {
			msg := details.String()
			return TaskToml{}, wrap(msg, err)
		}
		msg := "decode task.toml"
		return TaskToml{}, wrap(msg, err)
	}
	return taskToml, nil
}

func (dir TaskDirReader) Checker() (string, error) {
	path := "testlib/checker.cpp"
	content, err := dir.ReadFile(path)
	if err != nil {
		msg := "read checker.cpp"
		return "", wrap(msg, err)
	}
	return string(content), nil
}

func (dir TaskDirReader) Interactor() (string, error) {
	path := "testlib/interactor.cpp"
	content, err := dir.ReadFile(path)
	if err != nil {
		msg := "read interactor.cpp"
		return "", wrap(msg, err)
	}
	return string(content), nil
}

func (dir TaskDirReader) Testing() (Testing, error) {
	taskToml, err := dir.TaskToml()
	if err != nil {
		msg := "read task.toml"
		return Testing{}, wrap(msg, err)
	}
	t := Testing{
		TestingT:   taskToml.Testing.Type,
		MemLimMiB:  taskToml.Testing.MemMiB,
		CpuLimMs:   taskToml.Testing.CpuMs,
		Tests:      []Test{},
		Checker:    "",
		Interactor: "",
	}
	if taskToml.Testing.Type == "checker" {
		checker, err := dir.Checker()
		if err != nil {
			msg := "type is checker"
			return Testing{}, wrap(msg, err)
		}
		if checker == "" {
			msg := "checker.cpp is empty"
			return Testing{}, wrap(msg)
		}
		t.Checker = checker
	}
	if taskToml.Testing.Type == "interactor" {
		interactor, err := dir.Interactor()
		if err != nil {
			msg := "type is interactor"
			return Testing{}, wrap(msg, err)
		}
		if interactor == "" {
			msg := "interactor.cpp is empty"
			return Testing{}, wrap(msg)
		}
		t.Interactor = interactor
	}
	err = t.Validate()
	if err != nil {
		msg := "invalid config"
		return Testing{}, wrap(msg, err)
	}
	return t, nil
}

func (dir TaskDirReader) Task() (Task, error) {
	taskToml, err := dir.TaskToml()
	if err != nil {
		msg := "read task.toml"
		return Task{}, wrap(msg, err)
	}
	testing, err := dir.Testing()
	if err != nil {
		msg := "construct testing"
		return Task{}, wrap(msg, err)
	}
	task := Task{
		Testing:   testing,
		ShortID:   taskToml.Id,
		FullName:  taskToml.Name,
		ReadMe:    "",
		Origin:    Origin{},
		Scoring:   Scoring{},
		Archive:   Archive{},
		Solutions: Solutions{},
		Metadata:  Metadata{},
	}
	return task, nil
}

func (dir TaskDirReader) AllFilesWereRead() bool {
	return len(dir.readPaths) == len(dir.allPaths)
}

type ReadConfig struct {
	checkAllFilesRead bool
}

func NewReadConfig(opts ...ReadOption) ReadConfig {
	config := ReadConfig{checkAllFilesRead: true}
	for _, opt := range opts {
		opt(&config)
	}
	return config
}

type ReadOption func(*ReadConfig)

func WithCheckAllFilesRead(check bool) ReadOption {
	return func(config *ReadConfig) {
		config.checkAllFilesRead = check
	}
}

func Read(dirPath string, opts ...ReadOption) (Task, error) {
	conf := NewReadConfig(opts...)

	dir, err := NewTaskDir(dirPath)
	if err != nil {
		msg := "init dir reader"
		return Task{}, wrap(msg, err)
	}

	task, err := dir.Task()
	if err != nil {
		msg := "read task"
		return Task{}, wrap(msg, err)
	}

	if conf.checkAllFilesRead && !dir.AllFilesWereRead() {
		msg := "not all files were read"
		return task, wrap(msg)
	}

	return task, nil
}

func wrap(msg string, errs ...error) error {
	_, file, line, _ := runtime.Caller(1)
	dir := filepath.Base(filepath.Dir(file))
	file = filepath.Base(file)
	if len(errs) > 0 {
		err := errors.Join(errs...)
		return fmt.Errorf("[%s/%s:%d] %s\n%w", dir, file, line, msg, err)
	} else {
		err := errors.New(msg)
		return fmt.Errorf("[%s/%s:%d] %w", dir, file, line, err)
	}
}
