package taskfs

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
	_ "github.com/pelletier/go-toml/v2"
)

type TaskDirReader struct {
	dirAbsPath string
	readPaths  map[string]bool // map of relative paths that have been read
	allPaths   []string        // list of all relative paths in the task directory
}

func NewTaskDir(dirAbsPath string) (TaskDirReader, error) {
	dirAbsPath, err := filepath.Abs(dirAbsPath)
	if err != nil {
		msg := "get absolute path"
		return TaskDirReader{}, wrap(msg, err)
	}
	dir := TaskDirReader{
		dirAbsPath: dirAbsPath,
		readPaths:  make(map[string]bool),
	}
	err = dir.readAllPathsInDir()
	if err != nil {
		msg := "read all paths in dir"
		return TaskDirReader{}, wrap(msg, err)
	}
	return dir, nil
}

func (dir *TaskDirReader) readAllPathsInDir() error {
	var totalSize int64
	var fileCount int

	err := filepath.WalkDir(dir.dirAbsPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				msg := "get file info"
				return wrap(msg, err)
			}

			totalSize += info.Size()
			fileCount++

			if totalSize > 512*1024*1024 { // 512 MB
				msg := "directory exceeds maximum size of 512 MB"
				return wrap(msg)
			}
			if fileCount > 10000 {
				msg := "directory contains more than 10000 files"
				return wrap(msg)
			}

			relPath, err := filepath.Rel(dir.dirAbsPath, path)
			if err != nil {
				msg := "get relative path"
				return wrap(msg, err)
			}
			dir.allPaths = append(dir.allPaths, relPath)
		}
		return nil
	})
	if err != nil {
		return wrap("walk directory", err)
	}
	return nil
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
		msg := fmt.Sprintf("path %s attempts to leave task directory", relPath)
		return nil, wrap(msg)
	}

	bytes, err := os.ReadFile(clean)
	if err != nil {
		msg := "read file"
		return nil, wrap(msg, err)
	}
	dir.readPaths[filePathRel] = true
	return bytes, nil
}

func (dir TaskDirReader) ListDir(dirRelPath string) ([]string, error) {
	prefix := dirRelPath + string(filepath.Separator)

	var paths []string
	for _, path := range dir.allPaths {
		if strings.HasPrefix(path, prefix) {
			rel := strings.TrimPrefix(path, prefix)
			if !strings.Contains(rel, string(filepath.Separator)) {
				paths = append(paths, rel)
			}
		}
	}
	return paths, nil
}

