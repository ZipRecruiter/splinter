package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/ZipRecruiter/splinter/pairs"
)

func main() { singlechecker.Main(pairs.NewAnalyzer()) }
