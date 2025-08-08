package taskfs

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/programme-lv/task-zip/common/errwrap"
	"github.com/programme-lv/task-zip/common/fn"
	"github.com/programme-lv/task-zip/common/iso639"
)

// internationalization (language -> text or smth)
// TODO: consider https://github.com/emvi/iso-639-1
type I18N[T any] map[string]T

var ErrInvalidIso639LangCode = errwrap.Error("invalid ISO 639 language code")

func (m I18N[T]) ValidateLangs() error {
	for lang := range m {
		if _, ok := iso639.Languages[lang]; !ok {
			return errwrap.Trace(ErrInvalidIso639LangCode)
		}
	}
	return nil
}

type Task struct {
	ShortID   string // unique identifier; should match .zip filename
	FullName  I18N[string]
	ReadMe    string // readme md. all kinds of notes for maintainers.
	Statement Statement
	Origin    Origin
	Testing   Testing
	Scoring   Scoring
	Archive   Archive
	Solutions []Solution
	Metadata  Metadata
}

const MaxShortIDLen = 20

var (
	ErrShortIDEmpty   = errwrap.Error("shortID cannot be empty")
	ErrShortIDTooLong = errwrap.Error(fmt.Sprintf("shortID too long, max %d chars", MaxShortIDLen))
	ErrShortIDInvalid = errwrap.Error("shortID must contain only lowercase letters and digits")
)

func (t *Task) Validate() (err error) {
	if len(t.ShortID) == 0 {
		err = errors.Join(err, errwrap.Trace(ErrShortIDEmpty))
	}
	if len(t.ShortID) > MaxShortIDLen {
		err = errors.Join(err, errwrap.Trace(ErrShortIDTooLong))
	}
	for _, r := range t.ShortID {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			err = errors.Join(err, errwrap.Trace(ErrShortIDInvalid))
		}
	}

	if validateErr := t.Metadata.Validate(); validateErr != nil {
		msg := "validate metadata"
		err = errors.Join(err, errwrap.Wrap(msg, validateErr))
	}

	if validateErr := t.Origin.Validate(); validateErr != nil {
		msg := "validate origin"
		err = errors.Join(err, errwrap.Wrap(msg, validateErr))
	}

	if validateErr := t.Testing.Validate(); validateErr != nil {
		msg := "validate testing"
		err = errors.Join(err, errwrap.Wrap(msg, validateErr))
	}

	if validateErr := t.Statement.Validate(); validateErr != nil {
		msg := "validate statement"
		err = errors.Join(err, errwrap.Wrap(msg, validateErr))
	}

	noOfTests := len(t.Testing.Tests)
	if validateErr := t.Scoring.Validate(noOfTests, t.Statement.Subtasks); validateErr != nil {
		msg := "validate scoring"
		err = errors.Join(err, errwrap.Wrap(msg, validateErr))
	}

	return err
}

func (t *Task) ValidateOld() error {
	if len(t.ShortID) == 0 {
		return errwrap.Error("shortID cannot be empty")
	}
	if len(t.ShortID) > 20 {
		return errwrap.Error("shortID too long, max 20 chars")
	}
	for _, r := range t.ShortID {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			return errwrap.Error("shortID must contain only lowercase letters and digits")
		}
	}

	if err := t.Metadata.Validate(); err != nil {
		return errwrap.Trace(err)
	}

	if err := t.Origin.ValidateOld(); err != nil {
		return errwrap.Trace(err)
	}

	if err := t.Testing.Validate(); err != nil {
		return errwrap.Trace(err)
	}

	if err := t.Statement.Validate(); err != nil {
		return errwrap.Trace(err)
	}

	noOfTests := len(t.Testing.Tests)
	if err := t.Scoring.Validate(noOfTests, t.Statement.Subtasks); err != nil {
		return errwrap.Trace(err)
	}

	return nil
}

type Metadata struct {
	ProblemTags []string
	Difficulty  int // in programme.lv, difficulty ranges from 1 to 6
}