func (dir TaskDirReader) Toml() (TaskToml, error) {
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

func (dir TaskDirReader) Tests() ([]Test, error) {
	testDirPath := "tests"
	testFilePaths, err := dir.ListDir(testDirPath)
	if err != nil {
		msg := "list tests"
		return nil, wrap(msg, err)
	}
	if len(testFilePaths)%2 != 0 {
		msg := "number of tests must be even"
		return nil, wrap(msg)
	}
	if len(testFilePaths) > 999*2 {
		msg := "max 999 tests allowed (2 files per test)"
		return nil, wrap(msg)
	}
	slices.Sort(testFilePaths)
	tests := make([]Test, len(testFilePaths)/2)
	for i := 0; i < len(testFilePaths); i += 2 {
		expInFname := fmt.Sprintf("%03di.txt", (i/2)+1)
		expOutFname := fmt.Sprintf("%03do.txt", (i/2)+1)
		if testFilePaths[i] != expInFname {
			msg := "input test file path is incorrect"
			return nil, wrap(msg)
		}
		if testFilePaths[i+1] != expOutFname {
			msg := "output test file path is incorrect"
			return nil, wrap(msg)
		}
		inPath := filepath.Join(testDirPath, testFilePaths[i])
		input, err := dir.ReadFile(inPath)
		if err != nil {
			msg := "read test input"
			return nil, wrap(msg, err)
		}
		outPath := filepath.Join(testDirPath, testFilePaths[i+1])
		output, err := dir.ReadFile(outPath)
		if err != nil {
			msg := "read test output"
			return nil, wrap(msg, err)
		}
		tests[i/2] = Test{Input: string(input), Answer: string(output)}
	}
	return tests, nil
}

func (dir TaskDirReader) Testing() (Testing, error) {
	taskToml, err := dir.Toml()
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
	tests, err := dir.Tests()
	if err != nil {
		msg := "read tests"
		return Testing{}, wrap(msg, err)
	}
	t.Tests = tests
	err = t.Validate()
	if err != nil {
		msg := "invalid config"
		return Testing{}, wrap(msg, err)
	}
	return t, nil
}

func (dir TaskDirReader) Readme() (string, error) {
	content, err := dir.ReadFile("readme.md")
	if err != nil {
		msg := "read readme.md"
		return "", wrap(msg, err)
	}
	return string(content), nil
}

func (dir TaskDirReader) Origin() (Origin, error) {
	taskToml, err := dir.Toml()
	if err != nil {
		msg := "read task.toml"
		return Origin{}, wrap(msg, err)
	}

	o := Origin{
		Olympiad: taskToml.Origin.Olymp,
		OlyStage: taskToml.Origin.Stage,
		Org:      taskToml.Origin.Org,
		Notes:    taskToml.Origin.Notes,
		Authors:  taskToml.Origin.Authors,
		Year:     taskToml.Origin.Year,
	}
	err = o.Validate()
	if err != nil {
		msg := "invalid origin"
		return Origin{}, wrap(msg, err)
	}
	return o, nil
}

func (dir TaskDirReader) Metadata() (Metadata, error) {
	taskToml, err := dir.Toml()
	if err != nil {
		msg := "read task.toml"
		return Metadata{}, wrap(msg, err)
	}

	m := Metadata{
		ProblemTags: taskToml.Metadata.Tags,
		Difficulty:  taskToml.Metadata.Difficulty,
	}
	err = m.Validate()
	if err != nil {
		msg := "invalid metadata"
		return Metadata{}, wrap(msg, err)
	}
	return m, nil
}

func (dir TaskDirReader) Solutions() ([]Solution, error) {
	taskToml, err := dir.Toml()
	if err != nil {
		msg := "read task.toml"
		return []Solution{}, wrap(msg, err)
	}

	solutions := make([]Solution, len(taskToml.Solutions))
	for i, solutionToml := range taskToml.Solutions {
		solPath := filepath.Join("solutions", solutionToml.Fname)
		content, err := dir.ReadFile(solPath)
		if err != nil {
			msg := fmt.Sprintf("read solution file %s", solutionToml.Fname)
			return []Solution{}, wrap(msg, err)
		}

		solutions[i] = Solution{
			Fname:    solutionToml.Fname,
			Subtasks: solutionToml.Subtasks,
			Content:  string(content),
		}
	}

	return solutions, nil
}

func (dir TaskDirReader) Task() (Task, error) {
	taskToml, err := dir.Toml()
	if err != nil {
		msg := "read task.toml"
		return Task{}, wrap(msg, err)
	}
	testing, err := dir.Testing()
	if err != nil {
		msg := "construct testing"
		return Task{}, wrap(msg, err)
	}
	readme, err := dir.Readme()
	if err != nil {
		msg := "read readme.md"
		return Task{}, wrap(msg, err)
	}
	origin, err := dir.Origin()
	if err != nil {
		msg := "read origin"
		return Task{}, wrap(msg, err)
	}
	metadata, err := dir.Metadata()
	if err != nil {
		msg := "read metadata"
		return Task{}, wrap(msg, err)
	}
	solutions, err := dir.Solutions()
	if err != nil {
		msg := "read solutions"
		return Task{}, wrap(msg, err)
	}
	task := Task{
		Testing:   testing,
		ShortID:   taskToml.Id,
		FullName:  taskToml.Name,
		ReadMe:    readme,
		Origin:    origin,
		Scoring:   Scoring{},
		Archive:   Archive{},
		Solutions: solutions,
		Metadata:  metadata,
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
