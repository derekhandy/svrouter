package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	. "github.com/derekhandy/svclilib"
	. "github.com/derekhandy/svguilib"
)

var output string

type savedCommand struct {
	key   string
	value string
}

func appendArgsToCommand(base string, extra []string) string {
	if len(extra) == 0 {
		return base
	}

	builder := strings.Builder{}
	builder.WriteString(base)
	for _, arg := range extra {
		builder.WriteString(" ")
		builder.WriteString(quoteShellArg(arg))
	}
	return builder.String()
}

func buildCommandLine(args []string) string {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, quoteShellArg(arg))
	}
	return strings.Join(parts, " ")
}

func commandLineFromCmd(cmd *exec.Cmd) string {
	if runtime.GOOS == "windows" && len(cmd.Args) >= 3 && strings.EqualFold(filepath.Base(cmd.Args[0]), "cmd.exe") && strings.EqualFold(cmd.Args[1], "/C") {
		return cmd.Args[2]
	}

	if runtime.GOOS != "windows" && len(cmd.Args) >= 3 && cmd.Args[1] == "-c" {
		return cmd.Args[2]
	}

	return buildCommandLine(cmd.Args)
}

func defaultUnixShell() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	return "/bin/sh"
}

func execCommand(key string, args []string) (error, string) {
	newWindow, filteredArgs := parseLaunchOptions(args)

	cmd, err := parseCommand(key, filteredArgs)
	if err != nil {
		return err, ""
	}

	err, str := executeCommand(cmd, newWindow)

	return err, str
}

func executeCommand(cmd *exec.Cmd, newWindow bool) (error, string) {
	cmdStr := commandLineFromCmd(cmd)

	if !newWindow {
		var stdout, stderr bytes.Buffer
		streamer := newFormattedOutputStreamer(env)
		cmd.Stdout = io.MultiWriter(&stdout, streamer)
		cmd.Stderr = io.MultiWriter(&stderr, streamer)

		err := cmd.Run()
		streamer.Flush()
		if err != nil {
			if isExplorerCommand(cmd, stderr.String()) {
				return nil, ""
			}

			return fmt.Errorf("Failed to run command: %s", err.Error()), ""
		}

		return nil, ""
	}

	startCmd, err := newWindowCommand(cmdStr)
	if err != nil {
		return err, ""
	}
	if err := startCmd.Start(); err != nil {
		return fmt.Errorf("Failed to start command in new window: %w", err), ""
	}

	return nil, ""
}

func GuiCommand(args []string) (error, string) {

	components := IComponents{
		Buttons: []IButton{
			{Text: "Run", Command: func() {}},
			{Text: "List", Command: func() {}},
			{Text: "Read", Command: func() {}},
			{Text: "Set", Command: func() {}},
			{Text: "Remove", Command: func() {}},
		},
		Labels: []ILabel{
			{Text: "svrouter v2.1.5", Bold: true},
		},
		Entries: []IInputField{
			{Label: "Key"},
			{Label: "Command"},
		},
	}

	gui := GUI{
		Name:       "svrouter",
		Components: components,
		Data:       IData{},
		Options: IOptions{
			Size:       IVector2{X: 0, Y: 0},
			Resizeable: true,
			Order:      []string{"labels", "entries", "buttons"},
			ShowTitles: false,
		},
	}

	gui.Components.Buttons[0].Command = func() {
		key := gui.GetBoundData(0)
		Execute(env, []string{key, "-n"})
		gui.Components.Labels[0].Object.SetText(output)
	}

	gui.Components.Buttons[1].Command = func() {
		ListCommands([]string{})
		gui.Components.Labels[0].Object.SetText(output)
	}

	gui.Components.Buttons[2].Command = func() {
		key := gui.GetBoundData(0)
		ReadCommand([]string{key})
		gui.Components.Labels[0].Object.SetText(output)
	}

	gui.Components.Buttons[3].Command = func() {
		key := gui.GetBoundData(0)
		cmd := gui.GetBoundData(1)
		SetCommand([]string{key, cmd})
		gui.Components.Labels[0].Object.SetText(output)
		gui.Components.Entries[0].Object.SetText("")
		gui.Components.Entries[1].Object.SetText("")
	}

	gui.Components.Buttons[4].Command = func() {
		RemoveCommand([]string{gui.GetBoundData(0)})
		gui.Components.Labels[0].Object.SetText(output)
		gui.Components.Entries[0].Object.SetText("")
		gui.Components.Entries[1].Object.SetText("")
	}

	gui.StartGUI()
	return nil, ""
}

func isCommandFile(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".sv")
}

func isExplorerCommand(cmd *exec.Cmd, stderrOutput string) bool {
	if runtime.GOOS != "windows" {
		return false
	}

	base := strings.ToLower(filepath.Base(cmd.Path))
	if base != "explorer.exe" && base != "explorer" {
		return false
	}

	if strings.TrimSpace(stderrOutput) != "" {
		return false
	}

	return true
}

