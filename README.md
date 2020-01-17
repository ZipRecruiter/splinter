# ZipRecruiter linters for Go

This module is meant to hold analyzer packages that we build at ZipRecruiter.
When possible we'd prefer to contribute to other open source analyzers.

## pairs

The
[`github.com/ZipRecruiter/splinter/pairs`](https://godoc.org/github.com/ZipRecruiter/splinter/pairs)
linter will detect broken key/value pairs.  A key is defined as a string and a
value can be anything.

A missing value is an error:

```golang
logger.Log("name", "frew", "job", "engineer", "age" /* missing! */)
```

A non-string is also an error:

```golang
logger.Log("message", "successful!", /* missing key? */ 3)
```

### Example Run

```bash
$ splinter -pair-func ".Log=0" \                                            # anonymous interface
           -pair-func go.zr.org/common/go/errors/details.Pairs.AddPairs=0 \ # method
           -pair-func go.zr.org/common/go/errors.Wrap=2 \                   # func
           -assume-pair go.zr.org/common/go/errors/details.Pairs \          # type assumed safe
           ./...
```
