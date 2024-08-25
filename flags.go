package main

import (
	"github.com/gnailuy/sudoku/generator"
	"github.com/spf13/pflag"
	"github.com/thediveo/enumflag/v2"
)

// Define the Level enum type and identifiers.
type Level int

const (
	Easy Level = iota
	Medium
	Hard
	Extreme
	Evil
)

var LevelIdentities = map[Level][]string{
	Easy:    {"easy"},
	Medium:  {"medium"},
	Hard:    {"hard"},
	Extreme: {"extreme"},
	Evil:    {"evil"},
}

// Define the command line options struct.
type CommandLineOptions struct {
	Input *string
	Level *enumflag.EnumFlagValue[Level]
}

// Constructor like function to create a new CommandLineOptions struct.
func NewCommandLineOptions() CommandLineOptions {
	return CommandLineOptions{
		Input: nil,
		Level: new(enumflag.EnumFlagValue[Level]),
	}
}

// Function to parse the command line flags.
func (options *CommandLineOptions) Parse() {
	// Accept an optional argument to play a specific game given by an input string.
	// If the argument is not provided, a random game will be generated.
	options.Input = pflag.StringP("input", "i", "", "A Sudoku problem string to play.")

	// Accept an optional argument to specify the difficulty level of the generated problem.
	defaultLevel := Hard
	options.Level = enumflag.New(&defaultLevel, "level", LevelIdentities, enumflag.EnumCaseInsensitive)
	pflag.VarP(options.Level, "level", "l", "The target difficulty level in easy, medium, hard, extreme, or evil.")

	pflag.Parse()
}

// Function to create the difficulty options based on the command line flags.
func (options *CommandLineOptions) GetDifficultyOptions() generator.SudokuDifficulty {
	currentLevel := options.Level.Get()
	switch currentLevel {
	case Easy:
		return generator.NewEasySudokuDifficulty()
	case Medium:
		return generator.NewMediumSudokuDifficulty()
	case Hard:
		return generator.NewHardSudokuDifficulty()
	case Extreme:
		return generator.NewExtremeSudokuDifficulty()
	case Evil:
		return generator.NewEvilSudokuDifficulty()
	default:
		return generator.NewHardSudokuDifficulty()
	}
}
