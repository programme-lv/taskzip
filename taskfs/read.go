package taskfs

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/programme-lv/task-zip/common/etrace"
	"github.com/programme-lv/task-zip/common/fn"
)

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
		return Task{}, etrace.Trace(err)
	}

	task, err := dir.Task()
	if err != nil {
		return Task{}, etrace.Trace(err)
	}

	if conf.checkAllFilesRead && !dir.AllFilesWereRead() {
		for _, path := range dir.allPaths {
			if !dir.readPaths[path] {
				msg := fmt.Sprintf("file %s never read when parsing task", path)
				return task, etrace.NewError(msg)
			}
		}
		msg := "no unread files found but dir.AllFilesWereRead() is false"
		return task, etrace.Wrap(msg, nil)
	}

	err = task.Validate()
	if err != nil {
		return Task{}, etrace.Trace(err)
	}

	return task, nil
}

type TaskDirReader struct {
	dirAbsPath string
	readPaths  map[string]bool // map of relative paths that have been read
	allPaths   []string        // list of all relative paths in the task directory
}

func NewTaskDir(dirAbsPath string) (TaskDirReader, error) {
	dirAbsPath, err := filepath.Abs(dirAbsPath)
	if err != nil {
		msg := "get absolute path"
		return TaskDirReader{}, etrace.Wrap(msg, err)
	}
	dir := TaskDirReader{
		dirAbsPath: dirAbsPath,
		readPaths:  make(map[string]bool),
	}
	err = dir.readAllPathsInDir()
	if err != nil {
		msg := "read all paths in dir"
		return TaskDirReader{}, etrace.Wrap(msg, err)
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
				return etrace.Trace(err)
			}

			totalSize += info.Size()
			fileCount++

			if totalSize > 512*1024*1024 { // 512 MB
				msg := "directory exceeds maximum size of 512 MB"
				return etrace.NewError(msg)
			}
			if fileCount > 10000 {
				msg := "directory contains more than 10000 files"
				return etrace.NewError(msg)
			}

			relPath, err := filepath.Rel(dir.dirAbsPath, path)
			if err != nil {
				msg := "get relative path"
				return etrace.Wrap(msg, err)
			}
			dir.allPaths = append(dir.allPaths, relPath)
		}
		return nil
	})
	if err != nil {
		return etrace.Wrap("walk directory", err)
	}
	return nil
}

var ErrExpectedFileMissing = etrace.NewError("expected file is missing")

func (dir TaskDirReader) ReadFile(relPath string) ([]byte, error) {
	joined := filepath.Join(dir.dirAbsPath, relPath)
	clean := filepath.Clean(joined)

	// ensure the path is within the task directory
	filePathRel, err := filepath.Rel(dir.dirAbsPath, clean)
	if err != nil {
		msg := "get relative path"
		return nil, etrace.Wrap(msg, err)
	}
	if strings.Contains(filePathRel, "..") {
		msg := fmt.Sprintf("path %s attempts to leave task directory", relPath)
		return nil, etrace.Trace(errors.New(msg))
	}

	bytes, err := os.ReadFile(clean)
	if err != nil {
		return nil, etrace.Trace(ErrExpectedFileMissing.WithCause(err))
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
			paths = append(paths, rel)
		}
	}
	return paths, nil
}

func (dir TaskDirReader) Toml() (TaskToml, error) {
	content, err := dir.ReadFile("task.toml")
	if err != nil {
		return TaskToml{}, etrace.Trace(err)
	}
	taskToml := TaskToml{}
	d := toml.NewDecoder(bytes.NewReader(content))
	d.DisallowUnknownFields()
	err = d.Decode(&taskToml)
	if err != nil {
		var details *toml.StrictMissingError
		if errors.As(err, &details) {
			msg := details.String()
			return TaskToml{}, etrace.Wrap(msg, err)
		}
		return TaskToml{}, etrace.Trace(err)
	}
	return taskToml, nil
}

var ErrCheckerMissing = etrace.NewError("checker.cpp is missing")

func (dir TaskDirReader) Checker() (string, error) {
	path := "checker.cpp"
	content, err := dir.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", etrace.Trace(ErrCheckerMissing.WithCause(err))
	}
	if err != nil {
		return "", etrace.Trace(err)
	}
	return string(content), nil
}

func (dir TaskDirReader) Interactor() (string, error) {
	path := "interactor.cpp"
	content, err := dir.ReadFile(path)
	if err != nil {
		return "", etrace.Trace(err)
	}
	return string(content), nil
}

