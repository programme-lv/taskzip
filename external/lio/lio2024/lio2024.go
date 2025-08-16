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

	"github.com/programme-lv/task-zip/common/etrace"
	"github.com/programme-lv/task-zip/common/fn"
	"github.com/programme-lv/task-zip/external/lio"
	"github.com/programme-lv/task-zip/taskfs"
)

var ErrTestArchiveNotFound = etrace.NewError("test archive file not found")

func ParseLio2024TaskDir(dirPath string) (taskfs.Task, error) {
	parsedYaml, err := readYaml(dirPath)
	if err != nil {
		return taskfs.Task{}, etrace.Trace(err)
	}

	task := base(parsedYaml)

	var errs []error

	if err := testing(&task, dirPath, parsedYaml); err != nil {
		errs = append(errs, etrace.Trace(err))
	}

	testZipRelPath := parsedYaml.TestZipPathRelToYaml
	testZipAbsPath := filepath.Join(dirPath, testZipRelPath)
	lioTests, err := lio.ReadLioTestsFromZip(testZipAbsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return taskfs.Task{}, etrace.Trace(ErrTestArchiveNotFound)
		}
		return taskfs.Task{}, etrace.Trace(err)
	}

	if err := tests(&task, lioTests, dirPath, parsedYaml); err != nil {
		errs = append(errs, etrace.Trace(err))
	}

	if err := scoring(&task, lioTests, parsedYaml); err != nil {
		errs = append(errs, etrace.Trace(err))
	}

	if err := statement(&task, dirPath, task.Testing.TestingT); err != nil {
		errs = append(errs, etrace.Trace(err))
	}

	if err := solutions(&task, dirPath); err != nil {
		errs = append(errs, etrace.Trace(err))
	}

	if err := archive(&task, dirPath, parsedYaml); err != nil {
		errs = append(errs, etrace.Trace(err))
	}

	return task, errors.Join(errs...)
}

func readYaml(dirPath string) (ParsedLio2024Yaml, error) {
	taskYamlPath := filepath.Join(dirPath, "task.yaml")

	taskYamlContent, err := os.ReadFile(taskYamlPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			msg := fmt.Sprintf("task.yaml file not found: %s", taskYamlPath)
			return ParsedLio2024Yaml{}, etrace.NewError(msg)
		}
		return ParsedLio2024Yaml{}, etrace.Trace(err)
	}

	parsedYaml, err := ParseLio2024Yaml(taskYamlContent)
	if err != nil {
		return ParsedLio2024Yaml{}, etrace.Trace(err)
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

- [ ] specify the year & stage of the olympiad in task.toml
- [ ] paste descriptive note of the olympiad in task.toml
- [ ] port statement from .typ to .md in statement dir
- [ ] subtask descriptions from .typ to task.toml
- [ ] example notes from .typ to .md in example dir
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
			return etrace.Trace(err)
		}
		task.Testing.Checker = string(checkerBytes)
		task.Testing.TestingT = "checker"
	}

	if parsedYaml.InteractorPathRelToYaml != nil {
		relativePath := *parsedYaml.InteractorPathRelToYaml
		interactorPath := filepath.Join(dirPath, relativePath)
		interactorBytes, err := os.ReadFile(interactorPath)
		if err != nil {
			return etrace.Trace(err)
		}
		task.Testing.Interactor = string(interactorBytes)
		task.Testing.TestingT = "interactor"
	}

	return nil
}