// validates sanity of the metadata configuration
func (m *Metadata) Validate() error {
	if m.Difficulty != 0 && (m.Difficulty < 1 || m.Difficulty > 6) {
		return errwrap.Error("difficulty must be between 1 and 6")
	}

	if len(m.ProblemTags) > 20 {
		return errwrap.Error("max 20 problem tags allowed")
	}

	for _, tag := range m.ProblemTags {
		if len(tag) == 0 {
			return errwrap.Error("problem tag cannot be empty")
		}
		if len(tag) > 50 {
			return errwrap.Error("problem tag too long, max 50 chars")
		}
		// Tags should contain only lowercase letters, digits, and hyphens
		for _, r := range tag {
			if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
				return errwrap.Error("problem tag must contain only lowercase letters, digits, and hyphens")
			}
		}
	}

	return nil
}

type Origin struct {
	Olympiad string // abbrev of the olympiad name, if any
	OlyStage string
	Org      string       // abbrev of an organization or institution, if any.
	Notes    I18N[string] // language -> note. full name of olymp, org + details
	Authors  []string     // first name + last name list
	Year     string       // yyyy | yyyy/yyyy e.g. 2024/2025.
}

var OlympStages = []string{"online", "school", "municipal", "national", "selection", "regional", "international"}
var MaxAbbrevLen = 10
var MaxOrigNoteLen = 200
var MaxAuthorNameLen = 50
var MaxNoOfAuthors = 10

var (
	ErrOlympAbbrevInvalid = errwrap.Error(fmt.Sprintf("olympiad (abbrev) must be uppercase, alphanumeric, max %d chars", MaxAbbrevLen))
	ErrStageWithoutOlymp  = errwrap.Error("olympiad stage can't be set if olympiad is not set")
	WarnStageNotSet       = errwrap.Warning("stage should be set as the olympiad is set")
	WarnUnknownOlympStage = errwrap.Warning("stage should be one of [" + strings.Join(OlympStages, ", ") + "]")
	WarnNonTraceableTask  = errwrap.Warning("task origin can't be traced back to olympiad, organization, or author")
	ErrOrgAbbrevInvalid   = errwrap.Error(fmt.Sprintf("org must be uppercase letters/digits, max %d chars", MaxAbbrevLen))
	WarnOriginNoteTooLong = errwrap.Warning(fmt.Sprintf("note should be short and therefore at most %d chars", MaxOrigNoteLen))
	WarnAuthorNameTooLong = errwrap.Warning(fmt.Sprintf("author name should be at most %d chars", MaxAuthorNameLen))
	WarnTooManyAuthors    = errwrap.Warning(fmt.Sprintf("max %d authors allowed", MaxNoOfAuthors))
)

func (o *Origin) Validate() (err error) {
	if len(o.Olympiad) > MaxAbbrevLen || !isUpperOrDigits(o.Olympiad) {
		err = errors.Join(err, errwrap.Trace(ErrOlympAbbrevInvalid))
	}
	if o.Olympiad == "" && o.OlyStage != "" {
		err = errors.Join(err, errwrap.Trace(ErrStageWithoutOlymp))
	}
	if o.Olympiad != "" && o.OlyStage == "" {
		err = errors.Join(err, errwrap.Trace(WarnStageNotSet))
	}
	if o.OlyStage != "" {
		if !slices.Contains(OlympStages, o.OlyStage) {
			err = errors.Join(err, errwrap.Trace(WarnUnknownOlympStage))
		}
	}
	if !(len(o.Olympiad) > 0 || len(o.Org) > 0 || len(o.Authors) > 0) {
		err = errors.Join(err, errwrap.Trace(WarnNonTraceableTask))
	}
	if len(o.Org) > MaxAbbrevLen || !isUpperOrDigits(o.Org) {
		err = errors.Join(err, errwrap.Trace(ErrOrgAbbrevInvalid))
	}
	for _, note := range o.Notes {
		if len(note) > MaxOrigNoteLen {
			err = errors.Join(err, errwrap.Trace(WarnOriginNoteTooLong))
			break
		}
	}
	for _, author := range o.Authors {
		if len(author) > MaxAuthorNameLen {
			err = errors.Join(err, errwrap.Trace(WarnAuthorNameTooLong))
			break
		}
	}
	if len(o.Authors) > MaxNoOfAuthors {
		err = errors.Join(err, errwrap.Trace(WarnTooManyAuthors))
	}
	if err := o.Notes.ValidateLangs(); err != nil {
		err = errors.Join(err, errwrap.Trace(err))
	}
	if err := ValidateOriginYear(o.Year); err != nil {
		err = errors.Join(err, errwrap.Trace(err))
	}

	return err
}

