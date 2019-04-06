package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"
)

var ping = "1m"

//Run the process
func RunProcess(name string, p *Process) chan *Process {
	ch := make(chan *Process)
	go func() {
		p.start(name)
		p.ping(ping, func(time time.Duration, p *Process) {
			if p.Pid > 0 {
				p.respawns = 0
				fmt.Printf("%s refreshed after %s.\n", p.Name, time)
				p.Status = "running"
			}
		})
		go p.watch()
		ch <- p
	}()
	return ch
}

type Process struct {
	Name     string
	Command  string
	Args     []string
	Pidfile  Pidfile
	Logfile  string
	Errfile  string
	Path     string
	Respawn  int
	Delay    string
	Ping     string
	Pid      int
	Status   string
	x        *os.Process
	respawns int
}

func (p *Process) String() string {
	js, err := json.Marshal(p)
	if err != nil {
		log.Print(err)
		return ""
	}
	return string(js)
}

func (p *Process) IsRunning() (bool, error) {
	proc, err := p.FindProcess()
	if err != nil {
		return false, err
	}
	return proc != nil, nil
}

//Find a process by name
func (p *Process) FindProcess() (*os.Process, error) {
	if p.Pidfile == "" {
		return nil, errors.New("Pidfile is empty.")
	}
	if pid := p.Pidfile.read(); pid > 0 {
		process, err := os.FindProcess(pid)
		if err != nil {
			return nil, err
		}
		p.x = process
		p.Pid = process.Pid
		p.Status = "running"
		return process, nil
	}
	return nil, nil
}

//Start the process
func (p *Process) start(name string) string {
	p.Name = name
	wd, _ := os.Getwd()
	proc := &os.ProcAttr{
		Dir: wd,
		Env: os.Environ(),
		Files: []*os.File{
			os.Stdin,
			NewLog(p.Logfile),
			NewLog(p.Errfile),
		},
	}
	args := append([]string{p.Name}, p.Args...)
	process, err := os.StartProcess(p.Command, args, proc)
	if err != nil {
		log.Fatalf("%s failed. %s\n", p.Name, err)
		return ""
	}
	err = p.Pidfile.write(process.Pid)
	if err != nil {
		log.Printf("%s pidfile error: %s\n", p.Name, err)
		return ""
	}
	p.x = process
	p.Pid = process.Pid
	p.Status = "started"
	return fmt.Sprintf("%s is %#v\n", p.Name, process.Pid)
}

//Stop the process
func (p *Process) stop() error {
	log.Printf("Stopping %s", p.Name)
	if p.x != nil {
		// p.x.Kill() this seems to cause trouble
		cmd := exec.Command("kill", fmt.Sprintf("%d", p.x.Pid))
		_, err := cmd.CombinedOutput()
		if err != nil {
			log.Println(err)
			return err
		}
	}
	p.release("stopped")
	return nil
}

//Release process and remove pidfile
func (p *Process) release(status string) {
	if p.x != nil {
		p.x.Release()
	}
	p.Pid = 0
	p.Pidfile.delete()
	p.Status = status
}

//Restart the process
func (p *Process) restart() (chan *Process, string) {
	p.stop()
	message := fmt.Sprintf("%s restarted.\n", p.Name)
	ch := RunProcess(p.Name, p)
	return ch, message
}

//Run callback on the process after given duration.
func (p *Process) ping(duration string, f func(t time.Duration, p *Process)) {
	if p.Ping != "" {
		duration = p.Ping
	}
	t, err := time.ParseDuration(duration)
	if err != nil {
		t, _ = time.ParseDuration(ping)
	}
	go func() {
		select {
		case <-time.After(t):
			f(t, p)
		}
	}()
}

//Watch the process
func (p *Process) watch() {
	if p.x == nil {
		p.release("stopped")
		return
	}
	status := make(chan *os.ProcessState)
	died := make(chan error)
	go func() {
		state, err := p.x.Wait()
		if err != nil {
			died <- err
			return
		}
		status <- state
	}()
	select {
	case s := <-status:
		if p.Status == "stopped" {
			return
		}
		fmt.Fprintf(os.Stderr, "%s %s\n", p.Name, s)
		fmt.Fprintf(os.Stderr, "%s success = %#v\n", p.Name, s.Success())
		fmt.Fprintf(os.Stderr, "%s exited = %#v\n", p.Name, s.Exited())
		p.respawns++
		if p.respawns > p.Respawn {
			p.release("exited")
			log.Printf("%s respawn limit reached.\n", p.Name)
			return
		}
		fmt.Fprintf(os.Stderr, "%s respawns = %#v\n", p.Name, p.respawns)
		if p.Delay != "" {
			t, _ := time.ParseDuration(p.Delay)
			time.Sleep(t)
		}
		p.restart()
		p.Status = "restarted"
	case err := <-died:
		p.release("killed")
		log.Printf("%d %s killed = %#v", p.x.Pid, p.Name, err)
	}
}

//Child processes.
type services map[string]*Process

//Stringify
func (c services) String() string {
	js, err := json.Marshal(c)
	if err != nil {
		log.Print(err)
		return ""
	}
	return string(js)
}

//Get child processes names.
func (c services) keys() []string {
	keys := []string{}
	for k, _ := range c {
		keys = append(keys, k)
	}
	return keys
}

//Get child process.
func (c services) get(key string) *Process {
	if v, ok := c[key]; ok {
		return v
	}
	return nil
}

type Pidfile string

//Read the pidfile.
func (f *Pidfile) read() int {
	data, err := ioutil.ReadFile(string(*f))
	if err != nil {
		return 0
	}
	pid, err := strconv.ParseInt(string(data), 0, 32)
	if err != nil {
		return 0
	}
	return int(pid)
}

//Write the pidfile.
func (f *Pidfile) write(data int) error {
	err := ioutil.WriteFile(string(*f), []byte(strconv.Itoa(data)), 0660)
	if err != nil {
		return err
	}
	return nil
}

//Delete the pidfile
func (f *Pidfile) delete() bool {
	_, err := os.Stat(string(*f))
	if err != nil {
		return true
	}
	err = os.Remove(string(*f))
	if err == nil {
		return true
	}
	return false
}

//Create a new file for logging
func NewLog(path string) *os.File {
	if path == "" {
		return nil
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)
	if err != nil {
		log.Fatalf("%s", err)
		return nil
	}
	return file
}