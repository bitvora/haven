# Building Haven

This document provides instructions for building the Haven relay from source.

## Prerequisites

- **Go**: Ensure you have Go installed on your system. You can download it from [here](https://golang.org/dl/).

    ```bash
    sudo apt update #Update Package List
    sudo apt install snapd #install snapd to get a newer version of Go
    sudo snap install go --classic #Install Go
    go version #check if go was installed correctly
    ```

- **Build Essentials**: If you're using Linux, you may need to install build essentials. You can do this by running `sudo apt install build-essential`.

## Building from Source

### 1. Clone the repository

```bash
git clone https://github.com/bitvora/haven.git
cd haven
```

### 2. Build the project

Run the following command to build the relay:

```bash
go build
```

After building the project, you can proceed with the [setup instructions](../README.md#setup-instructions) in the main README.