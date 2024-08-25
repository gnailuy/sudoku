package cli

import (
	"fmt"

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

var defaultLevel = Hard
var levelIdentities = map[Level][]string{
	Easy:    {"easy"},
	Medium:  {"medium"},
	Hard:    {"hard"},
	Extreme: {"extreme"},
	Evil:    {"evil"},
}

// Define the command line options struct.
type CommandLineOptions struct {
	Input         *string
	Level         *enumflag.EnumFlagValue[Level]
	HelpRequested *bool
}

// Constructor like function to create a new CommandLineOptions struct.
func NewCommandLineOptions() CommandLineOptions {
	return CommandLineOptions{
		Input:         nil,
		Level:         new(enumflag.EnumFlagValue[Level]),
		HelpRequested: new(bool),
	}
}

// Function to parse the command line flags.
func (options *CommandLineOptions) Parse() {
	// Accept an optional argument to play a specific game given by an input string.
	// If the argument is not provided, a random game will be generated.
	options.Input = pflag.StringP("input", "i", "", "Specify a Sudoku problem string to play. If not provided, a random game will be generated.")

	// Accept an optional argument to specify the difficulty level of the generated problem.
	options.Level = enumflag.New(&defaultLevel, "level", levelIdentities, enumflag.EnumCaseInsensitive)
	pflag.VarP(options.Level, "level", "l", "Select the difficulty level for a new game. Options include: easy, medium, hard, extreme, evil.")

	// Define the help message.
	options.HelpRequested = pflag.BoolP("help", "h", false, "Show this help message.")

	pflag.Parse()
}

// Function to print the help message.
func PrintHelp() {
	fmt.Println("Usage: sudoku [options]")
	pflag.PrintDefaults()
}

// Function to create the difficulty options based on the command line flags.
func (options *CommandLineOptions) GetDifficultyOptions() generator.SudokuDifficulty {
	level := options.Level.Get()
	switch level {
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
