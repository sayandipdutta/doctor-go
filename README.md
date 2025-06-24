# doctor-go: Gather documents based on heuristics

## Building

### Install using `go`

```sh
go install github.com/sayandipdutta/doctor-go.git
```

### Build from source

```sh
git clone https://github.com/sayandipdutta/doctor-go.git
cd doctor-go
go build -o doc main.go doctypes.go
```

## Usage

```sh
doctor-go \
    -source "<path/to/batch or path/containing/batches>" \
    -dest "<output-dir-path>" [-task <taskname>] [-withindex, -withbatch]
```

If `-withindex` is given, filenames will contain document types.

If `-withbatch` is given, document types of each deed in a batch will be put under
a directory with the same name as batch.

`-task` specifies what type of task to perform. By default it is `doctype`.
Possible values of `-task` are: `doctype`, `topsheet`.


### Task `doctype`:

If `-task` is `doctype` (default), the program gathers starting image of doctype
sequences per deed, across batches (based on source path).


### Task `topsheet`

If `-task` is `topsheet`, the program gathers topsheet file per deed, for all deeds
under given source path.
