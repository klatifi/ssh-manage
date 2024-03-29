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
	"time"

	"github.com/divoxx/llog"
	"github.com/peterbourgon/diskv"
)

const (
	transformBlockSize = 2 // grouping of chars per directory depth
	VERSION            = 0.1
)

var l *llog.Log

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
		h.Port = getPort()
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

// Update the information from the user by asking a series of questions
func (h *Host) updateConfig() error {
	fmt.Printf("Hostname(s) or aliases for the server (%s): ", h.Name)
	val, err := getInput()
	if err != nil {
		return err
	}
	if val != "" {
		h.Name = val
	}

	fmt.Printf("Hostname(s) or IP addresses of the server (%s): ", h.IP)
	val, err = getInput()
	if err != nil {
		return nil
	}
	if val != "" {
		h.IP = val
	}

	fmt.Printf("Port number of the server (%d): ", h.Port)
	val, err = getInput()
	if err != nil {
		return nil
	}
	if val != "" {
		port, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		h.Port = int8(port)
	}

	fmt.Printf("User on the server (%s): ", h.User)
	val, err = getInput()
	if err != nil {
		return nil
	}
	if val != "" {
		h.User = val
	}

	fmt.Printf("SSH key (%s): ", h.Key)
	val, err = getInput()
	if err != nil {
		return nil
	}
	if val != "" {
		h.Key = val
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
	ll := flag.String("log", "ERROR", "Set the log level")
	version := flag.Bool("version", false, "Display the version number")

	configDir, err := getConfigPath()
	if err != nil {
		panic(err)
	}

	envFile := configDir + "/ssh-manage.env"
	_, err = os.Stat(envFile)
	if err == nil {
		loadConfig(envFile)
	}

	d := diskv.New(diskv.Options{
		BasePath:     configDir + "/hosts", // where the data is stored
		Transform:    BlockTransform,
		CacheSizeMax: 1024 * 1024, // 1MB
	})

	flag.Usage = usage
	flag.Parse()

	if *version {
		// TODO print version number
		fmt.Printf("%s version: %v\n", os.Args[0], VERSION)
		os.Exit(0)
	}

	logLevel := getLogLevel(*ll)
	l = llog.New(os.Stdout, logLevel)

	logHandler("DEBUG", fmt.Sprintln("configuration directory:", configDir))

	if flag.NArg() == 0 {
		logHandler("ERROR", "please supply a command")
		// TODO list supported commands (Redirect to help message or usage text?)
		os.Exit(1)
	}

	// TODO add ability to update a record
	// TODO add the ability to set if a record or records should get
	// printed.  This needs to be host dependant.
	switch flag.Arg(0) {
	case "add":
		var hostInfo string
		if flag.Arg(2) != "" {
			hostInfo = flag.Arg(2)
		}

		err = addRecord(d, strings.TrimSpace(flag.Arg(1)), hostInfo)
		if err != nil {
			logHandler("ERROR",
				fmt.Sprintf("failed creating a new record: %s\n", err.Error()))
			os.Exit(1)
		}
	case "get":
		err := getRecord(d, strings.TrimSpace(flag.Arg(1)))
		if err != nil {
			logHandler("ERROR",
				fmt.Sprintf("failed fetching record details: %s\n", err.Error()))
			os.Exit(1)
		}
	case "list":
		err := listRecords(d)
		if err != nil {
			logHandler("ERROR",
				fmt.Sprintf("failed fetching all records: %s\n", err.Error()))
			os.Exit(1)
		}
	case "rm":
		err := removeRecord(d, strings.TrimSpace(flag.Arg(1)))
		if err != nil {
			logHandler("ERROR",
				fmt.Sprintf("failed removing record: %s\n", err.Error()))
			os.Exit(1)
		}
	case "write":
		err := writeFile(d)
		if err != nil {
			logHandler("ERROR",
				fmt.Sprintf("failed when writing out SSH configuration file: %s\n",
					err.Error()))
			os.Exit(1)
		}
	case "update":
		if flag.Arg(1) == "" {
			logHandler("ERROR", "update requires an argument")
			os.Exit(1)
		}

		err := updateRecord(d, strings.TrimSpace(flag.Arg(1)))
		if err != nil {
			logHandler("ERROR",
				fmt.Sprintf("faild updating record: %s\n", err.Error()))
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(1)
	}

	os.Exit(0)
}

func usage() {
	command := os.Args[0]
	fmt.Fprintf(os.Stderr,
		`Usage: %s [options] {add|get|list|rm|write|update} [arguments]
%s requires one of the following commands:

add:   Add a new host record to the datastore
get:   Get details about a host record from the datastore
list:  Lists all records in the datastore
rm:    Removes a record from the datastore
write: Write out SSH configuration file

Options:
  --help: Displays this help message
`, command, command)
}

func getLogLevel(ll string) llog.Level {
	switch ll {
	case "debug":
		return llog.DEBUG
	case "info":
		return llog.INFO
	case "warn":
		return llog.WARNING
	case "error":
		return llog.ERROR
	default:
		return llog.ERROR
	}
}

func logHandler(lvl, msg string) {
	switch lvl {
	case "DEBUG":
		l.Debug("[DEBUG]", logTime(), msg)
	case "INFO":
		l.Info("[INFO]", logTime(), msg)
	case "WARN":
		l.Warning("[WARNING]", logTime(), msg)
	case "ERROR":
		l.Error("[ERROR]", logTime(), msg)
	default:
		return
	}
}

func logTime() string {
	return time.Now().Format(time.RFC3339)
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

// Given a record remove it from from the datastore.
func removeRecord(d *diskv.Diskv, name string) error {
	err := d.Erase(md5sum(name))
	if err != nil {
		return fmt.Errorf("no configuration found for %s\n", name)
	}
	return nil
}

// Given a record allow settings to be updated.
func updateRecord(d *diskv.Diskv, name string) error {
	var h Host
	val, err := d.Read(md5sum(name))
	if err != nil {
		return fmt.Errorf("no configuration found for %s\n", name)
	}

	err = json.Unmarshal(val, &h)
	if err != nil {
		return fmt.Errorf("failed to parse the configuration: %s\n", err.Error())
	}

	err = h.updateConfig()
	if err != nil {
		return fmt.Errorf("failed to update the host information: %s", err.Error())
	}

	val, err = json.Marshal(h)
	if err != nil {
		return err
	}

	d.Write(md5sum(name), []byte(val))

	return nil
}
