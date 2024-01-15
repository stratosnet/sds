package console

/*
 terminal console
type "exit" to exit console
*/

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/peterh/liner"

	"github.com/stratosnet/sds/framework/utils/cmd"
)

var (
	cmdNames [100]string
	cmdPos   int
)

// Mystdin default terminal process
var Mystdin = NewTerminal()

// ProcessFunc
// param parameters
//
//line command
type ProcessFunc func(line string, param []string) bool
type ProcessCmd struct {
	pFunc     ProcessFunc
	allowExec bool
}

type Terminal struct {
	*liner.State
	mapFunc    map[string]ProcessCmd
	supported  bool
	normalMode liner.ModeApplier
	rawMode    liner.ModeApplier
	Isrun      bool
}

func (c *Terminal) RegisterProcessFunc(key string, f ProcessFunc, allowExec bool) {
	strKey := strings.ToLower(key)
	c.mapFunc[strKey] = ProcessCmd{pFunc: f, allowExec: allowExec}
	cmdNames[cmdPos] = strKey
	cmdPos++

}

func NewTerminal() *Terminal {
	p := new(Terminal)

	normalMode, _ := liner.TerminalMode()
	p.State = liner.NewLiner()
	p.mapFunc = make(map[string]ProcessCmd)

	rawMode, err := liner.TerminalMode()
	if err != nil || !liner.TerminalSupported() {
		p.supported = false
	} else {
		p.supported = true
		p.normalMode = normalMode
		p.rawMode = rawMode

		_ = normalMode.ApplyMode()
	}
	p.SetCtrlCAborts(true)
	p.SetTabCompletionStyle(liner.TabPrints)
	p.SetMultiLineMode(true)

	return p
}

func (c *Terminal) Run() {

	c.Isrun = true
	defer c.Close()

	c.SetCompleter(func(line string) (c []string) {
		for _, n := range cmdNames {
			if strings.HasPrefix(n, strings.ToLower(line)) {
				c = append(c, n)
			}
		}
		return
	})

	if c.supported {
		_ = c.rawMode.ApplyMode()
		defer func() {
			_ = c.normalMode.ApplyMode()
		}()
	} else {
		defer fmt.Println()
	}

	for c.Isrun {
		if name, err := c.Prompt(">"); err == nil {
			//log.Print("Got: " + name)
			name = strings.TrimSpace(name)               // Trim leading and trailing whitespaces
			re := regexp.MustCompile(`\s+`)              // Matches one or more consecutive whitespace characters
			trimmedCmd := re.ReplaceAllString(name, " ") // Replace with a single space
			//log.Print("Trimmed: " + trimmedCmd)
			cmdstring := strings.Split(trimmedCmd, " ")
			// if len(cmdstring) == 2 {
			// 	param = strings.Split(cmdstring[1], " ")
			// 	utils.DebugLog("param", param)
			// }
			param := cmdstring[1:]
			// utils.DebugLog("cmdstring", cmdstring)
			strkey := strings.ToLower(cmdstring[0])
			// utils.DebugLog("cmdstring", strkey, c.mapFunc)
			if exit := c.RunCmd(strkey, param, false); exit {
				return
			}

			c.AppendHistory(name)

		} else if err == liner.ErrPromptAborted {
			fmt.Println("Exit")
			return
		} else {
			fmt.Println("Error reading line: ", err)
		}

	}

}

func (c *Terminal) RunCmd(strkey string, param []string, isExec bool) bool {
	if pCmd, ok := c.mapFunc[strkey]; ok {
		if isExec && !pCmd.allowExec {
			fmt.Println("The command is not supported with 'exec'. Please run it in interaction mode")
		} else {
			pCmd.pFunc(strkey, param[:])
		}
	} else {
		if strkey == "exit" {
			return true
		}
		fmt.Println("The command is not found: ", strkey)
	}
	return false
}

func (p *Terminal) PromptPassword(prompt string) (passwd string, err error) {
	if p.supported {
		_ = p.rawMode.ApplyMode()
		defer func() {
			_ = p.normalMode.ApplyMode()
		}()
		return p.State.PasswordPrompt(prompt)
	}

	// Just as in Prompt, handle printing the prompt here instead of relying on liner.
	fmt.Print(prompt)
	passwd, err = p.State.Prompt("")
	fmt.Println()
	return passwd, err
}

func MyGetPassword(prompt string, confirmation bool) string {
	// Otherwise prompt the user for the password
	if prompt != "" {
		fmt.Println(prompt)
	}
	password, err := Mystdin.PromptPassword("password: ")
	if err != nil {
		cmd.Fatalf("Failed to read passphrase: %v", err)
	}
	if confirmation {
		confirm, err := Mystdin.PromptPassword("Repeat password: ")
		if err != nil {
			cmd.Fatalf("Failed to read passphrase confirmation: %v", err)
		}
		if password != confirm {
			password = ""
			cmd.Fatalf("password do not match")
		}
	}

	if Mystdin.supported {
		_ = Mystdin.rawMode.ApplyMode()
	} else {
		defer fmt.Println()
	}
	return password
}
