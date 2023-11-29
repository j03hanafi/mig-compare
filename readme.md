# MIG Generator for Stroberi Tagihan V3

## How to use
```bash
$ go install github.com/j03hanafi/mig-compare@latest

$ cd <git-repository>
	
$ mig-compare --source <branch-name> --target <branch-name>
```

It will generate a file named `comparison.csv` in the current directory.

## Usage of mig-compare
```bash
$ mig-compare --help

  -dir string
        Path to the repository directory
  -output string
        Path/name to the output CSV file
  -source string
        Name of the first branch to compare
  -target string
        Name of the second branch to compare
```