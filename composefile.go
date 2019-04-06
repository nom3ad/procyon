package main

type Environ map[string]string

type ServiceConfig struct {
	Shell       bool    `yaml:"shell"`
	Command     string  `yaml:"command"`
	Environment Environ `yaml:"environment"`
}

type ComposeFileStruct struct {
	Services map[string]ServiceConfig `yaml:"services"`
}
