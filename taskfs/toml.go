package taskfs

type TaskToml struct {
	Id      string
	Name    map[string]string
	Testing struct {
		Type   string
		CpuMs  int `toml:"cpu_ms"`
		MemMiB int `toml:"mem_mib"`
	}
	Scoring struct {
		Type  string
		Total int
	}
	Origin struct {
		Olymp   string
		Year    string
		Stage   string
		Org     string
		Authors []string
		Notes   map[string]string `toml:"notes"`
	}
	Metadata struct {
		Tags       []string
		Difficulty int
	}
	Solutions []struct {
		Fname    string
		Subtasks []int
	}
	Subtasks []struct {
		Points      int
		Description map[string]string `toml:"description"`
	}
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
	t.Solutions = make([]struct {
		Fname    string
		Subtasks []int
	}, len(solutions))
	for i, sol := range solutions {
		t.Solutions[i].Fname = sol.Fname
		t.Solutions[i].Subtasks = sol.Subtasks
	}
}

func (t *TaskToml) SetSubtasks(subtasks []Subtask) {
	t.Subtasks = make([]struct {
		Points      int
		Description map[string]string `toml:"description"`
	}, len(subtasks))
	for i, subtask := range subtasks {
		t.Subtasks[i].Points = subtask.Points
		t.Subtasks[i].Description = subtask.Desc
	}
}
