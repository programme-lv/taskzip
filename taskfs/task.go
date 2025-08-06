package taskfs

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BooleanCat/go-functional/v2/it"
	"github.com/programme-lv/task-zip/common/errwrap"
)

// internationalization (language -> text or smth)
// TODO: consider https://github.com/emvi/iso-639-1
type i18n[T any] map[string]T

type Task struct {
	ShortID   string // unique identifier; should match .zip filename
	FullName  i18n[string]
	ReadMe    string // readme md. all kinds of notes for maintainers.
	Statement Statement
	Origin    Origin
	Testing   Testing
	Scoring   Scoring
	Archive   Archive
	Solutions []Solution
	Metadata  Metadata
}

func (t *Task) Validate() error {
	if len(t.ShortID) == 0 {
		return errwrap.ClientError("shortID cannot be empty")
	}
	if len(t.ShortID) > 20 {
		return errwrap.ClientError("shortID too long, max 20 chars")
	}
	for _, r := range t.ShortID {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			return errwrap.ClientError("shortID must contain only lowercase letters and digits")
		}
	}

	if err := t.Metadata.Validate(); err != nil {
		return errwrap.AddTrace(err)
	}

	if err := t.Origin.Validate(); err != nil {
		return errwrap.AddTrace(err)
	}

	if err := t.Testing.Validate(); err != nil {
		return errwrap.AddTrace(err)
	}

	if err := t.Statement.Validate(); err != nil {
		return errwrap.AddTrace(err)
	}

	noOfTests := len(t.Testing.Tests)
	noOfSubtasks := len(t.Statement.Subtasks)
	if err := t.Scoring.Validate(noOfTests, noOfSubtasks); err != nil {
		return errwrap.AddTrace(err)
	}

	return nil
}

type Metadata struct {
	ProblemTags []string
	Difficulty  int // in programme.lv, difficulty ranges from 1 to 6
}

// validates sanity of the metadata configuration
func (m *Metadata) Validate() error {
	if m.Difficulty < 1 || m.Difficulty > 6 {
		return errwrap.ClientError("difficulty must be between 1 and 6")
	}

	if len(m.ProblemTags) > 20 {
		return errwrap.ClientError("max 20 problem tags allowed")
	}

	for _, tag := range m.ProblemTags {
		if len(tag) == 0 {
			return errwrap.ClientError("problem tag cannot be empty")
		}
		if len(tag) > 50 {
			return errwrap.ClientError("problem tag too long, max 50 chars")
		}
		// Tags should contain only lowercase letters, digits, and hyphens
		for _, r := range tag {
			if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
				return errwrap.ClientError("problem tag must contain only lowercase letters, digits, and hyphens")
			}
		}
	}

	return nil
}

type Origin struct {
	Olympiad string       // abbrev of the olympiad name, if any
	OlyStage string       // school | municipal | national | selection | international
	Org      string       // abbrev of an organization or institution, if any
	Notes    i18n[string] // language -> note. full name of olymp, org + details
	Authors  []string     // first name + last name list
	Year     string       // yyyy | yyyy/yyyy e.g. 2024/2025.
}

// validates sanity of the origin configuration
func (o *Origin) Validate() error {
	if len(o.Olympiad) > 10 || !isUpperOrDigits(o.Olympiad) {
		return errwrap.ClientError("olympiad must be uppercase letters/digits, max 10 chars")
	}

	validStages := []string{"school", "municipal", "national", "selection", "international"}
	if !slices.Contains(validStages, o.OlyStage) {
		return errwrap.ClientError("invalid olympiad stage")
	}

	if len(o.Org) > 10 || !isUpperOrDigits(o.Org) {
		return errwrap.ClientError("org must be uppercase letters/digits, max 10 chars")
	}

	for _, note := range o.Notes {
		if len(note) > 500 {
			return errwrap.ClientError("note too long, max 500 chars")
		}
	}

	if len(o.Authors) == 0 {
		return errwrap.ClientError("at least 1 author required")
	}
	if len(o.Authors) > 10 {
		return errwrap.ClientError("max 10 authors allowed")
	}
	for _, author := range o.Authors {
		if len(author) > 50 {
			return errwrap.ClientError("author name too long, max 50 chars")
		}
	}

	// Year format: yyyy or yyyy/yyyy
	if !strings.Contains(o.Year, "/") {
		year, err := parseYear(o.Year)
		if err != nil {
			return errwrap.ClientError(err.Error())
		}
		if year < 1980 {
			return errwrap.ClientError("year must be at least 1980")
		}
	} else {
		parts := strings.Split(o.Year, "/")
		if len(parts) != 2 {
			return errwrap.ClientError("invalid year format, must be yyyy or yyyy/yyyy")
		}

		start, err := parseYear(parts[0])
		if err != nil {
			return errwrap.ClientError(err.Error())
		}
		end, err := parseYear(parts[1])
		if err != nil {
			return errwrap.ClientError(err.Error())
		}

		if start < 1980 {
			return errwrap.ClientError("year must be at least 1980")
		}

		if end != start+1 {
			return errwrap.ClientError("years must be consecutive")
		}
	}

	return nil
}