func isReservedCommand(key string) bool {
	switch strings.TrimSpace(key) {
	case "help", "list", "read", "set", "gui", "remove", "exit":
		return true
	default:
		return false
	}
}

func validateCommandKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("Command key is empty")
	}
	if isReservedCommand(key) {
		return fmt.Errorf("Key %q is a built-in command and cannot be overwritten", key)
	}
	if key != filepath.Base(key) || strings.ContainsAny(key, `/\`) {
		return errors.New("Command key cannot contain path separators")
	}

	return nil
}

func keepTerminalOpenCommand(commandLine string, shell string) string {
	return commandLine + "; exec " + quoteUnixArg(shell)
}

func linuxTerminalCommand(commandLine string) (*exec.Cmd, error) {
	shell := defaultUnixShell()
	terminalCommandLine := keepTerminalOpenCommand(commandLine, shell)
	candidates := []struct {
		name string
		args []string
	}{
		{"x-terminal-emulator", []string{"-e", shell, "-lc", terminalCommandLine}},
		{"gnome-terminal", []string{"--", shell, "-lc", terminalCommandLine}},
		{"konsole", []string{"-e", shell, "-lc", terminalCommandLine}},
		{"xfce4-terminal", []string{"-e", shell + " -lc " + quoteUnixArg(terminalCommandLine)}},
		{"xterm", []string{"-e", shell, "-lc", terminalCommandLine}},
	}

	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate.name); err == nil {
			return exec.Command(path, candidate.args...), nil
		}
	}

	return nil, errors.New("No supported terminal emulator found; omit -n to run in the current terminal")
}

func ListCommands(args []string) (error, string) {
	files, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("Failed to list commands: %w", err), ""
	}

	if len(files) == 0 {
		Logr(env, "No commands have been created yet.")
		return nil, ""
	}

	allKeys := ""
	for _, file := range files {
		if file.IsDir() || !isCommandFile(file.Name()) {
			continue
		}

		command, err := readCommandFile(file.Name())
		if err != nil {
			continue
		}
		if isReservedCommand(command.key) {
			continue
		}

		allKeys += ", " + command.key
	}

	if allKeys == "" {
		Logr(env, "No commands have been created yet.")
		return nil, ""
	}

	Logr(env, "Usage: read <command>\n")
	Logr(env, allKeys[2:])
	output = "Found:\n'" + allKeys[2:] + "'"
	return nil, ""
}

func LoadCommands() {
	files, err := os.ReadDir(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if mkdirErr := os.MkdirAll(path, 0755); mkdirErr != nil {
			}
			return
		}
		return
	}

	for _, file := range files {
		if file.IsDir() || !isCommandFile(file.Name()) {
			continue
		}

		command, err := readCommandFile(file.Name())
		if err != nil {
			continue
		}

		if isReservedCommand(command.key) {
			continue
		}
		registerCommand(command.key)
	}
}

func newWindowCommand(commandLine string) (*exec.Cmd, error) {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("cmd.exe", "/c", "start", "", "cmd.exe", "/k", commandLine), nil
	case "darwin":
		script := fmt.Sprintf(`tell application "Terminal" to do script %s`, quoteAppleScriptString(commandLine))
		return exec.Command("osascript", "-e", script), nil
	default:
		return linuxTerminalCommand(commandLine)
	}
}

func parseCommand(key string, args []string) (cmd *exec.Cmd, err error) {
	if err := validateCommandKey(key); err != nil {
		return nil, err
	}

	command, err := readCommandFile(key + ".sv")
	if err != nil {
		return nil, errors.New("Error reading command")
	}

	if command.value == "" {
		return nil, errors.New("Command is empty")
	}

	return shellCommand(appendArgsToCommand(command.value, args)), nil
}

func parseLaunchOptions(args []string) (bool, []string) {
	newWindow := false
	filteredArgs := make([]string, 0, len(args))

	for _, arg := range args {
		if arg == "-n" {
			newWindow = true
			continue
		}

		filteredArgs = append(filteredArgs, arg)
	}

	return newWindow, filteredArgs
}

func quoteAppleScriptString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

func quoteShellArg(arg string) string {
	if runtime.GOOS == "windows" {
		return quoteWindowsArg(arg)
	}
	return quoteUnixArg(arg)
}

func quoteUnixArg(arg string) string {
	if arg == "" {
		return "''"
	}

	if strings.IndexFunc(arg, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\'' || r == '"' || r == '\\' ||
			r == '$' || r == '`' || r == '!' || r == '&' || r == '|' || r == ';' ||
			r == '<' || r == '>' || r == '(' || r == ')' || r == '[' || r == ']' ||
			r == '{' || r == '}' || r == '*' || r == '?' || r == '~' || r == '#'
	}) == -1 {
		return arg
	}

	return "'" + strings.ReplaceAll(arg, "'", "'\"'\"'") + "'"
}

