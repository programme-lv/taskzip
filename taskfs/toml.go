package taskfs

type TaskToml struct {
	Id      string
	Name    map[string]string
	Testing struct {
		Type   string
		CpuMs  int
		MemMiB int
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
		Notes   map[string]string
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
		Description map[string]string
	}
}
