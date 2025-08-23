package taskfs

type TaskToml struct {
	Id        string             `toml:"id"`
	Name      map[string]string  `toml:"name"`
	Testing   TaskTomlTesting    `toml:"testing"`
	Scoring   TaskTomlScoring    `toml:"scoring"`
	Origin    TaskTomlOrigin     `toml:"origin"`
	Metadata  TaskTomlMetadata   `toml:"metadata"`
	Solutions []TaskTomlSolution `toml:"solutions,omitempty"`
	Subtasks  []TaskTomlSubtask  `toml:"subtasks,omitempty"`
}

type TaskTomlTesting struct {
	Type   string `toml:"type"`
	CpuMs  int    `toml:"cpu_ms"`
	MemMiB int    `toml:"mem_mib"`
}

type TaskTomlScoring struct {
	Type  string `toml:"type"`
	Total int    `toml:"total"`
}

type TaskTomlOrigin struct {
	Olymp   string            `toml:"olymp"`
	Year    string            `toml:"year"`
	Stage   string            `toml:"stage"`
	Org     string            `toml:"org"`
	Authors []string          `toml:"authors"`
	Notes   map[string]string `toml:"notes,inline"`
}

type TaskTomlMetadata struct {
	Tags       []string `toml:"tags"`
	Difficulty int      `toml:"difficulty"`
}

type TaskTomlSolution struct {
	Fname    string `toml:"fname"`
	Subtasks []int  `toml:"subtasks"`
}

type TaskTomlSubtask struct {
	Points      int               `toml:"points"`
	Description map[string]string `toml:"description,inline"`
	VisInput    bool              `toml:"vis_input"`
}

func NewTaskToml(t *Task) TaskToml {
	taskToml := TaskToml{
		Id:   t.ShortID,
		Name: t.FullName,
	}
	taskToml.SetTesting(t.Testing)
	taskToml.SetScoring(t.Scoring)
	taskToml.SetOrigin(t.Origin)
	taskToml.SetMetadata(t.Metadata)
	taskToml.SetSolutions(t.Solutions)
	taskToml.SetSubtasks(t.Statement.Subtasks)
	return taskToml
}

func (t *TaskToml) SetTesting(testing Testing) {
	t.Testing.Type = testing.TestingT
	t.Testing.CpuMs = testing.CpuLimMs
	t.Testing.MemMiB = testing.MemLimMiB
}

func (t *TaskToml) SetScoring(scoring Scoring) {
	t.Scoring.Type = scoring.ScoringT
	t.Scoring.Total = scoring.TotalP
}

func (t *TaskToml) SetOrigin(origin Origin) {
	t.Origin.Olymp = origin.Olympiad
	t.Origin.Year = origin.Year
	t.Origin.Stage = origin.OlyStage
	t.Origin.Org = origin.Org
	t.Origin.Authors = origin.Authors
	t.Origin.Notes = origin.Notes
}

func (t *TaskToml) SetMetadata(metadata Metadata) {
	t.Metadata.Tags = metadata.ProblemTags
	t.Metadata.Difficulty = metadata.Difficulty
}

func (t *TaskToml) SetSolutions(solutions []Solution) {
	t.Solutions = make([]TaskTomlSolution, len(solutions))
	for i, sol := range solutions {
		t.Solutions[i].Fname = sol.Fname
		t.Solutions[i].Subtasks = sol.Subtasks
	}
}

func (t *TaskToml) SetSubtasks(subtasks []Subtask) {
	t.Subtasks = make([]TaskTomlSubtask, len(subtasks))
	for i, subtask := range subtasks {
		t.Subtasks[i].Points = subtask.Points
		t.Subtasks[i].Description = subtask.Desc
		t.Subtasks[i].VisInput = subtask.VisInput
	}
}
