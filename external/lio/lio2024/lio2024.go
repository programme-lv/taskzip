package lio2024

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/programme-lv/task-zip/common/errwrap"
	"github.com/programme-lv/task-zip/external/lio"
	"github.com/programme-lv/task-zip/taskfs"
)

func ParseLio2024TaskDir(dirPath string) (taskfs.Task, error) {
	parsedYaml, err := readYaml(dirPath)
	if err != nil {
		return taskfs.Task{}, errwrap.AddTrace(err)
	}

	task := base(parsedYaml)

	var errs []error

	if err := testing(&task, dirPath, parsedYaml); err != nil {
		errs = append(errs, errwrap.AddTrace(err))
	}

	if err := tests(&task, dirPath, parsedYaml); err != nil {
		errs = append(errs, errwrap.AddTrace(err))
	}

	if err := scoring(&task, parsedYaml); err != nil {
		errs = append(errs, errwrap.AddTrace(err))
	}

	if err := statement(&task, dirPath, task.Testing.TestingT); err != nil {
		errs = append(errs, errwrap.AddTrace(err))
	}

	if err := solutions(&task, dirPath); err != nil {
		errs = append(errs, errwrap.AddTrace(err))
	}

	if err := archive(&task, dirPath, parsedYaml); err != nil {
		errs = append(errs, errwrap.AddTrace(err))
	}

	return task, errors.Join(errs...)
}

func readYaml(dirPath string) (ParsedLio2024Yaml, error) {
	taskYamlPath := filepath.Join(dirPath, "task.yaml")

	taskYamlContent, err := os.ReadFile(taskYamlPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			msg := fmt.Sprintf("task.yaml file not found: %s", taskYamlPath)
			return ParsedLio2024Yaml{}, errwrap.Error(msg)
		}
		return ParsedLio2024Yaml{}, errwrap.AddTrace(err)
	}

	parsedYaml, err := ParseLio2024Yaml(taskYamlContent)
	if err != nil {
		return ParsedLio2024Yaml{}, errwrap.AddTrace(err)
	}

	return parsedYaml, nil
}

func base(parsedYaml ParsedLio2024Yaml) taskfs.Task {
	task := taskfs.Task{}
	task.FullName = taskfs.I18N[string]{"lv": parsedYaml.FullTaskName}
	task.ShortID = strings.ToLower(parsedYaml.TaskShortIDCode)

	task.Metadata.Difficulty = 0
	task.Metadata.ProblemTags = []string{}

	task.Origin.Olympiad = "LIO"
	task.Origin.OlyStage = ""
	task.Origin.Notes = taskfs.I18N[string]{"lv": ""}
	task.Origin.Authors = []string{}
	task.Origin.Year = ""

	task.Testing.TestingT = "simple"
	task.Testing.CpuLimMs = int(parsedYaml.CpuTimeLimitInSeconds * 1000)
	task.Testing.MemLimMiB = parsedYaml.MemoryLimitInMegabytes

	task.ReadMe = `## TODO list
	
- [ ] port statement from .typ to .md in statement dir
- [ ] subtask descriptions from .typ to task.toml
- [ ] example notes from .typ to .md in example dir
- [ ] specify the year & stage of the olympiad in task.toml
- [ ] add detailed note of the olympiad in task.toml
- [ ] optionally list the authors in task.toml
- [ ] should list the authors in origin in task.toml
- [ ] determine difficulty based on # of ACs in contest
`

	return task
}

func testing(task *taskfs.Task, dirPath string, parsedYaml ParsedLio2024Yaml) error {
	if parsedYaml.CheckerPathRelToYaml != nil {
		relativePath := *parsedYaml.CheckerPathRelToYaml
		checkerPath := filepath.Join(dirPath, relativePath)
		checkerBytes, err := os.ReadFile(checkerPath)
		if err != nil {
			return errwrap.AddTrace(err)
		}
		task.Testing.Checker = string(checkerBytes)
		task.Testing.TestingT = "checker"
	}

	if parsedYaml.InteractorPathRelToYaml != nil {
		relativePath := *parsedYaml.InteractorPathRelToYaml
		interactorPath := filepath.Join(dirPath, relativePath)
		interactorBytes, err := os.ReadFile(interactorPath)
		if err != nil {
			return errwrap.AddTrace(err)
		}
		task.Testing.Interactor = string(interactorBytes)
		task.Testing.TestingT = "interactor"
	}

	return nil
}

