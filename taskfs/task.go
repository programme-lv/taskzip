package taskfs

import (
	"fmt"
	"slices"
	"strings"
)

// internationalization (language -> text or smth)
// TODO: consider https://github.com/emvi/iso-639-1
type i18n[T any] map[string]T

type Task struct {
	ShortID  string // unique identifier; should match .zip filename
	FullName i18n[string]
	ReadMe   string // readme md. all kinds of notes for maintainers.

	Origin    Origin
	Testing   Testing
	Scoring   Scoring
	Archive   Archive
	Solutions Solutions
	Metadata  Metadata
}

type Metadata struct {
	ProblemTags []string
	Difficulty  int // in programme.lv, difficulty ranges from 1 to 6
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
		return wrap("olympiad must be uppercase letters/digits, max 10 chars")
	}

	validStages := []string{"school", "municipal", "national", "selection", "international"}
	if !slices.Contains(validStages, o.OlyStage) {
		return wrap("invalid olympiad stage")
	}

	if len(o.Org) > 10 || !isUpperOrDigits(o.Org) {
		return wrap("org must be uppercase letters/digits, max 10 chars")
	}

	for _, note := range o.Notes {
		if len(note) > 500 {
			return wrap("note too long, max 500 chars")
		}
	}

	if len(o.Authors) == 0 {
		return wrap("at least 1 author required")
	}
	if len(o.Authors) > 10 {
		return wrap("max 10 authors allowed")
	}
	for _, author := range o.Authors {
		if len(author) > 50 {
			return wrap("author name too long, max 50 chars")
		}
	}

	// Year format: yyyy or yyyy/yyyy
	if !strings.Contains(o.Year, "/") {
		year, err := parseYear(o.Year)
		if err != nil {
			return wrap(err.Error())
		}
		if year < 1980 {
			return wrap("year must be at least 1980")
		}
	} else {
		parts := strings.Split(o.Year, "/")
		if len(parts) != 2 {
			return wrap("invalid year format, must be yyyy or yyyy/yyyy")
		}

		start, err := parseYear(parts[0])
		if err != nil {
			return wrap(err.Error())
		}
		end, err := parseYear(parts[1])
		if err != nil {
			return wrap(err.Error())
		}

		if start < 1980 {
			return wrap("year must be at least 1980")
		}

		if end != start+1 {
			return wrap("years must be consecutive")
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
		return wrap(fmt.Sprintf("invalid testing type - %s", t.TestingT))
	}
	checker := t.Checker != ""
	if (t.TestingT == "checker" && !checker) || (t.TestingT != "checker" && checker) {
		return wrap("checker is required iff testing type is checker")
	}
	interactor := t.Interactor != ""
	if (t.TestingT == "interactor" && !interactor) || (t.TestingT != "interactor" && interactor) {
		return wrap("interactor is required iff testing type is interactor")
	}
	if len(t.Tests) == 0 {
		return wrap("at least 1 test is required")
	}
	if len(t.Tests) > 999 {
		return wrap("max 999 tests allowed")
	}
	if t.MemLimMiB < 40 {
		return wrap("memory limit must be at least 40 MiB")
	}
	if t.MemLimMiB > 2048 {
		return wrap("memory limit must be at most 2048 MiB")
	}
	if t.CpuLimMs < 100 {
		return wrap("cpu time limit must be at least 100 ms")
	}
	if t.CpuLimMs > 8000 {
		return wrap("cpu time limit must be at most 8000 ms")
	}
	if len(t.Checker) > 1e6 {
		return wrap("checker must be at most 1 MB")
	}
	if len(t.Interactor) > 1e6 {
		return wrap("interactor must be at most 1 MB")
	}
	// tests can't weigh more than 500 MB
	totalTestSize := 0
	for _, test := range t.Tests {
		totalTestSize += len(test.Input) + len(test.Answer)
	}
	if totalTestSize > 500*1024*1024 {
		return wrap("tests must be at most 500 MB")
	}
	return nil
}

type Scoring struct {
	ScoringT string      // scoring type. documented in readme.md
	TotalP   int         // total/max points. to verify correct configuration.
	Groups   []TestGroup // can be 1:1 to subtasks. nil if scoringT == "test-sum".
}

type Statement struct {
	MdStatements []MdStatement
	Subtasks     []Subtask
	Examples     []Example
}

type Solutions struct {
	Solutions []Solution // both good & bad. used for constratint calibration.
}

type Subtask struct {
	Desc     i18n[string] // description
	Points   int
	VisInput bool // compatibility with latvian informatics olympiad (LIO)
}

type TestGroup struct {
	Points  int
	Tests   []int
	Public  bool // results visible during contest
	Subtask int  // subtask it belongs to. 0 if nil
}

type Test struct {
	Input  string
	Answer string
}

type Example struct {
	Input  string
	Output string
	MdNote string
}

type OriginalPdf struct {
	Language string
	Content  []byte
}

type MdStatement struct {
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
	Correct  bool   // whether it should receive max points
	Subtasks []int  // subtasks that it should correctly solve
	Content  []byte
}

type Archive struct {
	Files []ArchiveFile // testcase gen scripts, og pdfs, etc.
}

type ArchiveFile struct {
	RelPath string // relative to archive root
	Content []byte
}

func (archive *Archive) GetIllustrImgs() []ArchiveFile {
	prefix := "/reserved/illustr/img."
	imgs := []ArchiveFile{}
	for _, file := range archive.Files {
		if strings.HasPrefix(file.RelPath, prefix) {
			imgs = append(imgs, file)
		}
	}
	return imgs
}

func (archive *Archive) GetOgStatementPdfs() []OriginalPdf {
	prefix := "/reserved/og-pdfs/"
	ext := ".pdf"
	pdfs := []OriginalPdf{}
	for _, file := range archive.Files {
		if strings.HasSuffix(file.RelPath, ext) &&
			strings.HasPrefix(file.RelPath, prefix) {
			pdfs = append(pdfs, OriginalPdf{
				Language: strings.TrimPrefix(file.RelPath, prefix),
				Content:  file.Content,
			})
		}
	}
	return pdfs
}
