package lio2024_test

import (
	"testing"

	"github.com/programme-lv/taskzip/external/lio/lio2024"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLio2024Yaml(t *testing.T) {
	yamlContent := `name: 'Kp'
title: 'Kvadrātveida putekļsūcējs'
statements:
  - ['./teksts/kp.typ', 'lv']

time_limit: 0.5
memory_limit: 256

subtask_points: [0, 3, 48, 28, 21]

tests_archive: './testi/tests.zip'
checker: './riki/checker.cpp'

tests_groups:
  - groups: 0
    points: 0
    public: true
    subtask: 0
    comment: Piemēri

  - groups: 1
    points: 3
    public: true
    subtask: 1

  - groups: [2, 2]
    points: 8
    public: [2]
    subtask: 2

  - groups: [3, 4]
    points: 10
    public: [4]
    subtask: 2
`

	expected := lio2024.ParsedLio2024Yaml{
		CpuTimeLimitInSeconds:  0.5,
		MemoryLimitInMegabytes: 256,
		TaskShortIDCode:        "Kp",
		FullTaskName:           "Kvadrātveida putekļsūcējs",
		TestZipPathRelToYaml:   "./testi/tests.zip",
		CheckerPathRelToYaml:   &([]string{"./riki/checker.cpp"}[0]),
		// InteractorPathRelToYaml: &([]string{"./riki/interactor.cpp"}[0]),
		SubtaskPoints: []int{0, 3, 48, 28, 21},
		TestGroups: []lio2024.ParsedLio2024YamlTestGroup{
			{
				GroupID: 0,
				Points:  0,
				Public:  true,
				Subtask: 0,
				Comment: &([]string{"Piemēri"}[0]),
			},
			{
				GroupID: 1,
				Points:  3,
				Public:  true,
				Subtask: 1,
			},
			{
				GroupID: 2,
				Points:  8,
				Public:  true,
				Subtask: 2,
			},
			{
				GroupID: 3,
				Points:  10,
				Public:  false,
				Subtask: 2,
			},
			{
				GroupID: 4,
				Points:  10,
				Public:  true,
				Subtask: 2,
			},
		},
	}

	actual, err := lio2024.ParseLio2024Yaml([]byte(yamlContent))
	require.NoErrorf(t, err, "Failed to parse Lio2024 YAML: %v", err)

	assert.Equal(t, expected, actual)
}

func TestParseLio2024YamlTramvaji(t *testing.T) {
	yamlContent := `name: 'Tramvaji'
title: 'Tramvaju vērošana'
statements:
  - ['./teksts/tramvaji.typ', 'lv']

statement_contest:
  description: LIO — Valsts olimpiāde - iesildīšanās diena
  document_titles:
    - LATVIJAS 38. INFORMĀTIKAS OLIMPIĀDE
    - VALSTS OLIMPIĀDES OTRĀ DIENA - 2025. GADA 28. FEBRUĀRIS
    - ABAS VECUMA GRUPAS

time_limit: 0.7
memory_limit: 256

validator: './riki/validator.cpp'
tests_archive: './testi/tests.zip'

# == Nepieciešams ne komunikāciju uzdevumos, 
# kur žūrijas atbilde var nesakrist ar dalībnieka atbildi
# checker: './riki/checker.cpp'

# == Nepieciešams komunikāciju uzdevumos
# task_type: "Communication"
# interactor: './riki/interactor.cpp'

subtask_points: [0, 2, 10, 15, 27, 46]

# subtask 0

tests_groups:
  - groups: 0
    points: 0
    public: true
    subtask: 0
    comment: Piemēri

# subtask 1

  - groups: 1
    points: 2
    public: true
    subtask: 1

# subtask 2

  - groups: [2,6]
    points: 2
    public: [3]
    subtask: 2

# subtask 3

  - groups: [7,11]
    points: 3
    public: [8]
    subtask: 3

# subtask 4

  - groups: [12,20]
    points: 3
    public: [13]
    subtask: 4

# subtask 5

  - groups: [21,43]
    points: 2
    public: [29]
    subtask: 5`

	expected := lio2024.ParsedLio2024Yaml{
		CpuTimeLimitInSeconds:  0.7,
		MemoryLimitInMegabytes: 256,
		TaskShortIDCode:        "Tramvaji",
		FullTaskName:           "Tramvaju vērošana",
		TestZipPathRelToYaml:   "./testi/tests.zip",
		ValidatorPathRelToYaml: &([]string{"./riki/validator.cpp"}[0]),
		SubtaskPoints:          []int{0, 2, 10, 15, 27, 46},
		TestGroups: []lio2024.ParsedLio2024YamlTestGroup{
			{
				GroupID: 0,
				Points:  0,
				Public:  true,
				Subtask: 0,
				Comment: &([]string{"Piemēri"}[0]),
			},
			{
				GroupID: 1,
				Points:  2,
				Public:  true,
				Subtask: 1,
			},
			{
				GroupID: 2,
				Points:  2,
				Public:  false,
				Subtask: 2,
			},
			{
				GroupID: 3,
				Points:  2,
				Public:  true,
				Subtask: 2,
			},
			{
				GroupID: 4,
				Points:  2,
				Public:  false,
				Subtask: 2,
			},
			{
				GroupID: 5,
				Points:  2,
				Public:  false,
				Subtask: 2,
			},
			{
				GroupID: 6,
				Points:  2,
				Public:  false,
				Subtask: 2,
			},
			{
				GroupID: 7,
				Points:  3,
				Public:  false,
				Subtask: 3,
			},
			{
				GroupID: 8,
				Points:  3,
				Public:  true,
				Subtask: 3,
			},
			{
				GroupID: 9,
				Points:  3,
				Public:  false,
				Subtask: 3,
			},
			{
				GroupID: 10,
				Points:  3,
				Public:  false,
				Subtask: 3,
			},
			{
				GroupID: 11,
				Points:  3,
				Public:  false,
				Subtask: 3,
			},
			{
				GroupID: 12,
				Points:  3,
				Public:  false,
				Subtask: 4,
			},
			{
				GroupID: 13,
				Points:  3,
				Public:  true,
				Subtask: 4,
			},
			{
				GroupID: 14,
				Points:  3,
				Public:  false,
				Subtask: 4,
			},
			{
				GroupID: 15,
				Points:  3,
				Public:  false,
				Subtask: 4,
			},
			{
				GroupID: 16,
				Points:  3,
				Public:  false,
				Subtask: 4,
			},
			{
				GroupID: 17,
				Points:  3,
				Public:  false,
				Subtask: 4,
			},
			{
				GroupID: 18,
				Points:  3,
				Public:  false,
				Subtask: 4,
			},
			{
				GroupID: 19,
				Points:  3,
				Public:  false,
				Subtask: 4,
			},
			{
				GroupID: 20,
				Points:  3,
				Public:  false,
				Subtask: 4,
			},
			{
				GroupID: 21,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 22,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 23,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 24,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 25,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 26,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 27,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 28,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 29,
				Points:  2,
				Public:  true,
				Subtask: 5,
			},
			{
				GroupID: 30,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 31,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 32,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 33,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 34,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 35,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 36,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 37,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 38,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 39,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 40,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 41,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 42,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
			{
				GroupID: 43,
				Points:  2,
				Public:  false,
				Subtask: 5,
			},
		},
	}

	actual, err := lio2024.ParseLio2024Yaml([]byte(yamlContent))
	require.NoError(t, err)

	assert.Equal(t, expected, actual)
}