const MinYear = 1980

var (
	ErrInvalidYearFormat   = errwrap.Error("invalid year format, must be yyyy or yyyy/yyyy")
	ErrYearTooEarly        = errwrap.Error(fmt.Sprintf("year must be at least %d", MinYear))
	ErrYearsNotConsecutive = errwrap.Error("origin years must be consecutive")
	WarnYearInTheFuture    = errwrap.Warning("origin year is in the future")
)

func ValidateOriginYear(year string) error {
	if !strings.Contains(year, "/") {
		if len(year) != 4 {
			return ErrInvalidYearFormat
		}
		yearInt, err := parseYear(year)
		if err != nil {
			return err
		}
		if yearInt < MinYear {
			return ErrYearTooEarly
		}
		if yearInt > time.Now().Year() {
			return WarnYearInTheFuture
		}
		return nil
	} else {
		parts := strings.Split(year, "/")
		if len(parts) != 2 {
			return ErrInvalidYearFormat
		}
		if len(parts[0]) != 4 || len(parts[1]) != 4 {
			return ErrInvalidYearFormat
		}
		first, err := parseYear(parts[0])
		if err != nil {
			return err
		}
		second, err := parseYear(parts[1])
		if err != nil {
			return err
		}
		if first != second-1 {
			return ErrYearsNotConsecutive
		}
		if first < MinYear {
			return ErrYearTooEarly
		}
		if first > time.Now().Year() {
			return WarnYearInTheFuture
		}
	}
	return nil
}

