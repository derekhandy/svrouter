//
//												svrouter @ v2.1.1
//
//									MIT License, Copyright (c) 2026 Derek Handy
//							Project can be found at: https://github.com/derekhandy/svrouter
//

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"

	. "github.com/derekhandy/svclilib"
)

var env Environment
var commands []Command
var path string

var scd = []string{
	"lists all system key : commands",
	"shows all user-generated key : commands in dictionary",
	"reads key : command in dictionary",
	"writes key : command to dictionary",
	"removes a key : command from dictionary",
	"opens a user-interface for the program",
	"exits the program",
}

var scu = []string{
	"help, help <command>",
	"list",
	"read <key>",
	"set <key> <command> [args...]",
	"remove <key>",
	"gui",
	"exit",
}

func main() {
	SetEnvironment()
	LoadCommands()
	ClearConsole()
	env.REPL = false

	if len(os.Args) < 2 {
		fmt.Print("\033[999;1H")
		Logh(env)
		EnterInterface()
		return
	}

	ClearConsole()
	ReadInput(os.Args)
}

func ClearConsole() {
	if runtime.GOOS == "windows" {
		Exec := exec.Command("cmd.exe", "/c", "cls")
		Exec.Stdout = os.Stdout
		_ = Exec.Run()
		return
	}

	fmt.Print("\033[H\033[2J\033[3J")
}

func EnterInterface() {
	env.REPL = true

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\033[999;1H")

		print(env.Glyphs[0])

		if !scanner.Scan() {
			if scanner.Err() != nil {
				return
			}

			break
		}

		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		args := strings.Fields(line)

		args = append([]string{"sv"}, args...)

		Logv(env)

		ReadInput(args)

	}
}

func ExecWrapper() {
	reader := bufio.NewScanner(os.Stdin)
	Logm(env, "Wrapper mode: type commands directly. Use 'help' for usage, 'exit' to quit.")

	for {
		fmt.Print(env.Prefix + " ")
		if !reader.Scan() {
			break
		}

		line := strings.TrimSpace(reader.Text())
		if line == "" {
			continue
		}

		if strings.EqualFold(line, "exit") {
			break
		}

		args := strings.Fields(line)
		Execute(env, args)
	}

	if err := reader.Err(); err != nil {
		Logr(env, "ERROR Input error: "+err.Error())
	}
}

func ExitCommand(args []string) (error, string) {
	os.Exit(0)
	return nil, ""
}

func HelpCommand(args []string) (error, string) {
	if len(args) < 1 {
		Logr(env, "Usage: help <command>\n")
		Logr(env, "help, list, read, set, remove, gui, exit")
	} else {
		PrintUsage(args[0])
	}

	return nil, ""
}

func PrintUsage(arg string) {
	idx := slices.IndexFunc(commands, func(c Command) bool {
		return c.Name == arg
	})
	if idx != -1 {
		Logu(env, []Command{env.Commands[idx]})
	} else {
		Logr(env, "ERROR Unknown command: "+arg+"\n")
	}
}

func ReadInput(args []string) {
	if args[1] == "gui" {
		env.REPL = true
		Execute(env, []string{"gui"})
	} else {
		Execute(env, args[1:])
	}
}

func SetEnvironment() {
	commands = []Command{
		{Name: "help", Desc: scd[0], Usage: scu[0], ArgRequired: 0, Function: HelpCommand},
		{Name: "list", Desc: scd[1], Usage: scu[1], ArgRequired: 0, Function: ListCommands},
		{Name: "read", Desc: scd[2], Usage: scu[2], ArgRequired: 1, Function: ReadCommand},
		{Name: "set", Desc: scd[3], Usage: scu[3], ArgRequired: 2, Function: SetCommand},
		{Name: "remove", Desc: scd[4], Usage: scu[4], ArgRequired: 1, Function: RemoveCommand},
		{Name: "gui", Desc: scd[5], Usage: scu[5], ArgRequired: 0, Function: GuiCommand},
		{Name: "exit", Desc: scd[6], Usage: scu[6], ArgRequired: 0, Function: ExitCommand},
	}

	env = Environment{
		Header:      "\n  svrouter v2.1.5\n",
		Prefix:      "  ",
		Footer:      "  -\n\n",
		Glyphs:      []string{"  ❍  ", "  ┊\n", "  ┊   "},
		Commands:    commands,
		Spacing:     "\n\n",
		UsageFormat: "{prefix}┊   NAME	{name}\n{prefix}┊   DESC	{desc}\n{prefix}┊   USAGE	{usage}",
	}

	var err error
	path, err = resolveAppDataPath("commands")
	if err != nil {
		Logr(env, "ERROR Failed to resolve app data path: "+err.Error())
	}
}
