Pasang
======
[![Build Status](https://travis-ci.org/asasmoyo/pasang.svg?branch=master)](https://travis-ci.org/asasmoyo/pasang)

Single binary to deploy a release via SSH.

# Installation

You can download pre-built binary in [release page](https://github.com/asasmoyo/pasang/releases/latest).

You can also download and build it for yourself using `go get github.com/asasmoyo/pasang`.

# Usage

Create a configuration file in json, then pass the file as argument to binary using `--config` flag.

Supported options are:

1. `source` source file / folder to deploy
1. `dest` remote folder where to deploy
1. `keep` how many releases you want to keep
1. `local` deploy to local folder
1. `before_deploy_cmd` array of commands you want to run in `source` folder before `source` is copied to `dest`
1. `after_deploy_cmd` array of commands you want to run in `dest` folder after `source` is copied to `dest`. If `dest` is in remote machine, the commands will be run on remote machine.
1. `ssh_password` password for doing SSH authentication
1. `ssh_private_key` path to private key for doing SSH authentication. If both private key and password are present, authentication will be done using private key
1. `ssh_secret_key` secret key if your private key is protected

# Limitation

1. SSH host key check is ignored.
1. Complex path in `dest`, eg: `user@host:port:~/deploy`, may not be supported. Please use absolute path for now.

# Licence

MIT Licence
