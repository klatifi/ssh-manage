package main

import (
	"bufio"
	"crypto/md5"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/peterbourgon/diskv"
)

const (
	transformBlockSize = 2 // grouping of chars per directory depth
)

// Host contains the configuration details for a SSH Host.
type Host struct {
	// Name provided to ssh-manage, used to get/update a configuration
	Nickname string
	// Hostname or alias for the server
	Name string
	// Fully qualified hostname or IP address to connect
	IP string
	// Port number that the SSH daemon is listening on
	Port int8
	// The name of the remote user to connect as
	User string
	// Path the SSH key that should be used when connecting
	Key string
	// Interval that the ServerKeepAlive command should be passed to the server
	KeepAlive int8
}

// Get configuration details from defaults and arguments
func (h *Host) config(info []string) error {
	var err error
	if info[0] == "" {
		return errors.New("hostname or alias is required")
	}

	if info[1] == "" {
		return errors.New("SSH key is required")
	}

	h.Name = info[0]
	h.IP = info[0]
	h.Port = getPort()

	h.User, err = getCurrentUser()
	if err != nil {
		return err
	}
	h.Key = info[1]
	return nil
}

// Get the information from the user by asking a series of questions
func (h *Host) interactiveConfig() error {
	var err error
	fmt.Print("Hostname(s) or alias(s) of server: ")
	h.Name, err = getInput()
	if err != nil {
		return err
	}

	if h.Name == "" {
		return errors.New("hostname or alias is required to continue")
	}

	fmt.Print("Hostname or IP address of server: ")
	h.IP, err = getInput()
	if err != nil {
		return err
	}

	if h.IP == "" {
		h.IP = h.Name
	}

	fmt.Print("Port number of server: ")
	p, err := getInput()
	if err != nil {
		return err
	}

	if p != "" {
		port, err := strconv.Atoi(p)
		if err != nil {
			return err
		}
		h.Port = int8(port)
	} else {
		h.Port = 22
	}

	fmt.Print("User on server: ")
	h.User, err = getInput()
	if err != nil {
		return err
	}

	if h.User == "" {
		h.User, err = getCurrentUser()
		if err != nil {
			return err
		}
	}

	fmt.Print("SSH key: ")
	h.Key, err = getInput()
	if err != nil {
		return err
	}

	_, err = os.Stat(h.Key)
	if err != nil {
		return errors.New("SSH key does not exist")
	}
	return nil
}

// BlockTransform builds out the directory structure for file.
func BlockTransform(s string) []string {
	sliceSize := len(s) / transformBlockSize
	pathSlice := make([]string, sliceSize)
	for i := 0; i < sliceSize; i++ {
		from, to := i*transformBlockSize, (i*transformBlockSize)+transformBlockSize
		pathSlice[i] = s[from:to]
	}
	return pathSlice
}

func main() {
	home, err := getHome()
	if err != nil {
		glog.Errorln(err)
		os.Exit(1)
	}
	
	envFile := home+"/.ssh-manage.env"
	_, err = os.Stat(envFile)
	if err == nil {
		loadConfig(envFile)
	}
	
	d := diskv.New(diskv.Options{
		BasePath:     "data", // where the data is stored
		Transform:    BlockTransform,
		CacheSizeMax: 1024 * 1024, // 1MB
	})

	flag.Parse()
	if flag.NArg() == 0 {
		glog.Errorln("please supply a command")
		// TODO list supported commands (Redirect to help message or usage text?)
		os.Exit(1)
	}

	switch flag.Arg(0) {
	case "add":
		var hostInfo string
		if flag.Arg(2) != "" {
			hostInfo = flag.Arg(2)
		}

		err = addRecord(d, strings.TrimSpace(flag.Arg(1)), hostInfo)
		if err != nil {
			glog.Errorf("an error has occured when adding a new record: %s\n", err.Error())
			os.Exit(1)
		}
	case "get":
		err := getRecord(d, strings.TrimSpace(flag.Arg(1)))
		if err != nil {
			glog.Errorf("an error has occured when fetching record details: %s\n", err.Error())
			os.Exit(1)
		}
	case "list":
		err := listRecords(d)
		if err != nil {
			glog.Errorf("an error has occured when fetching all records: %s\n", err.Error())
			os.Exit(1)
		}
	case "rm":
		err := removeRecord(d, strings.TrimSpace(flag.Arg(1)))
		if err != nil {
			glog.Errorf("an error has occured when removing a record: %s\n", err.Error())
			os.Exit(1)
		}
	}

	os.Exit(0)
}

func md5sum(s string) string {
	h := md5.New()
	io.WriteString(h, s)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Read in input using STDIN, returns the value as a string.
func getInput() (string, error) {
	s := bufio.NewScanner(os.Stdin)
	s.Scan()
	if err := s.Err(); err != nil {
		return "", err
	}
	t := s.Text()
	return t, nil
}

// Create the configuration record and store it in the datastore
func addRecord(d *diskv.Diskv, name, hostInfo string) error {
	h := Host{Nickname: name, KeepAlive: 30}
	var err error

	if hostInfo == "" {
		err = h.interactiveConfig()
		if err != nil {
			return err
		}
	} else {
		info := strings.Split(hostInfo, ":")
		err = h.config(info)
		if err != nil {
			return err
		}
	}

	val, err := json.Marshal(h)
	if err != nil {
		return err
	}
	d.Write(md5sum(name), []byte(val))
	return nil
}

// Fetch a record from the datastore and display the current configuration to the user
func getRecord(d *diskv.Diskv, name string) error {
	val, err := d.Read(md5sum(name))
	if err != nil {
		return fmt.Errorf("no configuration found for %s", name)
	}

	fmt.Println("Configuration for", name)
	fmt.Println(string(val))
	return nil
}

// Lets the nickname for all configurations currently in the datastore
func listRecords(d *diskv.Diskv) error {
	var h Host
	keyChan, keyCount := d.Keys(), 0
	for key := range keyChan {
		val, err := d.Read(key)
		if err != nil {
			return err
		}

		err = json.Unmarshal(val, &h)
		if err != nil {
			return err
		}
		fmt.Printf("%s: %s\n", h.Nickname, val)
		keyCount++
	}
	fmt.Println("Total configuration(s) currently stored:", keyCount)
	return nil
}

func removeRecord(d *diskv.Diskv, name string) error {
	err := d.Erase(md5sum(name))
	if err != nil {
		return fmt.Errorf("no configuration found for %s\n", name)
	}
	return nil
}
