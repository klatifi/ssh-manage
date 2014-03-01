package main

import (
	"errors"
	"os/user"
	
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

// Return the default SSH port
func getPort() int8 {
	return 22
}