func (dir TaskDirReader) Tests() ([]Test, error) {
	testDirPath := "tests"
	testFilePaths, err := dir.ListDir(testDirPath)
	if err != nil {
		msg := "list tests"
		return nil, etrace.Wrap(msg, err)
	}
	if len(testFilePaths)%2 != 0 {
		msg := "number of tests must be even"
		return nil, etrace.NewError(msg)
	}
	if len(testFilePaths) > 999*2 {
		msg := "max 999 tests allowed (2 files per test)"
		return nil, etrace.NewError(msg)
	}
	slices.Sort(testFilePaths)
	tests := make([]Test, len(testFilePaths)/2)
	for i := 0; i < len(testFilePaths); i += 2 {
		expInFname := fmt.Sprintf("%03di.txt", (i/2)+1)
		expOutFname := fmt.Sprintf("%03do.txt", (i/2)+1)
		if testFilePaths[i] != expInFname {
			msg := fmt.Sprintf("input test file path is incorrect: %s != %s", testFilePaths[i], expInFname)
			return nil, etrace.NewError(msg)
		}
		if testFilePaths[i+1] != expOutFname {
			msg := fmt.Sprintf("output test file path is incorrect: %s != %s", testFilePaths[i+1], expOutFname)
			return nil, etrace.NewError(msg)
		}
		inPath := filepath.Join(testDirPath, testFilePaths[i])
		input, err := dir.ReadFile(inPath)
		if err != nil {
			return nil, etrace.Trace(err)
		}
		outPath := filepath.Join(testDirPath, testFilePaths[i+1])
		output, err := dir.ReadFile(outPath)
		if err != nil {
			return nil, etrace.Trace(err)
		}
		tests[i/2] = Test{Input: string(input), Answer: string(output)}
	}
	return tests, nil
}

var ErrCheckerEmpty = etrace.NewError("checker.cpp is empty")

func (dir TaskDirReader) Testing() (Testing, error) {
	taskToml, err := dir.Toml()
	if err != nil {
		return Testing{}, etrace.Trace(err)
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
		msg := "testing type in task.toml is checker"
		if err != nil {
			return Testing{}, etrace.Wrap(msg, err)
		}
		if checker == "" {
			return t, etrace.Wrap(msg, ErrCheckerEmpty)
		}
		t.Checker = checker
	}
	if taskToml.Testing.Type == "interactor" {
		interactor, err := dir.Interactor()
		if err != nil {
			return Testing{}, etrace.Trace(err)
		}
		if interactor == "" {
			msg := "interactor.cpp is empty"
			return Testing{}, etrace.NewError(msg)
		}
		t.Interactor = interactor
	}
	tests, err := dir.Tests()
	if err != nil {
		return Testing{}, etrace.Trace(err)
	}
	t.Tests = tests
	err = t.Validate()
	if err != nil {
		return Testing{}, etrace.Trace(err)
	}
	return t, nil
}

func (dir TaskDirReader) Readme() (string, error) {
	content, err := dir.ReadFile("readme.md")
	if err != nil {
		return "", etrace.Trace(err)
	}
	return string(content), nil
}

func (dir TaskDirReader) Origin() (Origin, error) {
	taskToml, err := dir.Toml()
	if err != nil {
		return Origin{}, etrace.Trace(err)
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
		return Origin{}, etrace.Trace(err)
	}
	return o, nil
}

func (dir TaskDirReader) Metadata() (Metadata, error) {
	taskToml, err := dir.Toml()
	if err != nil {
		return Metadata{}, etrace.Trace(err)
	}

	m := Metadata{
		ProblemTags: taskToml.Metadata.Tags,
		Difficulty:  taskToml.Metadata.Difficulty,
	}
	err = m.Validate()
	if err != nil {
		return Metadata{}, etrace.Trace(err)
	}
	return m, nil
}

func (dir TaskDirReader) Solutions() ([]Solution, error) {
	taskToml, err := dir.Toml()
	if err != nil {
		return []Solution{}, etrace.Trace(err)
	}

	solutions := make([]Solution, len(taskToml.Solutions))
	for i, sol := range taskToml.Solutions {
		solPath := filepath.Join("solutions", sol.Fname)
		content, err := dir.ReadFile(solPath)
		if err != nil {
			return []Solution{}, etrace.Trace(err)
		}

		solutions[i] = Solution{
			Fname:    sol.Fname,
			Subtasks: sol.Subtasks,
			Content:  string(content),
		}
	}

	return solutions, nil
}

