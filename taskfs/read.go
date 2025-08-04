package taskfs

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strconv"
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
		msg := "invalid testing component"
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

func filter[T any](ss []T, test func(T) bool) (ret []T) {
	for _, s := range ss {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}

func (dir TaskDirReader) Stories() (i18n[StoryMd], error) {
	stories := make(i18n[StoryMd])
	files, err := dir.ListDir("statement")
	if err != nil {
		msg := "list statement dir"
		return i18n[StoryMd]{}, wrap(msg, err)
	}
	mdFiles := filter(files, func(file string) bool {
		return strings.HasSuffix(file, ".md")
	})
	for _, file := range mdFiles {
		content, err := dir.ReadFile(filepath.Join("statement", file))
		if err != nil {
			msg := fmt.Sprintf("read story %s", file)
			return i18n[StoryMd]{}, wrap(msg, err)
		}
		lang := strings.TrimSuffix(file, ".md")
		story, err := ParseMdStory(string(content), lang)
		if err != nil {
			msg := fmt.Sprintf("parse story %s", file)
			return i18n[StoryMd]{}, wrap(msg, err)
		}
		stories[lang] = story
	}
	return stories, nil
}

type MdStorySection func(*StoryMd) *string

// returns pointer to the corresponding field of the story
var (
	StorySection   MdStorySection = func(s *StoryMd) *string { return &s.Story }
	InputSection   MdStorySection = func(s *StoryMd) *string { return &s.Input }
	OutputSection  MdStorySection = func(s *StoryMd) *string { return &s.Output }
	NoteSection    MdStorySection = func(s *StoryMd) *string { return &s.Notes }
	ScoringSection MdStorySection = func(s *StoryMd) *string { return &s.Scoring }
	ExampleSection MdStorySection = func(s *StoryMd) *string { return &s.Example }
	TalkSection    MdStorySection = func(s *StoryMd) *string { return &s.Talk }
)

var mdStorySectionI18n = i18n[map[string]MdStorySection]{
	"en": {
		"Story":       StorySection,
		"Input":       InputSection,
		"Output":      OutputSection,
		"Notes":       NoteSection,
		"Scoring":     ScoringSection,
		"Example":     ExampleSection,
		"Interaction": TalkSection,
	},
	"lv": {
		"Stāsts":       StorySection,
		"Ievaddati":    InputSection,
		"Izvaddati":    OutputSection,
		"Piezīmes":     NoteSection,
		"Vērtēšana":    ScoringSection,
		"Piemērs":      ExampleSection,
		"Komunikācija": TalkSection,
	},
}

func trimLinesInText(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
		lines[i] = strings.ReplaceAll(lines[i], "\r", "")
	}
	return strings.Join(lines, "\n")
}

func mapSliceElemsToNew[S any, T any](ss []S, f func(S) T) []T {
	res := make([]T, len(ss))
	for i, s := range ss {
		res[i] = f(s)
	}
	return res
}

type Pair[F any, S any] struct {
	First  F
	Second S
}

func SplitByDividers(content string, dividers map[string]MdStorySection) ([]Pair[MdStorySection, string], error) {
	indices := []Pair[int, MdStorySection]{}
	for divider, section := range dividers {
		dividerStr := fmt.Sprintf("%s\n-", divider)
		fst := strings.Index(content, dividerStr)
		lst := strings.LastIndex(content, dividerStr)
		if fst == -1 && lst == -1 {
			continue
		}
		if fst != lst {
			msg := fmt.Sprintf("divider %s occurs multiple times", divider)
			return nil, wrap(msg)
		}
		indices = append(indices, Pair[int, MdStorySection]{fst, section})
		content = strings.ReplaceAll(content, dividerStr, "#")
	}
	if len(indices) == 0 {
		return []Pair[MdStorySection, string]{}, nil
	}
	for strings.Contains(content, "#-") {
		content = strings.ReplaceAll(content, "#-", "#")
	}
	sort.Slice(indices, func(i, j int) bool {
		return indices[i].First < indices[j].First
	})
	contentSlice := strings.Split(content, "#")
	contentSlice = filter(contentSlice, func(s string) bool {
		return s != ""
	})
	contentSlice = mapSliceElemsToNew(contentSlice, func(s string) string {
		return strings.TrimSpace(s)
	})
	if len(indices) != len(contentSlice) {
		msg := fmt.Sprintf("dividers (%d) != content segments (%d)", len(indices), len(contentSlice))
		return nil, wrap(msg)
	}
	parsed := make([]Pair[MdStorySection, string], len(indices))
	for i, index := range indices {
		parsed[i] = Pair[MdStorySection, string]{index.Second, contentSlice[i]}
	}
	return parsed, nil
}

