# Go Licenser

A simple utility which can extract dependencies from GO projects, by parsing `*.go` files and `Godeps*`.

## Usage

### Find all dependencies

This is currently done with a bash script.

```bash
# Process a single repo
scripts/go-find-github-deps.sh /path/to/repo > licenses.txt

# Process multiple repos
scripts/go-find-github-deps.sh /path/to/repo1 /path/to/repo2 ... > licenses.txt
```

### Compile and find licenses

```bash
# Compile the tool (tested on GO 1.12)
git clone git@github.com:MihaiBojin/golicenser.git && cd golicenser
make install

# Identify licenses using the GitHub API
golicenser licenses.txt GIT_PERSONAL_TOKEN

# The license report will be saved to a file
```

Replace the [Personal Token](https://github.com/settings/tokens) with a real value.