func quoteWindowsArg(arg string) string {
	if arg == "" {
		return `""`
	}

	needsQuotes := strings.IndexFunc(arg, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '"'
	}) != -1

	if !needsQuotes {
		return arg
	}

	var builder strings.Builder
	builder.WriteByte('"')

	backslashes := 0
	for i := 0; i < len(arg); i++ {
		c := arg[i]
		if c == '\\' {
			backslashes++
			continue
		}

		if c == '"' {
			builder.WriteString(strings.Repeat(`\`, backslashes*2+1))
			builder.WriteByte('"')
			backslashes = 0
			continue
		}

		if backslashes > 0 {
			builder.WriteString(strings.Repeat(`\`, backslashes))
			backslashes = 0
		}

		builder.WriteByte(c)
	}

	if backslashes > 0 {
		builder.WriteString(strings.Repeat(`\`, backslashes*2))
	}

	builder.WriteByte('"')
	return builder.String()
}

func readCommandFile(name string) (savedCommand, error) {
	data, err := os.ReadFile(filepath.Join(path, name))
	if err != nil {
		return savedCommand{}, err
	}

	parts := strings.SplitN(string(data), ",", 2)
	if len(parts) < 2 {
		return savedCommand{}, errors.New("Invalid command format in file")
	}

	command := savedCommand{
		key:   strings.TrimSpace(parts[0]),
		value: strings.TrimSpace(parts[1]),
	}
	if command.key == "" {
		return savedCommand{}, errors.New("Command key is empty")
	}

	return command, nil
}

func ReadCommand(args []string) (error, string) {
	if len(args) < 1 {
		return errors.New("Read command requires at least 1 argument"), ""
	}
	if err := validateCommandKey(args[0]); err != nil {
		return err, ""
	}

	command, err := readCommandFile(args[0] + ".sv")
	if err != nil {
		return fmt.Errorf("Failed to read command: %w", err), ""
	}

	Logr(env, "Key: "+command.key+"\n")
	Logr(env, "Command: "+command.value+"\n")
	output = "Read:\n'" + command.key + "', '" + command.value + "'"
	return nil, ""
}

func registerCommand(key string) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}

	keyCopy := key
	command := Command{
		Name:        key,
		ArgRequired: 0,
		Function: func(args []string) (error, string) {
			return execCommand(keyCopy, args)
		},
	}

	for index, existing := range commands {
		if existing.Name == key {
			commands[index] = command
			env.Commands = commands
			return
		}
	}

	commands = append(commands, command)
	env.Commands = commands
}

func RemoveCommand(args []string) (error, string) {
	if len(args) < 1 {
		return errors.New("Remove command requires at least 1 argument"), ""
	}
	if err := validateCommandKey(args[0]); err != nil {
		return err, ""
	}

	filePath := filepath.Join(path, args[0]+".sv")
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("Failed to remove file: %w", err), ""
	}

	unregisterCommand(args[0])
	Logr(env, "Removed: "+args[0])
	output = "Removed:\n'" + args[0] + "'"
	return nil, ""
}

func SetCommand(args []string) (error, string) {
	if len(args) < 2 {
		return errors.New("'Set' command requires at least 2 arguments"), ""
	}

	key := args[0]
	if err := validateCommandKey(key); err != nil {
		return err, ""
	}

	value := strings.Join(args[1:], " ")
	fileString := key + "," + value

	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("Failed to create directory: %w", err), ""
	}

	filePath := filepath.Join(path, key+".sv")
	if err := os.WriteFile(filePath, []byte(fileString), 0644); err != nil {
		return fmt.Errorf("Failed to write file: %w", err), ""
	}

	registerCommand(key)
	Logr(env, "Key: "+key+"\n")
	Logr(env, "Command: "+value+"\n")
	output = "Set:\n'" + key + "', '" + value + "'"
	return nil, ""
}

func shellCommand(commandLine string) *exec.Cmd {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("cmd.exe", "/C", commandLine)
	default:
		return exec.Command(defaultUnixShell(), "-c", commandLine)
	}
}

func unregisterCommand(key string) {
	for index, existing := range commands {
		if existing.Name != key {
			continue
		}

		commands = append(commands[:index], commands[index+1:]...)
		env.Commands = commands
		return
	}
}

type formattedOutputStreamer struct {
	env    Environment
	mu     sync.Mutex
	buffer strings.Builder
}

func newFormattedOutputStreamer(env Environment) *formattedOutputStreamer {
	return &formattedOutputStreamer{env: env}
}

func (streamer *formattedOutputStreamer) Write(data []byte) (int, error) {
	streamer.mu.Lock()
	defer streamer.mu.Unlock()

	for _, char := range string(data) {
		switch char {
		case '\r':
			streamer.flushLocked()
		case '\n':
			streamer.flushLocked()
		default:
			streamer.buffer.WriteRune(char)
		}
	}

	return len(data), nil
}

func (streamer *formattedOutputStreamer) Flush() {
	streamer.mu.Lock()
	defer streamer.mu.Unlock()

	streamer.flushLocked()
}

func (streamer *formattedOutputStreamer) flushLocked() {
	line := streamer.buffer.String()
	streamer.buffer.Reset()
	if line == "" {
		return
	}

	Logr(streamer.env, line)
}
