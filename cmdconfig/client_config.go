package cmdconfig

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

//StartClientFromFile is good
func StartClientFromFile(filePath string) error {
	startReadingCmd()
	return nil
}

func startReadingCmd() {
	invalidCommandError := errors.New("Invalid command")
	var err error

	for scanner.Scan() {
		cmd := strings.Fields(scanner.Text())

		err = nil
		l := len(cmd)

		if l == 0 {
			err = errors.New("Command cannot be empty")
		}

		if err == nil {
			switch cmd[0] {
			case cmdQuery:
				if l != 2 {
					err = combineErrorUsage(invalidCommandError, cmd[0])
					break
				}
				// result, e := p.QueryStateMachine(cmd[1])
				// if e != nil {
				// 	err = e
				// 	break
				// }
				// fmt.Print("The query result for key: ", cmd[1], " is ", result)
			case cmdSet, cmdIncre:
				if l != 3 {
					err = combineErrorUsage(invalidCommandError, cmd[0])
					break
				}
				_, e := strconv.Atoi(cmd[2])
				if e != nil {
					err = errors.New("value should be an integer")
					break
				}
				// switch cmd[0] {
				// case cmdSet:
				// 	go func() {
				// 		p.HandleClientCmd(sm.TSMActionSetValue(cmd[1], value))
				// 		fmt.Println("Request ", cmd, " sent")
				// 	}()
				// case cmdIncre:
				// 	go func() {
				// 		p.HandleClientCmd(sm.TSMActionIncrValue(cmd[1], value))
				// 		fmt.Println("Request ", cmd, " sent")
				// 	}()
				// }
			case cmdMove:
				if l != 4 {
					err = combineErrorUsage(invalidCommandError, cmd[0])
					break
				}
				_, e := strconv.Atoi(cmd[3])
				if e != nil {
					err = errors.New("value should be an integer")
					break
				}
				// go func() {
				// 	p.HandleClientCmd(sm.TSMActionMoveValue(cmd[1], cmd[2], value))
				// 	fmt.Println("Request ", cmd, " sent")
				// }()
			default:
				err = invalidCommandError
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Failed reading stdout: ", err)
	}
}

func combineErrorUsage(e error, cmd string) error {
	return errors.New(e.Error() + "\nUsage: " + cmd + " " + usageMp[cmd])
}
