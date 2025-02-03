package vtwo

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type chunk struct {
	date    time.Time
	title   string
	content string
}

func (app *VTwo) TrackFiles() {
	// for now all that we want to do is to get through all relevant notes, break them down into chunks, and print them
	// we're also ignoring non daily notes for now

	dailyNotesPath := filepath.Join(app.rootDir, "Daily")
	dailyNotes, err := os.ReadDir(dailyNotesPath)
	if err != nil {
		log.Fatal(err)
	}

	for _, note := range dailyNotes {
		notePath := filepath.Join(dailyNotesPath, note.Name())
		noteContent, err := os.ReadFile(notePath)
		if err != nil {
			log.Fatal(err)
		}

		dateString := strings.Replace(note.Name(), ".md", "", 1)
		date, err := time.Parse("2006-01-02", dateString)
		if err != nil {
			log.Fatal(err)
		}

		chunks := breakDownNote(string(noteContent), date)
		for _, chunk := range chunks {
			headerLine := fmt.Sprintf("------- %s: %s -------\n", chunk.date.Format("2006-01-02"), chunk.title)
			footerLine := strings.Repeat("-", len(headerLine))
			fmt.Println(headerLine)
			fmt.Println(chunk.content)
			fmt.Println(footerLine)
		}
	}

}

func breakDownNote(noteContent string, date time.Time) []chunk {
	// we can assume that noteContent is a valid markdown file, with multiple h3 headers
	// indicated by three hashes separating each chunk

	chunks := []chunk{}

	lines := strings.Split(noteContent, "\n")

	startLineNum := -1
	currentTitle := ""
	for lineNum, line := range lines {
		if strings.HasPrefix(line, "### ") || lineNum == len(lines)-1 {
			// make sure we guard against the very first header
			if startLineNum >= 0 {
				endLineNum := lineNum - 1
				chunks = append(chunks, chunk{
					date:    date,
					title:   currentTitle,
					content: strings.TrimSpace(strings.Join(lines[startLineNum:endLineNum], "\n")),
				})
			}

			if lineNum < len(lines)-1 {
				startLineNum = lineNum + 1
				currentTitle = strings.TrimSpace(line[4:]) // since we got here from a HasPrefix check we know it has the ignorable 4 preffixes
			}
		}
	}

	return chunks
}