func tests(task *taskfs.Task, dirPath string, parsedYaml ParsedLio2024Yaml) error {
	testZipRelPath := parsedYaml.TestZipPathRelToYaml
	testZipAbsPath := filepath.Join(dirPath, testZipRelPath)
	tests, err := lio.ReadLioTestsFromZip(testZipAbsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			msg := fmt.Sprintf("test archive file not found: %s", testZipRelPath)
			return errwrap.Error(msg)
		}
		return errwrap.AddTrace(err)
	}

	sort.Slice(tests, func(i, j int) bool {
		if tests[i].TestGroup == tests[j].TestGroup {
			return tests[i].NoInTestGroup < tests[j].NoInTestGroup
		}
		return tests[i].TestGroup < tests[j].TestGroup
	})

	for _, t := range tests {
		if t.TestGroup == 0 {
			task.Statement.Examples = append(task.Statement.Examples, taskfs.Example{
				Input:  string(t.Input),
				Output: string(t.Answer),
				MdNote: map[string]string{},
			})
			continue
		}
		task.Testing.Tests = append(task.Testing.Tests, taskfs.Test{
			Input:  string(t.Input),
			Answer: string(t.Answer),
		})
	}

	return nil
}

func scoring(task *taskfs.Task, parsedYaml ParsedLio2024Yaml) error {
	// Set up subtasks based on parsedYaml.SubtaskPoints (skip first 0 entry)
	for i, points := range parsedYaml.SubtaskPoints {
		if i == 0 || points == 0 {
			continue // skip examples or 0-point subtasks
		}
		task.Statement.Subtasks = append(task.Statement.Subtasks, taskfs.Subtask{
			Desc:     map[string]string{"lv": fmt.Sprintf("Subtask %d", i)},
			Points:   points,
			VisInput: false,
		})
	}

	for _, g := range parsedYaml.TestGroups {
		if g.GroupID == 0 {
			continue // examples
		}
		task.Scoring.Groups = append(task.Scoring.Groups, taskfs.TestGroup{
			Points:  g.Points,
			Range:   [2]int{g.GroupID, g.GroupID},
			Public:  g.Public,
			Subtask: g.Subtask,
		})
	}

	// Set up scoring configuration
	task.Scoring.ScoringT = "min-groups"
	totalPoints := 0
	for _, group := range task.Scoring.Groups {
		totalPoints += group.Points
	}
	task.Scoring.TotalP = totalPoints

	// verify that subtask points sum up to 100
	subtaskTotalPoints := 0
	for _, subtask := range task.Statement.Subtasks {
		subtaskTotalPoints += subtask.Points
	}
	if subtaskTotalPoints != 100 {
		msg := fmt.Sprintf("subtask points do not sum up to 100: got %d", subtaskTotalPoints)
		return errwrap.Error(msg)
	}

	return nil
}

// statement must be after testing type is determined
func statement(task *taskfs.Task, dirPath string, testingType string) error {
	if testingType != "interactor" {
		task.Statement.Stories = taskfs.I18N[taskfs.StoryMd]{
			"lv": taskfs.StoryMd{
				Story:   "TODO",
				Input:   "TODO",
				Output:  "TODO",
				Notes:   "",
				Scoring: "",
				Example: "",
				Talk:    "",
			},
		}
	} else {
		task.Statement.Stories = taskfs.I18N[taskfs.StoryMd]{
			"lv": taskfs.StoryMd{
				Story:   "TODO",
				Input:   "",
				Output:  "",
				Notes:   "",
				Scoring: "",
				Talk:    "TODO",
				Example: "TODO",
			},
		}
	}

	// Find and add images from teksts directory
	tekstsDir := filepath.Join(dirPath, "teksts")
	if _, err := os.Stat(tekstsDir); !errors.Is(err, fs.ErrNotExist) {
		err := filepath.Walk(tekstsDir, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return errwrap.AddTrace(err)
			}
			if info.IsDir() {
				return nil
			}

			// Check for image files (png, jpg, jpeg)
			if strings.HasSuffix(strings.ToLower(path), ".png") ||
				strings.HasSuffix(strings.ToLower(path), ".jpg") ||
				strings.HasSuffix(strings.ToLower(path), ".jpeg") {
				content, err := os.ReadFile(path)
				if err != nil {
					return errwrap.AddTrace(err)
				}

				task.Statement.Images = append(task.Statement.Images, taskfs.Image{
					Fname:   filepath.Base(path),
					Content: content,
				})
			}
			return nil
		})
		if err != nil {
			return errwrap.AddTrace(err)
		}
	}

	// Set VisInput for first subtask based on .typ file content
	if len(task.Statement.Subtasks) > 0 {
		hasOutputFalse, err := checkTypFileForOutputFalse(dirPath)
		if err != nil {
			return errwrap.AddTrace(err)
		}
		task.Statement.Subtasks[0].VisInput = hasOutputFalse
	}

	return nil
}

