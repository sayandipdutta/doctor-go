# Building

## Install using `go`

```sh
go install github.com/sayandipdutta/doctor-go.git
```

## Install from source

```sh
git clone https://github.com/sayandipdutta/doctor-go.git
cd doctor-go
go build -o doc main.go doctypes.go
```

## Usage

```sh
doctor-go \
    -source "<path/to/batch or path/containing/batches>" \
    -dest "<output-dir-path>" [-withindex, -withbatch]
```

If `-withindex` is given, filenames will contain document types.

If `-withbatch` is given, document types of each deed in a batch will be put under
a directory with the same name as batch.