func ParseMdStory(content string, lang string) (StoryMd, error) {
	content = trimLinesInText(content)
	res := StoryMd{}
	foundLang := false
	for _, translation := range mdStorySectionI18n {
		parsed, err := SplitByDividers(content, translation)
		if err != nil {
			msg := "split by dividers"
			return StoryMd{}, wrap(msg, err)
		}
		if len(parsed) > 0 {
			if foundLang {
				msg := "story section dividers in multiple langs"
				return StoryMd{}, wrap(msg)
			}
			foundLang = true
		}
		for _, pair := range parsed {
			*pair.First(&res) = pair.Second
		}
	}

	return res, nil
}

func (dir TaskDirReader) Statement() (Statement, error) {
	taskToml, err := dir.Toml()
	if err != nil {
		msg := "read task.toml"
		return Statement{}, wrap(msg, err)
	}

	subtasks := make([]Subtask, len(taskToml.Subtasks))
	for i, subtaskToml := range taskToml.Subtasks {
		subtasks[i] = Subtask{
			Desc:     subtaskToml.Description,
			Points:   subtaskToml.Points,
			VisInput: false,
		}
	}

	examples, err := dir.Examples()
	if err != nil {
		msg := "read examples"
		return Statement{}, wrap(msg, err)
	}

	stories, err := dir.Stories()
	if err != nil {
		msg := "read stories"
		return Statement{}, wrap(msg, err)
	}

	statement := Statement{
		Stories:  stories,
		Subtasks: subtasks,
		Examples: examples,
	}
	return statement, nil
}

func (dir TaskDirReader) Example(index int) (Example, error) {
	notePath := fmt.Sprintf("examples/%03d.md", index)
	noteContent, err := dir.ReadFile(notePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		msg := fmt.Sprintf("read example note %d", index)
		return Example{}, wrap(msg, err)
	}
	note := parseMultilingualMd(string(noteContent))
	inPath := fmt.Sprintf("examples/%03di.txt", index)
	inContent, err := dir.ReadFile(inPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		msg := fmt.Sprintf("read example input %d", index)
		return Example{}, wrap(msg, err)
	}
	outPath := fmt.Sprintf("examples/%03do.txt", index)
	outContent, err := dir.ReadFile(outPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		msg := fmt.Sprintf("read example output %d", index)
		return Example{}, wrap(msg, err)
	}
	return Example{
		Input:  string(inContent),
		Output: string(outContent),
		MdNote: note,
	}, nil
}

func ensureConsecutiveIdsFrom1(idxMap map[int]bool) (int, error) {
	minIdx := math.MaxInt32
	maxIdx := 0
	for idx := range idxMap {
		if idx > maxIdx {
			maxIdx = idx
		}
		if idx < minIdx {
			minIdx = idx
		}
	}

	if minIdx != 1 {
		return 0, fmt.Errorf("first entry must have index 1")
	}

	for i := minIdx; i <= maxIdx; i++ {
		if !idxMap[i] {
			return 0, fmt.Errorf("entry %d missing", i)
		}
	}
	return maxIdx, nil
}

func (dir TaskDirReader) Examples() ([]Example, error) {
	files, err := dir.ListDir("examples")
	if err != nil {
		msg := "list examples dir"
		return []Example{}, wrap(msg, err)
	}

	idxMap := make(map[int]bool)
	for _, file := range files {
		index, err := strconv.Atoi(file[:3])
		if err != nil {
			msg := fmt.Sprintf("parse example index %s", file)
			return []Example{}, wrap(msg, err)
		}
		idxMap[index] = true
	}

	maxIdx, err := ensureConsecutiveIdsFrom1(idxMap)
	if err != nil {
		msg := "ensure consecutive ids"
		return []Example{}, wrap(msg, err)
	}

	examples := make([]Example, maxIdx)
	for i := 1; i <= maxIdx; i++ {
		example, err := dir.Example(i)
		if err != nil {
			msg := fmt.Sprintf("read example %d", i)
			return []Example{}, wrap(msg, err)
		}
		examples[i-1] = example
	}
	return examples, nil
}