func checkTypFileForOutputFalse(dirPath string) (bool, error) {
	tekstsDir := filepath.Join(dirPath, "teksts")

	var typFilePath string
	err := filepath.Walk(tekstsDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".typ") {
			typFilePath = path
			return filepath.SkipDir // Stop after finding first .typ file
		}
		return nil
	})

	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, errwrap.AddTrace(err)
	}

	if typFilePath == "" {
		return false, nil // No .typ file found
	}

	content, err := os.ReadFile(typFilePath)
	if err != nil {
		return false, errwrap.AddTrace(err)
	}

	// Check for "output: false" that is not commented out with //
	// This regex matches lines that contain "output: false" but don't start with // (ignoring whitespace)
	pattern := `(?m)^[^/]*output:\s*false`
	matched, err := regexp.MatchString(pattern, string(content))
	if err != nil {
		return false, errwrap.AddTrace(err)
	}

	return matched, nil
}

func solutions(task *taskfs.Task, dirPath string) error {
	solutionsDirPath := filepath.Join(dirPath, "risin")
	if _, err := os.Stat(solutionsDirPath); errors.Is(err, fs.ErrNotExist) {
		return nil // no solutions directory is ok
	}

	err := filepath.Walk(solutionsDirPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return errwrap.AddTrace(err)
		}
		if info.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(solutionsDirPath, path)
		if err != nil {
			return errwrap.AddTrace(err)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return errwrap.AddTrace(err)
		}

		filename := filepath.Base(relativePath)
		subtasks := []int{} // Initialize as empty slice, not nil

		// If solution name contains "ok", it should pass all subtasks
		if strings.Contains(strings.ToLower(filename), "ok") {
			// Create slice with all subtask numbers (1-based indexing)
			for i := range task.Statement.Subtasks {
				subtasks = append(subtasks, i+1)
			}
		}

		task.Solutions = append(task.Solutions, taskfs.Solution{
			Fname:    filename,
			Content:  string(content),
			Subtasks: subtasks,
		})

		return nil
	})

	if err != nil {
		return errwrap.AddTrace(err)
	}
	return nil
}

func archive(task *taskfs.Task, dirPath string, parsedYaml ParsedLio2024Yaml) error {
	testZipAbsPath := filepath.Join(dirPath, parsedYaml.TestZipPathRelToYaml)

	excludePrefixFromArchive := []string{
		testZipAbsPath,
	}

	err := filepath.Walk(dirPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return errwrap.AddTrace(err)
		}
		if info.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return errwrap.AddTrace(err)
		}
		relativePath = "./" + relativePath

		for _, prefix := range excludePrefixFromArchive {
			prefixAbs, err := filepath.Abs(prefix)
			if err != nil {
				return errwrap.AddTrace(err)
			}
			pathAbs, err := filepath.Abs(path)
			if err != nil {
				return errwrap.AddTrace(err)
			}
			if pathAbs == prefixAbs {
				return nil
			}
			if strings.HasPrefix(pathAbs, prefixAbs+string(filepath.Separator)) {
				return nil
			}
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return errwrap.AddTrace(err)
		}

		task.Archive.Files = append(task.Archive.Files, taskfs.ArchiveFile{
			RelPath: relativePath,
			Content: content,
		})
		return nil
	})

	if err != nil {
		return errwrap.AddTrace(err)
	}
	return nil
}
