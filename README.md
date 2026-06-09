# svrouter


<b> svrouter </b> A command dictionary and execution manager tool for creating custom console commands with shortcuts. Cross-platform support.

Requires <b> Go 1.26 </b> or newer.


```Bash
### Usage Examples

# Fire-and-forget use

$ sv set hi echo Hello World.
$ sv hi
Hello World.


# In REPL Interface

❍ set p git push origin
┊ Key: p
┊ Command: git push origin
❍ p main
┊ Everything up-to-date
```

## Install

Download from repository then build and move the executable to environment path:

```bash
go get github.com/derekhandy/svrouter
go mod tidy
```

```bash
# Linux
go build -o sv
sudo mv sv /usr/local/bin/sv

# Mac OS
go build -o sv
sudo mv sv /usr/local/bin/sv

# Windows PowerShell
go build -o sv.exe
```

*The 'Fyne.io' dependency's compile time can be 5+ minutes long on first build.
It's normal for the first 'go run .' or 'go build' to stall for longer than expected.*

*GUI builds depend on Fyne's native desktop requirements. Build on the target operating system, or use a configured Fyne cross-compilation toolchain for Windows and Mac OS release binaries.*

## Use

There are three supported ways to interact with the app: fire-and-forget, console REPL, and GUI.

```bash
# Runs a REPL interface in your console
sv

# Opens up a fyne.io window interface
sv gui

# Fire-and-forget, runs command then exits
sv help
```

## Commands

```bash
# Set
$ sv set build go build -o sv
┊ Key: build
┊ Command: go build -o sv

# Read
$ sv read cd-proj
┊ Key: cd-proj
┊ Command: cd ~/Projects

# Remove
$ sv remove p
┊ Removed: p

# Call
$ sv build
$ sv cd-proj -n # Flag [-n] runs in new window
```

## Info

Setting commands is generally safer in the GUI since parsing syntax doesn't face the same commandline issues.
      
      - operators: [ && ] [ ┊ ] [ ; ]
      
      - language symbols: [ ' ] [ ` ] [ " ]
      
      - variables: [$T] [$HOME] [export PATH=""]

<br>

Currently, using in-line variables is unstable and not recommended.
<br>
If necessary, building on to the existing framework to allow calling variables from commands pairs i.e. 'git tag $VERSION' would be fairly easy.

*Not all commands are available yet, however, later updates will provide parsing for more complex syntax*

## Examples

```bash
# Basic fire-and-forget usage

$ sv set ga git add .
┊ Key: ga
┊ Command: git add .
┊
$ sv ga


# REPL Interface usage
# Usable only through [-n] flag

❍ set docs cd ~/Documents
┊ Key: docs
┊ Command: cd ~/Documents
┊
❍ docs -n


# Setting operator commands (&&) is possible in CLI with ' ' but GUI is recommended.

❍ set build 'go build -o sv && ./sv'
┊
┊ Key: build
┊ Command: go build -o sv && ./sv
┊
❍ build


# Flag [-n] calls each one of these together in a new console

❍ set startallnow 'firefox spotify.com ┊ dolphin ┊ rider ┊ discord ┊ spotify'
┊
┊ Key: startallnow
┊ Command: firefox spotify.com ┊ dolphin ┊ rider ┊ discord ┊ spotify
┊
❍ startallnow [-n]


# 'sv p v1.4.8' runs the command 'git push origin v1.4.8'

❍ set p 'git push origin'
┊
┊ Key: p
┊ Command: git push origin
┊
❍ p v1.4.8

```

## NOTICE
<b> There are no safeguard implementations preventing unsafe command declarations and execution. The security of activity relies solely on the operating system in-use.
It is not recommended to use powerful commands such as: sudo, chmod, rm, rm -rf, mv, mount, pacman, git, export, or any other related irreversible commands as shortcuts. Extensive caution must be assumed when setting or using commands, particularly if combined with operators or variables.

Additionally, recursion through commands fields using sv or && can occur, and requires extra attention when using.</b>
