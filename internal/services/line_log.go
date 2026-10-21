package services

import "sync"

// LogSize is the number of lines a LineLog keeps.
const LogSize = 200

type LineSource uint8

const (
	SourceCube LineSource = iota
	SourcePico
)

type LogLine struct {
	Source LineSource
	Text   string
}

// LineLog keeps the latest cube and Pico lines for the screen and for desktops that connect later.
// It is safe for concurrent use.
type LineLog struct {
	mutex sync.Mutex
	lines [LogSize]LogLine
	count int
	total uint32 // lines added so far
}

func (this *LineLog) Add(source LineSource, text string) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	this.lines[this.total%LogSize] = LogLine{Source: source, Text: text}
	this.total++
	this.count = min(this.count+1, LogSize)
}

// Total changes whenever a line is added.
func (this *LineLog) Total() uint32 {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	return this.total
}

// Last copies the newest lines into dst, oldest first, and returns how many were copied.
func (this *LineLog) Last(dst []LogLine) int {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	n := min(len(dst), this.count)
	for i := range n {
		dst[i] = this.lines[(this.total-uint32(n-i))%LogSize]
	}
	return n
}
