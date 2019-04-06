package main

import (
	"errors"
	"log"
	"os"
	"path"
)

type Manager struct {
	services   services
	DataDir    string
	logDirBase string
}

type ServiceStatus struct {
	Name  string
	State string
	Pid   int
}

func NewManger(compose ComposeFileStruct) *Manager {
	var services = make(services)
	cacheDir, _ := os.UserCacheDir()
	var dataDir = path.Join(cacheDir, "procyon")
	var logDirBase = path.Join(os.TempDir())
	log.Printf("dataDir mkdir: %s", dataDir)
	os.MkdirAll(dataDir, os.ModePerm)
	os.MkdirAll(logDirBase, os.ModePerm)
	for name, sd := range compose.Services {
		logdir := path.Join(logDirBase, name)
		log.Printf("making logging dir %s", logdir)
		os.MkdirAll(logdir, os.ModePerm)
		services[name] = &Process{
			Command: "/bin/sh",
			Args:    []string{"-c", sd.Command},
			Pidfile: Pidfile(path.Join(dataDir, name+".pid")),
			Errfile: path.Join(logdir, "stdout.log"),
			Logfile: path.Join(logdir, "stderr.log"),
		}
	}
	var m = Manager{
		DataDir:  dataDir,
		services: services,
	}
	return &m
}

func (m *Manager) List() []ServiceStatus {
	ss := make([]ServiceStatus, 0, 10)
	for _, s := range m.services {
		ss = append(ss, ServiceStatus{
			Name:  s.Name,
			State: s.Status,
			Pid:   s.Pid,
		})
	}
	return ss
}
func (m *Manager) Stop(serviceName string) error {
	if serviceName == "all" {
		for _, p := range m.services {
			p.stop()
			// delete(c, name)
		}
		return nil
	}
	p := m.services.get(serviceName)
	if p == nil {
		return errors.New("no service " + serviceName)
	}
	return p.stop()
}
func (m *Manager) Start(serviceName string) error {
	p := m.services.get(serviceName)
	if p == nil {
		return errors.New("no service " + serviceName)
	}
	if running, err := p.IsRunning(); running == true {
		return errors.New("service already running")
	} else if err != nil {
		return err
	}
	<-RunProcess(p.Name, p)
	return nil
}

func (m *Manager) IsRunning(serviceName string) (bool, error) {
	s := m.services.get(serviceName)
	if s == nil {
		return false, errors.New("no service " + serviceName)
	}
	if s != nil {
		return s.IsRunning()
	}
	return false, nil
}

//Run child processes
func (m *Manager) run() {
	for name, s := range m.services {
		RunProcess(name, s)
	}
}
