# ssh-manage

The goal of ssh-manage is to be a simple tool to help manage user configuration 
files for SSH.  The host configurations are stored in a unique key-value data 
store (think nosql like).

## Installation

ssh-manage requires Go 1 or higher, and can be installed with the /go get/ tool:

    go get github.com/vendion/ssh-manage

## Configuring

ssh-manage is designed to work out of the box making sane defaults.  These 
defaults can be changed by setting environment variables.  These can be 
manually set, or ssh-manage can load them on start up.

To have ssh-manage load these environment variables, first create 
~/.config/ssh-manage/ssh-manage.env and add the values to that file.

Here is a example file:

    SSH-PORT=2222

This tells ssh-manage to use port 2222 as the default SSH port.

## Usage

### Adding a host

Adding a host can be done one of two ways

#### Interactive

The interactive method asks a series of questions to get the information needed 
to create the host record.  Some of these fields can be left blank and defualts 
will be used instead.

    ssh-manage add example
    Hostname(s) or alias(es) of the server: example.com
    Hostname or IP address of the server: example.com
    Port number of server: 22
    User on server: john
    SSH key: ~/.ssh/id_rsa

#### Non-interactive

The Non-interactive method only takes two fields hostname or IP address of the 
server and which key to use.  Everything else defaults are used.

    ssh-manage add example example.com:~/.ssh/id_rsa

### Getting a host details

    ssh-manage get example

### Listing all hosts

    ssh-manage List

### Remove a host

    ssh-manage rm example

### Write the ssh configuration file

    ssh-manage Write