func parseMultilingualMd(content string) i18n[string] {
	result := make(i18n[string])
	content = strings.TrimSpace(content)

	if content == "" {
		return result
	}

	lines := strings.Split(content, "\n")
	var currentLang string
	var currentContent []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "---" {
			// Start collecting content for the current language
			currentContent = []string{}
		} else if currentLang == "" && line != "" && !strings.Contains(line, " ") {
			// This is a language code at the beginning
			currentLang = line
		} else if currentLang != "" && len(currentContent) >= 0 && line != "" && !strings.Contains(line, " ") && line != currentLang {
			// This looks like a new language code, save previous and start new
			if len(currentContent) > 0 {
				result[currentLang] = strings.TrimSpace(strings.Join(currentContent, "\n"))
			}
			currentLang = line
			currentContent = []string{}
		} else if currentLang != "" && line != "" {
			// Content line for current language
			currentContent = append(currentContent, line)
		}
	}

	// Save the last section
	if currentLang != "" && len(currentContent) > 0 {
		result[currentLang] = strings.TrimSpace(strings.Join(currentContent, "\n"))
	}

	return result
}

func (dir TaskDirReader) TestGroups() ([]TestGroup, error) {
	testgroupstTxt, err := dir.ReadFile("testgroups.txt")
	if err != nil {
		msg := "read testgroups.txt"
		return []TestGroup{}, wrap(msg, err)
	}
	testgroups := []TestGroup{}
	lines := strings.Split(string(testgroupstTxt), "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		tg, err := parseTestGroupLine(i+1, line)
		if err != nil {
			msg := fmt.Sprintf("parse tg line %d", i+1)
			return []TestGroup{}, wrap(msg, err)
		}
		testgroups = append(testgroups, tg)
	}
	return testgroups, nil
}

func parseTestGroupLine(idx int, line string) (tg TestGroup, err error) {
	// 01: 001-005 3p (1)
	// 02: 006-010 3p (1) *
	// 03: 011-013 4p (2)
	tg.Public = strings.Contains(line, "*")
	line = strings.TrimSpace(strings.ReplaceAll(line, "*", ""))
	for strings.Contains(line, "  ") {
		line = strings.ReplaceAll(line, "  ", " ")
	}
	re := regexp.MustCompile(`^(\d+): (\d+)-(\d+) (\d+)p \((\d+)\)$`)
	matches := re.FindAllStringSubmatch(line, -1)
	if len(matches) != 1 || len(matches[0]) < 1 {
		msg := "invalid structure of tg line"
		return TestGroup{}, wrap(msg)
	}
	matches[0] = matches[0][1:]
	ints := make([]int, len(matches[0]))
	for j, match := range matches[0] {
		ints[j], err = strconv.Atoi(match)
		if err != nil {
			msg := fmt.Sprintf("converting %s to int", match)
			return TestGroup{}, wrap(msg, err)
		}
	}
	if len(ints) != 4 && len(ints) != 5 {
		msg := "invalid no. of parts"
		return TestGroup{}, wrap(msg)
	}
	if ints[0] != idx {
		msg := fmt.Sprintf("tg id %d does not match idx %d", ints[0], idx)
		return TestGroup{}, wrap(msg)
	}
	tg.Range = [2]int{ints[1], ints[2]}
	tg.Points = ints[3]
	if len(ints) == 5 {
		tg.Subtask = ints[4]
	}
	return tg, nil
}

func (dir TaskDirReader) Scoring(noOfTests int) (Scoring, error) {
	taskToml, err := dir.Toml()
	if err != nil {
		msg := "read task.toml"
		return Scoring{}, wrap(msg, err)
	}
	tgs, err := dir.TestGroups()
	if err != nil {
		msg := "read testgroups.txt"
		return Scoring{}, wrap(msg, err)
	}
	scoring := Scoring{
		ScoringT: taskToml.Scoring.Type,
		TotalP:   taskToml.Scoring.Total,
		Groups:   tgs,
	}
	noOfSubtasks := len(taskToml.Subtasks)
	if err := scoring.Validate(noOfSubtasks, noOfTests); err != nil {
		msg := "validate scoring"
		return Scoring{}, wrap(msg, err)
	}
	return scoring, nil
}

func (dir TaskDirReader) Task() (task Task, err error) {
	var taskToml TaskToml
	if taskToml, err = dir.Toml(); err != nil {
		return
	}
	task.ShortID = taskToml.Id
	task.FullName = taskToml.Name

	if task.Testing, err = dir.Testing(); err != nil {
		return
	}
	if task.ReadMe, err = dir.Readme(); err != nil {
		return
	}
	if task.Origin, err = dir.Origin(); err != nil {
		return
	}
	if task.Metadata, err = dir.Metadata(); err != nil {
		return
	}
	if task.Solutions, err = dir.Solutions(); err != nil {
		return
	}
	if task.Statement, err = dir.Statement(); err != nil {
		return
	}
	if task.Scoring, err = dir.Scoring(len(task.Testing.Tests)); err != nil {
		return
	}
	return
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
