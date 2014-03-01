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

TODO