// validates sanity of the origin configuration
func (o *Origin) ValidateOld() error {
	if len(o.Olympiad) > 10 || !isUpperOrDigits(o.Olympiad) {
		return errwrap.Error("olympiad must be uppercase letters/digits, max 10 chars")
	}

	if o.Olympiad == "" {
		if o.OlyStage != "" {
			return errwrap.Error("olympiad stage must be empty if olympiad is empty")
		}
	} else {
		validStages := []string{"school", "municipal", "national", "selection", "regional", "international"}
		if !slices.Contains(validStages, o.OlyStage) {
			return errwrap.Error("invalid olympiad stage")
		}
	}

	if len(o.Org) > 10 || !isUpperOrDigits(o.Org) {
		return errwrap.Error("org must be uppercase letters/digits, max 10 chars")
	}

	for _, note := range o.Notes {
		if len(note) > 500 {
			return errwrap.Error("note too long, max 500 chars")
		}
	}

	if len(o.Authors) == 0 {
		return errwrap.Error("at least 1 author required")
	}
	if len(o.Authors) > 10 {
		return errwrap.Error("max 10 authors allowed")
	}
	for _, author := range o.Authors {
		if len(author) > 50 {
			return errwrap.Error("author name too long, max 50 chars")
		}
	}

	// Year format: yyyy or yyyy/yyyy
	if !strings.Contains(o.Year, "/") {
		year, err := parseYear(o.Year)
		if err != nil {
			return errwrap.Error(err.Error())
		}
		if year < 1980 {
			return errwrap.Error("year must be at least 1980")
		}
	} else {
		parts := strings.Split(o.Year, "/")
		if len(parts) != 2 {
			return errwrap.Error("invalid year format, must be yyyy or yyyy/yyyy")
		}

		start, err := parseYear(parts[0])
		if err != nil {
			return errwrap.Error(err.Error())
		}
		end, err := parseYear(parts[1])
		if err != nil {
			return errwrap.Error(err.Error())
		}

		if start < 1980 {
			return errwrap.Error("year must be at least 1980")
		}

		if end != start+1 {
			return errwrap.Error("years must be consecutive")
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
		return errwrap.Error(fmt.Sprintf("invalid testing type - %s", t.TestingT))
	}
	checker := t.Checker != ""
	if (t.TestingT == "checker" && !checker) || (t.TestingT != "checker" && checker) {
		return errwrap.Error("checker is required iff testing type is checker")
	}
	interactor := t.Interactor != ""
	if (t.TestingT == "interactor" && !interactor) || (t.TestingT != "interactor" && interactor) {
		return errwrap.Error("interactor is required iff testing type is interactor")
	}
	if len(t.Tests) == 0 {
		return errwrap.Error("at least 1 test is required")
	}
	if len(t.Tests) > 999 {
		return errwrap.Error("max 999 tests allowed")
	}
	if t.MemLimMiB < 40 {
		return errwrap.Error("memory limit must be at least 40 MiB")
	}
	if t.MemLimMiB > 2048 {
		return errwrap.Error("memory limit must be at most 2048 MiB")
	}
	if t.CpuLimMs < 100 {
		return errwrap.Error("cpu time limit must be at least 100 ms")
	}
	if t.CpuLimMs > 8000 {
		return errwrap.Error("cpu time limit must be at most 8000 ms")
	}
	if len(t.Checker) > 1e6 {
		return errwrap.Error("checker must be at most 1 MB")
	}
	if len(t.Interactor) > 1e6 {
		return errwrap.Error("interactor must be at most 1 MB")
	}
	// tests can't weigh more than 500 MB
	totalTestSize := 0
	for _, test := range t.Tests {
		totalTestSize += len(test.Input) + len(test.Answer)
	}
	if totalTestSize > 500*1024*1024 {
		return errwrap.Error("tests must be at most 500 MB")
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
		return errwrap.Error("test groups not allowed for test-sum scoring")
	}
	if s.TotalP != noOfTests {
		return errwrap.Error("total points must equal number of tests for test-sum scoring")
	}
	return nil
}

func (s *Scoring) validateMinGroupsT(noOfTests int, subtasks []Subtask) error {
	hasGroups := len(s.Groups) > 0
	if !hasGroups {
		return errwrap.Error("test groups required for min-groups scoring")
	}
	if err := s.validateGroupSubtaskLinks(len(subtasks)); err != nil {
		return err
	}
	if err := s.validateGroupPointSumPerSubtask(subtasks); err != nil {
		return err
	}
	if err := s.validateGroupPointSum(); err != nil {
		return err
	}
	if err := s.validateGroupTestsOkay(noOfTests); err != nil {
		return err
	}
	return nil
}

var ErrGroupTestIdxOutOfRange = errwrap.Error("tg test idx out of range")
var WarnGroupTestIdxBadOrdering = errwrap.Warning("tg test idx should be in ascending order")
var ErrGroupTestIdxOverlapping = errwrap.Error("tg test idx overlapping")

func (s *Scoring) validateGroupTestsOkay(noOfTests int) error {
	for _, group := range s.Groups {
		if group.Range[0] < 1 || group.Range[1] > noOfTests {
			return errwrap.Wrap(fmt.Sprintf("tg test idx %d-%d out of range (1-%d)", group.Range[0], group.Range[1], noOfTests), ErrGroupTestIdxOutOfRange)
		}
	}

	for i, group1 := range s.Groups {
		for j, group2 := range s.Groups {
			if i == j {
				continue
			}
			if group1.Range[0] <= group2.Range[1] && group2.Range[0] <= group1.Range[1] {
				return errwrap.Trace(ErrGroupTestIdxOverlapping)
			}
		}
	}

	for i, group := range s.Groups {
		if i > 0 {
			prevGroup := s.Groups[i-1]
			if group.Range[0] < prevGroup.Range[0] {
				return errwrap.Trace(WarnGroupTestIdxBadOrdering)
			}
		}
	}

	return nil
}

func (s *Scoring) validateGroupPointSum() error {
	sumPoints := 0
	for _, group := range s.Groups {
		if group.Points <= 0 {
			return errwrap.Error("test group points must be positive")
		}
		sumPoints += group.Points
	}
	if sumPoints != s.TotalP {
		return errwrap.Error("sum of test group points must equal total points")
	}
	return nil
}

var ErrSubtaskGroupSumPointsMismatch = errwrap.Error("subtask points must equal sum over its groups")

func (s *Scoring) validateGroupPointSumPerSubtask(subtasks []Subtask) error {
	pointsPerSubtask := make([]int, len(subtasks))
	for _, group := range s.Groups {
		pointsPerSubtask[group.Subtask-1] += group.Points
	}
	for i, subtask := range subtasks {
		if pointsPerSubtask[i] != subtask.Points {
			msg := fmt.Sprintf("subtask %d points %d != sum of its groups %d", i+1, subtask.Points, pointsPerSubtask[i])
			return errwrap.Wrap(msg, ErrSubtaskGroupSumPointsMismatch)
		}
	}
	return nil
}

func (s *Scoring) validateGroupSubtaskLinks(noOfSubtasks int) error {
	tgStLink := func(group TestGroup) int { return group.Subtask }
	count := len(fn.Unique(fn.Map(s.Groups, tgStLink)))
	if count != noOfSubtasks {
		return errwrap.Error("all subtasks must be linked to in testgroups")
	}
	if noOfSubtasks == 0 && count == 0 {
		return nil
	}
	if noOfSubtasks != count {
		return errwrap.Error("testgroups must link to existing subtasks")
	}

	outOfRange := func(link int) bool { return link < 1 || link > noOfSubtasks }
	anyOutOfRange := fn.Any(fn.Map(s.Groups, tgStLink), outOfRange)
	if anyOutOfRange {
		return errwrap.Error("subtask link in testgroups are out of range")
	}

	return nil
}

func (s *Scoring) Validate(noOfTests int, subtasksIfAny []Subtask) error {
	if s.TotalP <= 0 {
		return errwrap.Error("total points must be positive")
	}
	if s.ScoringT == "test-sum" {
		return s.validateTestSumT(noOfTests)
	}
	if s.ScoringT == "min-groups" {
		return s.validateMinGroupsT(noOfTests, subtasksIfAny)
	}
	return errwrap.Error(fmt.Sprintf("invalid scoring type - %s", s.ScoringT))
}

type Statement struct {
	Stories  I18N[StoryMd]
	Subtasks []Subtask
	Examples []Example
	Images   []Image
}

func (s *Statement) Validate() error {
	for _, example := range s.Examples {
		if err := example.Validate(); err != nil {
			return errwrap.Trace(err)
		}
	}
	return nil
}

type Subtask struct {
	Desc     I18N[string] // description
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
	MdNote I18N[string]
}

func (e *Example) Validate() error {
	if len(e.Input) > 1024 {
		return errwrap.Error("input too long, max 1024 bytes")
	}
	if len(e.Output) > 1024 {
		return errwrap.Error("output too long, max 1024 bytes")
	}
	if len(e.Input) == 0 || len(e.Output) == 0 {
		return errwrap.Error("input and output must not be empty")
	}
	for _, note := range e.MdNote {
		if len(note) > 1000 {
			return errwrap.Error("note too long, max 1000 chars")
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
	Talk    string // aka communication (interactive tasks)
	Example string // maybe grader usage examples...
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

func (t Archive) GetTestlibValidator() string {
	prefix := "reserved/validator.cpp"
	ext := ".cpp"
	for _, file := range t.Files {
		if strings.HasSuffix(file.RelPath, ext) &&
			strings.HasPrefix(file.RelPath, prefix) {
			return string(file.Content)
		}
	}
	return ""
}
