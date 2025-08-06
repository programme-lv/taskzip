package lio

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/programme-lv/task-zip/common/errwrap"
	"github.com/programme-lv/task-zip/common/zips"
)

type LioTest struct {
	TaskName string

	TestGroup         int
	NoInTestGroup     int
	NoInLexFnameOrder int

	Input  []byte
	Answer []byte
}

func ReadLioTestsFromZip(testZipPath string) ([]LioTest, error) {
	// create a tmp directory where to unzip the test zip
	tmpDirPath, err := os.MkdirTemp("", "lio-tests")
	if err != nil {
		msg := "create tmp directory"
		return nil, errwrap.Unexpected(msg, err)
	}
	defer os.RemoveAll(tmpDirPath)

	err = zips.Unzip(testZipPath, tmpDirPath)
	if err != nil {
		msg := fmt.Sprintf("unzip %s", testZipPath)
		return nil, errwrap.Unexpected(msg, err)
	}

	return ReadLioTestsFromDir(tmpDirPath)
}

func ReadLioTestsFromDir(testDir string) ([]LioTest, error) {
	res := []LioTest{}

	listDir, err := os.ReadDir(testDir)
	if err != nil {
		msg := fmt.Sprintf("read directory %s", testDir)
		return nil, errwrap.Unexpected(msg, err)
	}

	// sort by filename in lexicographical order
	sort.Slice(listDir, func(i, j int) bool {
		return listDir[i].Name() < listDir[j].Name()
	})

	if len(listDir)%2 != 0 {
		msg := fmt.Sprintf("unexpected number of files in the directory: %d", len(listDir))
		return nil, errwrap.Unexpected(msg, nil)
	}

	inputEntries := listDir[:len(listDir)/2]
	answerEntries := listDir[len(listDir)/2:]

	for i := 0; i < len(inputEntries); i++ {
		inputPath := filepath.Join(testDir, inputEntries[i].Name())
		answerPath := filepath.Join(testDir, answerEntries[i].Name())

		inFname := filepath.Base(inputPath)
		ansFname := filepath.Base(answerPath)

		inFnameSplit, err := lioTestName(inFname)
		if err != nil {
			msg := fmt.Sprintf("parse input filename %s", inFname)
			return nil, errwrap.Unexpected(msg, err)
		}
		ansFnameSplit, err := lioTestName(ansFname)
		if err != nil {
			msg := fmt.Sprintf("parse answer filename %s", ansFname)
			return nil, errwrap.Unexpected(msg, err)
		}

		inTaskName := inFnameSplit[0]
		ansTaskName := ansFnameSplit[0]

		if inTaskName != ansTaskName {
			msg := fmt.Sprintf("input and answer task names do not match: %s, %s", inTaskName, ansTaskName)
			return nil, errwrap.Unexpected(msg, nil)
		}

		if inFnameSplit[1] != "i" || ansFnameSplit[1] != "o" {
			msg := fmt.Sprintf("unexpected filename format: %s, %s", inFname, ansFname)
			return nil, errwrap.Unexpected(msg, nil)
		}

		inGroup, err := strconv.Atoi(inFnameSplit[2])
		if err != nil {
			msg := fmt.Sprintf("convert %s to int", inFnameSplit[2])
			return nil, errwrap.Unexpected(msg, err)
		}
		ansGroup, err := strconv.Atoi(ansFnameSplit[2])
		if err != nil {
			msg := fmt.Sprintf("convert %s to int", ansFnameSplit[2])
			return nil, errwrap.Unexpected(msg, err)
		}

		if inGroup != ansGroup {
			msg := fmt.Sprintf("input and answer groups do not match: %d, %d", inGroup, ansGroup)
			return nil, errwrap.Unexpected(msg, nil)
		}

		inGroupNo := 1
		if len(inFnameSplit) == 4 {
			if len(inFnameSplit[3]) != 1 {
				msg := fmt.Sprintf("unexpected filename format: %s", inFname)
				return nil, errwrap.Unexpected(msg, nil)
			}
			inGroupNo = int(inFnameSplit[3][0]) - int('a') + 1
		}

		ansGroupNo := 1
		if len(ansFnameSplit) == 4 {
			if len(ansFnameSplit[3]) != 1 {
				msg := fmt.Sprintf("unexpected filename format: %s", ansFname)
				return nil, errwrap.Unexpected(msg, nil)
			}
			ansGroupNo = int(ansFnameSplit[3][0]) - int('a') + 1
		}

		if inGroupNo != ansGroupNo {
			msg := fmt.Sprintf("input and answer groups do not match: %d, %d", inGroupNo, ansGroupNo)
			return nil, errwrap.Unexpected(msg, nil)
		}

		inBytes, err := os.ReadFile(inputPath)
		if err != nil {
			msg := fmt.Sprintf("read input file %s", inputPath)
			return nil, errwrap.Unexpected(msg, err)
		}
		ansBytes, err := os.ReadFile(answerPath)
		if err != nil {
			msg := fmt.Sprintf("read answer file %s", answerPath)
			return nil, errwrap.Unexpected(msg, err)
		}

		res = append(res, LioTest{
			TaskName:          inTaskName,
			TestGroup:         inGroup,
			NoInTestGroup:     inGroupNo,
			NoInLexFnameOrder: i,
			Input:             inBytes,
			Answer:            ansBytes,
		})
	}

	return res, nil
}

/*
kp.i00 -> ["kp", "i", "00"]
kp.i01a -> ["kp", "i", "01", "a"]
kp.i01b
kp.o00
kp.o01a
kp.o01b
*/
func lioTestName(fname string) ([]string, error) {
	res := []string{}

	splitByDot := strings.Split(fname, ".")
	if len(splitByDot) != 2 {
		msg := fmt.Sprintf("unexpected filename: %s", fname)
		return nil, errwrap.Unexpected(msg, nil)
	}
	res = append(res, splitByDot[0])

	ext := splitByDot[1]
	if ext[0] != 'i' && ext[0] != 'o' {
		msg := fmt.Sprintf("unexpected second part: %s", ext)
		return nil, errwrap.Unexpected(msg, nil)
	}

	res = append(res, ext[:1])

	hasLetter := false
	for i := 1; i < len(ext); i++ {
		if !(ext[i] >= '0' && ext[i] <= '9') {
			res = append(res, ext[1:i])
			res = append(res, ext[i:])
			hasLetter = true
			break
		}
	}
	if !hasLetter {
		res = append(res, ext[1:])
	}

	if len(res) != 3 && len(res) != 4 {
		return nil, fmt.Errorf("unexpected number of parts: %d", len(res))
	}

	return res, nil
}
