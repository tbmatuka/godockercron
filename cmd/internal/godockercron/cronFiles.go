package godockercron

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

type cronFile struct {
	Stack    string
	Service  string
	FilePath string
}

type cronFileEntry struct {
	Stack   string
	Service string
	Timing  string
	Command string
}

func getAllCronFileEntries(dir string) []cronFileEntry {
	var cronFileEntries []cronFileEntry

	cronFilePaths := getCronFilePaths(dir)
	for _, cronFilePath := range cronFilePaths {
		cronFileEntries = append(cronFileEntries, getCronFileEntries(cronFilePath)...)
	}

	return cronFileEntries
}

func getCronFilePaths(dir string) []cronFile {
	stacks, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	regex := regexp.MustCompile(`^(?P<service>[^.]+).cron$`)

	var cronFiles []cronFile

	for _, stack := range stacks {
		if !stack.IsDir() {
			continue
		}

		cronDirPath := fmt.Sprintf(`%s/%s/cron`, strings.TrimRight(dir, `/`), stack.Name())

		cronFilePaths, err := os.ReadDir(cronDirPath)
		if err != nil {
			continue
		}

		for _, cronFilePath := range cronFilePaths {
			match := regex.FindStringSubmatch(cronFilePath.Name())

			if match != nil {
				newCronFile := cronFile{
					Stack:    stack.Name(),
					Service:  match[regex.SubexpIndex(`service`)],
					FilePath: fmt.Sprintf(`%s/%s`, cronDirPath, cronFilePath.Name()),
				}
				cronFiles = append(cronFiles, newCronFile)
			}
		}
	}

	return cronFiles
}

func getCronFileEntries(file cronFile) []cronFileEntry {
	var entries []cronFileEntry

	readFile, err := os.Open(file.FilePath)
	defer func(readFile *os.File) {
		_ = readFile.Close()
	}(readFile)

	if err != nil {
		log.Println(err)

		return entries
	}

	scanner := bufio.NewScanner(readFile)
	scanner.Split(bufio.ScanLines)

	// m h  dom mon dow   command
	regex := regexp.MustCompile(`^\s*(?P<m>[^#\s]+)\s+(?P<h>\S+)\s+(?P<dom>\S+)\s+(?P<mon>\S+)\s+(?P<dow>\S+)\s+(?P<command>.+)$`)
	for scanner.Scan() {
		match := regex.FindStringSubmatch(scanner.Text())

		if match != nil {
			entries = append(entries, cronFileEntry{
				Stack:   file.Stack,
				Service: file.Service,
				Timing: fmt.Sprintf(
					`%s %s %s %s %s`,
					match[regex.SubexpIndex(`m`)],
					match[regex.SubexpIndex(`h`)],
					match[regex.SubexpIndex(`dom`)],
					match[regex.SubexpIndex(`mon`)],
					match[regex.SubexpIndex(`dow`)],
				),
				Command: match[regex.SubexpIndex(`command`)],
			})
		}
	}

	return entries
}