func parseYear(s string) (int, error) {
	if len(s) != 4 {
		return 0, fmt.Errorf("invalid year format, must be yyyy")
	}
	var year int
	_, err := fmt.Sscanf(s, "%d", &year)
	if err != nil {
		return 0, fmt.Errorf("invalid year format, must be yyyy")
	}
	return year, nil
}

func isUpperOrDigits(s string) bool {
	for _, r := range s {
		if !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

type Testing struct {
	TestingT   string // testing type. documented in readme.md
	MemLimMiB  int    // rss memory limit in mebibytes
	CpuLimMs   int    // cpu time limit in milliseconds
	Tests      []Test // no more than 999...
	Checker    string // only if testingT == "checker"
	Interactor string // only if testingT == "interactor"
}

// validates sanity of the testing configuration
func (t *Testing) Validate() error {
	validTypes := []string{"simple", "checker", "interactor"}
	if !slices.Contains(validTypes, t.TestingT) {
		return errwrap.ClientError(fmt.Sprintf("invalid testing type - %s", t.TestingT))
	}
	checker := t.Checker != ""
	if (t.TestingT == "checker" && !checker) || (t.TestingT != "checker" && checker) {
		return errwrap.ClientError("checker is required iff testing type is checker")
	}
	interactor := t.Interactor != ""
	if (t.TestingT == "interactor" && !interactor) || (t.TestingT != "interactor" && interactor) {
		return errwrap.ClientError("interactor is required iff testing type is interactor")
	}
	if len(t.Tests) == 0 {
		return errwrap.ClientError("at least 1 test is required")
	}
	if len(t.Tests) > 999 {
		return errwrap.ClientError("max 999 tests allowed")
	}
	if t.MemLimMiB < 40 {
		return errwrap.ClientError("memory limit must be at least 40 MiB")
	}
	if t.MemLimMiB > 2048 {
		return errwrap.ClientError("memory limit must be at most 2048 MiB")
	}
	if t.CpuLimMs < 100 {
		return errwrap.ClientError("cpu time limit must be at least 100 ms")
	}
	if t.CpuLimMs > 8000 {
		return errwrap.ClientError("cpu time limit must be at most 8000 ms")
	}
	if len(t.Checker) > 1e6 {
		return errwrap.ClientError("checker must be at most 1 MB")
	}
	if len(t.Interactor) > 1e6 {
		return errwrap.ClientError("interactor must be at most 1 MB")
	}
	// tests can't weigh more than 500 MB
	totalTestSize := 0
	for _, test := range t.Tests {
		totalTestSize += len(test.Input) + len(test.Answer)
	}
	if totalTestSize > 500*1024*1024 {
		return errwrap.ClientError("tests must be at most 500 MB")
	}
	return nil
}

type Scoring struct {
	ScoringT string      // scoring type. documented in readme.md
	TotalP   int         // total/max points. to verify correct configuration.
	Groups   []TestGroup // can be 1:1 to subtasks. nil if scoringT == "test-sum".
}

func (s *Scoring) validateTestSumT(noOfTests int) error {
	if len(s.Groups) > 0 {
		return errwrap.ClientError("test groups not allowed for test-sum scoring")
	}
	if s.TotalP != noOfTests {
		return errwrap.ClientError("total points must equal number of tests for test-sum scoring")
	}
	return nil
}

func (s *Scoring) validateMinGroupsT(noOfSubtasks int) error {
	hasGroups := len(s.Groups) > 0
	if !hasGroups {
		return errwrap.ClientError("test groups required for min-groups scoring")
	}
	if err := s.validateGroupSubtaskLinks(noOfSubtasks); err != nil {
		return err
	}
	if err := s.validateGroupPointSum(); err != nil {
		return err
	}
	return nil
}

func (s *Scoring) validateGroupPointSum() error {
	sumPoints := 0
	for _, group := range s.Groups {
		if group.Points <= 0 {
			return errwrap.ClientError("test group points must be positive")
		}
		sumPoints += group.Points
	}
	if sumPoints != s.TotalP {
		return errwrap.ClientError("sum of test group points must equal total points")
	}
	return nil
}

func (s *Scoring) validateGroupSubtaskLinks(noOfSubtasks int) error {
	tgStLink := func(group TestGroup) int { return group.Subtask }
	stLinks := it.Map(slices.Values(s.Groups), tgStLink)
	count := len(slices.Collect(it.FilterUnique(stLinks)))
	if count != noOfSubtasks {
		return errwrap.ClientError("all subtasks must be linked to in testgroups")
	}
	if noOfSubtasks == 0 && count == 0 {
		return nil
	}
	if noOfSubtasks != count {
		return errwrap.ClientError("testgroups must link to existing subtasks")
	}

	outOfRange := func(link int) bool { return link < 1 || link > noOfSubtasks }
	anyOutOfRange := it.Any(it.Map(stLinks, outOfRange))
	if anyOutOfRange {
		return errwrap.ClientError("subtask link in testgroups are out of range")
	}

	return nil
}

func (s *Scoring) Validate(noOfTests int, noOfSubtasks int) error {
	if s.TotalP <= 0 {
		return errwrap.ClientError("total points must be positive")
	}
	if s.ScoringT == "test-sum" {
		return s.validateTestSumT(noOfTests)
	}
	if s.ScoringT == "min-groups" {
		return s.validateMinGroupsT(noOfSubtasks)
	}
	return errwrap.ClientError(fmt.Sprintf("invalid scoring type - %s", s.ScoringT))
}

type Statement struct {
	Stories  i18n[StoryMd]
	Subtasks []Subtask
	Examples []Example
	Images   []Image
}

func (s *Statement) Validate() error {
	for _, example := range s.Examples {
		if err := example.Validate(); err != nil {
			return errwrap.AddTrace(err)
		}
	}
	return nil
}

type Subtask struct {
	Desc     i18n[string] // description
	Points   int
	VisInput bool // compatibility with latvian informatics olympiad (LIO)
}

type TestGroup struct {
	Points  int
	Range   [2]int // [from, to] (inclusive)
	Public  bool   // results visible during contest
	Subtask int    // subtask it belongs to. 0 if nil
}

type Test struct {
	Input  string
	Answer string
}

type Example struct {
	Input  string
	Output string
	MdNote i18n[string]
}

func (e *Example) Validate() error {
	if len(e.Input) > 1024 {
		return errwrap.ClientError("input too long, max 1024 bytes")
	}
	if len(e.Output) > 1024 {
		return errwrap.ClientError("output too long, max 1024 bytes")
	}
	if len(e.Input) == 0 || len(e.Output) == 0 {
		return errwrap.ClientError("input and output must not be empty")
	}
	for _, note := range e.MdNote {
		if len(note) > 1000 {
			return errwrap.ClientError("note too long, max 1000 chars")
		}
	}
	return nil
}

type OriginalPdf struct {
	Language string
	Content  []byte
}

type StoryMd struct {
	Story   string
	Input   string
	Output  string
	Notes   string // usually has explanations of examples
	Scoring string // e.g. tasks with partial scoring
	Example string // maybe grader usage examples...
	Talk    string // aka communication (interactive tasks)
}

type Solution struct {
	Fname    string // filename
	Subtasks []int  // subtasks that it should correctly solve
	Content  string
}

type ArchiveFile struct {
	RelPath string // relative to archive root
	Content []byte
}

type Image struct {
	Fname   string
	Content []byte
}

type Archive struct {
	Files []ArchiveFile
}

func (t Archive) GetIllustrImgs() []Image {
	prefix := "reserved/illustration/img."
	imgs := []Image{}
	for _, file := range t.Files {
		if strings.HasPrefix(file.RelPath, prefix) {
			imgs = append(imgs, Image{
				Fname:   filepath.Base(file.RelPath),
				Content: file.Content,
			})
		}
	}
	return imgs
}

func (t Archive) GetOgStatementPdfs() []OriginalPdf {
	prefix := "reserved/statement/"
	ext := ".pdf"
	pdfs := []OriginalPdf{}
	for _, file := range t.Files {
		if strings.HasSuffix(file.RelPath, ext) &&
			strings.HasPrefix(file.RelPath, prefix) {
			lang := strings.TrimSuffix(strings.TrimPrefix(file.RelPath, prefix), ext)
			pdfs = append(pdfs, OriginalPdf{
				Language: lang,
				Content:  file.Content,
			})
		}
	}
	return pdfs
}