func tests(task *taskfs.Task, lioTests []lio.LioTest, dirPath string, parsedYaml ParsedLio2024Yaml) error {
	sort.Slice(lioTests, func(i, j int) bool {
		if lioTests[i].TestGroup == lioTests[j].TestGroup {
			return lioTests[i].NoInTestGroup < lioTests[j].NoInTestGroup
		}
		return lioTests[i].TestGroup < lioTests[j].TestGroup
	})

	for _, t := range lioTests {
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

var ErrSubtaskPointsSumNot100 = etrace.NewError("subtask points do not sum up to 100")
var ErrNoTestsForGroup = etrace.NewError("no tests for group")
var ErrGroupTestIdxNotConsecutive = etrace.NewError("test idx in group not consecutive")

func scoring(task *taskfs.Task, lioTests []lio.LioTest, parsedYaml ParsedLio2024Yaml) error {
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

	// build linear test index over non-example tests (1..N)
	ordered := make([]lio.LioTest, len(lioTests))
	copy(ordered, lioTests)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].TestGroup == ordered[j].TestGroup {
			return ordered[i].NoInTestGroup < ordered[j].NoInTestGroup
		}
		return ordered[i].TestGroup < ordered[j].TestGroup
	})
	nonExamples := []lio.LioTest{}
	for _, t := range ordered {
		if t.TestGroup != 0 {
			nonExamples = append(nonExamples, t)
		}
	}

	for _, g := range parsedYaml.TestGroups {
		if g.GroupID == 0 {
			continue // examples
		}
		idxs := []int{}
		for i, t := range nonExamples {
			if t.TestGroup == g.GroupID {
				idxs = append(idxs, i+1)
			}
		}
		if len(idxs) == 0 {
			return etrace.Trace(ErrNoTestsForGroup)
		}
		if !fn.AreConsecutive(idxs) {
			return etrace.Trace(ErrGroupTestIdxNotConsecutive)
		}
		rng := [2]int{fn.MinInt(idxs), fn.MaxInt(idxs)}

		task.Scoring.Groups = append(task.Scoring.Groups, taskfs.TestGroup{
			Points:  g.Points,
			Range:   rng,
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
		msg := fmt.Sprintf("got %d", subtaskTotalPoints)
		return etrace.Wrap(msg, ErrSubtaskPointsSumNot100)
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
				return etrace.Trace(err)
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
					return etrace.Trace(err)
				}

				task.Statement.Images = append(task.Statement.Images, taskfs.Image{
					Fname:   filepath.Base(path),
					Content: content,
				})
			}
			return nil
		})
		if err != nil {
			return etrace.Trace(err)
		}
	}

	// Set VisInput for first subtask based on .typ file content
	if len(task.Statement.Subtasks) > 0 {
		hasOutputFalse, err := checkTypFileForOutputFalse(dirPath)
		if err != nil {
			return etrace.Trace(err)
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
		return false, etrace.Trace(err)
	}

	if typFilePath == "" {
		return false, nil // No .typ file found
	}

	content, err := os.ReadFile(typFilePath)
	if err != nil {
		return false, etrace.Trace(err)
	}

	// Check for "output: false" that is not commented out with //
	// This regex matches lines that contain "output: false" but don't start with // (ignoring whitespace)
	pattern := `(?m)^[^/]*output:\s*false`
	matched, err := regexp.MatchString(pattern, string(content))
	if err != nil {
		return false, etrace.Trace(err)
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
			return etrace.Trace(err)
		}
		if info.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(solutionsDirPath, path)
		if err != nil {
			return etrace.Trace(err)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return etrace.Trace(err)
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
		return etrace.Trace(err)
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
			return etrace.Trace(err)
		}
		if info.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return etrace.Trace(err)
		}
		relativePath = "./" + relativePath

		for _, prefix := range excludePrefixFromArchive {
			prefixAbs, err := filepath.Abs(prefix)
			if err != nil {
				return etrace.Trace(err)
			}
			pathAbs, err := filepath.Abs(path)
			if err != nil {
				return etrace.Trace(err)
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
			return etrace.Trace(err)
		}

		task.Archive.Files = append(task.Archive.Files, taskfs.ArchiveFile{
			RelPath: relativePath,
			Content: content,
		})
		return nil
	})

	if err != nil {
		return etrace.Trace(err)
	}
	return nil
}
