Elastos ELA Arbiter
===========
[![Build Status](https://travis-ci.org/elastos/Elastos.ELA.Arbiter.svg?branch=master)](https://travis-ci.org/elastos/Elastos.ELA.Arbiter) [![Go Report Card](https://goreportcard.com/badge/github.com/elastos/Elastos.ELA.Arbiter)](https://goreportcard.com/report/github.com/elastos/Elastos.ELA.Arbiter)

## Introduction
##### This project is the source code that can build a arbiter of ELA.


## Table of Contents
- [Elastos ELA Arbiter](#elastos-ela-arbiter)
    - [Introduction](#introduction)
    - [Table of Contents](#table-of-contents)
- [Pre-requisites on Mac](#pre-requisites-on-mac)
    - [1. Check OS version](#1-check-os-version)
    - [2. Install Go distribution 1.9](#2-install-go-distribution-19)
- [Pre-requisites on Ubuntu](#pre-requisites-on-ubuntu)
    - [1. Check OS version](#1-check-os-version-1)
    - [2. Install git](#2-install-git)
    - [3. Install Go distribution 1.9](#3-install-go-distribution-19)
- [Configure the node](#configure-the-node)
- [Build the node](#build-the-node)
    - [1. Setup basic workspace](#1-setup-basic-workspace)
    - [2. Set correct environment variables](#2-set-correct-environment-variables)
    - [3. Check Go version](#3-check-go-version)
    - [4. Clone source code to $GOPATH/src/github/elastos folder](#4-clone-source-code-to-gopathsrcgithubelastos-folder)
    - [5. Make](#5-make)
    - [6. Run the node on Mac](#6-run-the-node-on-mac)
- [Interact with the node](#interact-with-the-node)
    - [1. JSON RPC API of the node](#1-json-rpc-api-of-the-node)
- [Contribution](#contribution)
- [Acknowledgments](#acknowledgments)
- [License](#license)

## Pre-requisites on Mac

#### 1. Check OS version

Make sure the OSX version is 16.7+

```shell
$ uname -srm
Darwin 16.7.0 x86_64
```

#### 2. Install Go distribution 1.9

Use Homebrew to install Golang 1.9.

```shell
$ brew install go@1.9
```

> If you install older version, such as v1.8, you may get missing math/bits package error when build.


## Pre-requisites on Ubuntu

#### 1. Check OS version

Make sure your ubuntu version is 16.04+

```shell
$ cat /etc/issue
Ubuntu 16.04.3 LTS \n \l
```

#### 2. Install git

```shell
$ sudo apt-get install -y git
```

#### 3. Install Go distribution 1.9

```shell
$ sudo apt-get install -y software-properties-common
$ sudo add-apt-repository -y ppa:gophers/archive
$ sudo apt update
$ sudo apt-get install -y golang-1.9-go
```

> If you install older version, such as v1.8, you may get missing math/bits package error when build.

## Configure the node

See the [documentation](./docs/config.json.md) about config.json to understand what each parameter means on the configuration file.

If you would like to connect to testnet, do the following:
```shell
$ cp docs/testnet_config.json.sample config.json
```

If you would like to connect to mainnet, do the following:
```shell
$ cp docs/mainnet_config.json.sample config.json
```

Make sure to modify the parameters to what your own specification. 

## Build the node

#### 1. Setup basic workspace
In this instruction we use ~/dev/src/github.com/elastos as our working directory. If you clone the source code to a different directory, please make sure you change other environment variables accordingly (not recommended). 

```shell
$ mkdir -p ~/dev/bin
$ mkdir -p ~/dev/src/github.com/elastos/
```

#### 2. Set correct environment variables

```shell
export GOROOT=/usr/local/opt/go@1.9/libexec
export GOPATH=$HOME/dev
export GOBIN=$GOPATH/bin
export PATH=$GOROOT/bin:$PATH
export PATH=$GOBIN:$PATH
```

#### 3. Check Go version

Check the golang version. Make sure they are the following version number or above.

```shell
$ go version
go version go1.13.4 darwin/amd64
```

If you cannot see the version number, there must be something wrong when install.

#### 4. Clone source code to $GOPATH/src/github/elastos folder
Make sure you are in the folder of $GOPATH/src/github.com/elastos
```shell
$ git clone https://github.com/elastos/Elastos.ELA.Arbiter.git
```

If clone works successfully, you should see folder structure like $GOPATH/src/github.com/elastos/Elastos.ELA.Arbiter/Makefile

#### 5. Make

Build the node.
```shell
$ cd $GOPATH/src/github.com/elastos/Elastos.ELA.Arbiter
$ make
```

If you did not see any error message, congratulations, you have made the arbiter full node.

#### 6. Run the node on Mac

Run the node.
```shell
$ ./arbiter -p password
```

## Interact with the node

#### 1. JSON RPC API of the node

Once the node is running successfully, you can access Arbiter Node's JSON RPC APIs:

Example 1: Get the block height synchronized by SPV from the main chain
```bash
curl -H 'Content-Type: application/json' -H 'Accept:application/json' --data '{"method":"getspvheight"}' http://localhost:20606
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": 2509
}
```

Example 2: Get the block height of side chain by side chain genesis block hash
```bash
curl -H 'Content-Type: application/json' -H 'Accept:application/json' --data '{"method":"getsidechainblockheight","params":{"hash":"56be936978c261b2e649d58dbfaf3f23d4a868274f5522cd2adb4308a955c4a3"}}' http://localhost:20606
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": 70
}
```

If you would like to learn more about what other JSON RPC APIs are available for the node, please check out the [JSON RPC API](docs/jsonrpc_apis.md)

## Contribution

We welcome contributions to the Elastos ELA Arbiter Project.

## Acknowledgments

A sincere thank you to all teams and projects that we rely on directly or indirectly.

## License 

This project is licensed under the terms of the [MIT license](https://github.com/elastos/Elastos.ELA.Arbiter/blob/master/LICENSE).