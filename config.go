package main

import (
	"errors"
	"os"
	"os/user"
	"strconv"

	"github.com/subosito/gotenv"
)

// Return the name of the current user.
func getCurrentUser() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", errors.New("failed getting current user information")
	}
	return u.Username, nil
}

// Return the home directory of the current user.
func getHome() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", errors.New("faild getting current user information")
	}
	return u.HomeDir, nil
}

// Read the evironment file, populating the environment variables to use.
func loadConfig(envFile string) {
	gotenv.Load()
}

// Return the default SSH port, tries fetching from the environment first.
func getPort() int8 {
	var port int8
	envPort := os.Getenv("SSH-PORT")
	if envPort != "" {
		p, err := strconv.Atoi(envPort)
		if err != nil {
			return 22
		}
		port = int8(p)
	} else {
		port = 22
	}
	return port
}
