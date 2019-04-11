package manager

import "time"

type InputArgs struct {
	GoRoutines       uint
	MaxGuesses       uint64
	Debug            bool
	Port             string
	RuleName         string
	HashFile         string
	HashcatMode      string
	TerminalsQueSize int
	ChunkStartSize   uint64
	ChunkDuration    time.Duration
}