func (dir TaskDirReader) Stories() (I18N[StoryMd], error) {
	stories := make(I18N[StoryMd])
	files, err := dir.ListDir("statement")
	if err != nil {
		msg := "list statement dir"
		return I18N[StoryMd]{}, etrace.Wrap(msg, err)
	}
	mdFiles := fn.Filter(files, func(file string) bool {
		return strings.HasSuffix(file, ".md")
	})
	for _, file := range mdFiles {
		content, err := dir.ReadFile(filepath.Join("statement", file))
		if err != nil {
			msg := fmt.Sprintf("read story %s", file)
			return I18N[StoryMd]{}, etrace.Wrap(msg, err)
		}
		lang := strings.TrimSuffix(file, ".md")
		story, err := ParseMdStory(string(content), lang)
		if err != nil {
			msg := fmt.Sprintf("parse story %s", file)
			return I18N[StoryMd]{}, etrace.Wrap(msg, err)
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

var mdStorySectionI18n = I18N[map[string]MdStorySection]{
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
			return nil, etrace.NewError(msg)
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
	contentSlice = fn.Filter(contentSlice, func(s string) bool {
		return s != ""
	})
	contentSlice = fn.Map(contentSlice, func(s string) string {
		return strings.TrimSpace(s)
	})
	if len(indices) != len(contentSlice) {
		msg := fmt.Sprintf("dividers (%d) != content segments (%d)", len(indices), len(contentSlice))
		return nil, etrace.NewError(msg)
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
			return StoryMd{}, etrace.Wrap(msg, err)
		}
		if len(parsed) > 0 {
			if foundLang {
				msg := "story section dividers in multiple langs"
				return StoryMd{}, etrace.NewError(msg)
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
		return Statement{}, etrace.Wrap(msg, err)
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
		return Statement{}, etrace.Wrap(msg, err)
	}

	stories, err := dir.Stories()
	if err != nil {
		msg := "read stories"
		return Statement{}, etrace.Wrap(msg, err)
	}

	imagePaths, err := dir.ListDir("statement")
	if err != nil {
		msg := "list statement dir"
		return Statement{}, etrace.Wrap(msg, err)
	}
	isImage := func(path string) bool {
		exts := []string{".png", ".jpg", ".jpeg"}
		for _, ext := range exts {
			if strings.HasSuffix(path, ext) {
				return true
			}
		}
		return false
	}
	imagePaths = fn.Filter(imagePaths, isImage)
	images := make([]Image, len(imagePaths))
	for i, path := range imagePaths {
		content, err := dir.ReadFile(filepath.Join("statement", path))
		if err != nil {
			msg := fmt.Sprintf("read image %s", path)
			return Statement{}, etrace.Wrap(msg, err)
		}
		images[i] = Image{
			Fname:   path,
			Content: content,
		}
	}

	statement := Statement{
		Stories:  stories,
		Subtasks: subtasks,
		Examples: examples,
		Images:   images,
	}
	return statement, nil
}

func (dir TaskDirReader) Example(index int) (Example, error) {
	notePath := fmt.Sprintf("examples/%03d.md", index)
	noteContent, err := dir.ReadFile(notePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		msg := fmt.Sprintf("read example note %d", index)
		return Example{}, etrace.Wrap(msg, err)
	}
	note := parseMultilingualMd(string(noteContent))
	inPath := fmt.Sprintf("examples/%03di.txt", index)
	inContent, err := dir.ReadFile(inPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		msg := fmt.Sprintf("read example input %d", index)
		return Example{}, etrace.Wrap(msg, err)
	}
	outPath := fmt.Sprintf("examples/%03do.txt", index)
	outContent, err := dir.ReadFile(outPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		msg := fmt.Sprintf("read example output %d", index)
		return Example{}, etrace.Wrap(msg, err)
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
		return []Example{}, etrace.Wrap(msg, err)
	}

	idxMap := make(map[int]bool)
	for _, file := range files {
		index, err := strconv.Atoi(file[:3])
		if err != nil {
			msg := fmt.Sprintf("parse example index %s", file)
			return []Example{}, etrace.NewError(msg)
		}
		idxMap[index] = true
	}

	maxIdx, err := ensureConsecutiveIdsFrom1(idxMap)
	if err != nil {
		return []Example{}, etrace.NewError(err.Error())
	}

	examples := make([]Example, maxIdx)
	for i := 1; i <= maxIdx; i++ {
		example, err := dir.Example(i)
		if err != nil {
			msg := fmt.Sprintf("read example %d", i)
			return []Example{}, etrace.Wrap(msg, err)
		}
		examples[i-1] = example
	}
	return examples, nil
}

func parseMultilingualMd(content string) I18N[string] {
	result := make(I18N[string])
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
		return []TestGroup{}, etrace.Wrap(msg, err)
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
			return []TestGroup{}, etrace.Wrap(msg, err)
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
		return TestGroup{}, etrace.NewError(msg)
	}
	matches[0] = matches[0][1:]
	ints := make([]int, len(matches[0]))
	for j, match := range matches[0] {
		ints[j], err = strconv.Atoi(match)
		if err != nil {
			msg := fmt.Sprintf("converting %s to int", match)
			return TestGroup{}, etrace.NewError(msg)
		}
	}
	if len(ints) != 4 && len(ints) != 5 {
		msg := "invalid no. of parts"
		return TestGroup{}, etrace.NewError(msg)
	}
	if ints[0] != idx {
		msg := fmt.Sprintf("tg id %d does not match idx %d", ints[0], idx)
		return TestGroup{}, etrace.NewError(msg)
	}
	tg.Range = [2]int{ints[1], ints[2]}
	tg.Points = ints[3]
	if len(ints) == 5 {
		tg.Subtask = ints[4]
	}
	return tg, nil
}

func (dir TaskDirReader) Scoring(noOfTests int, subtasks []Subtask) (Scoring, error) {
	taskToml, err := dir.Toml()
	if err != nil {
		msg := "read task.toml"
		return Scoring{}, etrace.Wrap(msg, err)
	}
	tgs, err := dir.TestGroups()
	if err != nil {
		msg := "read testgroups.txt"
		return Scoring{}, etrace.Wrap(msg, err)
	}
	scoring := Scoring{
		ScoringT: taskToml.Scoring.Type,
		TotalP:   taskToml.Scoring.Total,
		Groups:   tgs,
	}
	if err := scoring.Validate(noOfTests, subtasks); err != nil {
		msg := "validate scoring"
		return Scoring{}, etrace.Wrap(msg, err)
	}
	return scoring, nil
}

func (dir TaskDirReader) Archive() (Archive, error) {
	files, err := dir.ListDir("archive")
	if err != nil {
		msg := "list archive dir"
		return Archive{}, etrace.Wrap(msg, err)
	}
	archive := Archive{}
	for _, file := range files {
		content, err := dir.ReadFile(filepath.Join("archive", file))
		if err != nil {
			msg := fmt.Sprintf("read archive file %s", file)
			return Archive{}, etrace.Wrap(msg, err)
		}
		archive.Files = append(archive.Files, ArchiveFile{
			RelPath: file,
			Content: content,
		})
	}
	return archive, nil
}

func (dir TaskDirReader) Task() (task Task, errs error) {
	var err error
	var taskToml TaskToml
	taskToml, err = dir.Toml()
	if err != nil {
		if etrace.IsCritical(err) {
			return task, err
		}
		errs = errors.Join(errs, etrace.Trace(err))
	}

	task.ShortID = taskToml.Id
	task.FullName = taskToml.Name

	task.Testing, err = dir.Testing()
	if err != nil {
		if etrace.IsCritical(err) {
			return task, err
		}
		errs = errors.Join(errs, etrace.Trace(err))
	}

	task.ReadMe, err = dir.Readme()
	if err != nil {
		if etrace.IsCritical(err) {
			return task, err
		}
		errs = errors.Join(errs, etrace.Trace(err))
	}

	task.Origin, err = dir.Origin()
	if err != nil {
		if etrace.IsCritical(err) {
			return task, err
		}
		errs = errors.Join(errs, etrace.Trace(err))
	}

	task.Metadata, err = dir.Metadata()
	if err != nil {
		if etrace.IsCritical(err) {
			return task, err
		}
		errs = errors.Join(errs, etrace.Trace(err))
	}

	task.Solutions, err = dir.Solutions()
	if err != nil {
		if etrace.IsCritical(err) {
			return task, err
		}
		errs = errors.Join(errs, etrace.Trace(err))
	}

	task.Statement, err = dir.Statement()
	if err != nil {
		if etrace.IsCritical(err) {
			return task, err
		}
		errs = errors.Join(errs, etrace.Trace(err))
	}

	task.Scoring, err = dir.Scoring(len(task.Testing.Tests), task.Statement.Subtasks)
	if err != nil {
		if etrace.IsCritical(err) {
			return task, err
		}
		errs = errors.Join(errs, etrace.Trace(err))
	}

	task.Archive, err = dir.Archive()
	if err != nil {
		if etrace.IsCritical(err) {
			return task, err
		}
		errs = errors.Join(errs, etrace.Trace(err))
	}

	return
}

func (dir TaskDirReader) AllFilesWereRead() bool {
	return len(dir.readPaths) == len(dir.allPaths)
}
