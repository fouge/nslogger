# nslogger parser

Very basic parser package for nslogger files. It will append text messages in a log file named after the input file.

Binary and images are not supported.

## Usage

`go get github.com/fouge/nslogger`

Here is an example where parsed data in plain text will be accessible in `fileToParse.rawnsloggerdata.txt`.

```go
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"github.com/fouge/nslogger"
)

func main() {
	filename := os.Args[1]

	var separator string

	/** Check if separator is specified
	 * Default separator in case no separator is specified */
	if len(os.Args) == 3 {
		separator = os.Args[2]
	} else {
		separator = ","
	}

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		
    	log.Fatal(err)
	}

	fmt.Println("Parsing file", filename, "of size", len(data), "bytes.")
	
	parsedDataStr, err := nslogger.NsLoggerParse(data, separator)
	if err != nil {
		log.Fatal(err)
	}

	outputFilename := filename + ".txt"
	fmt.Println("Writing log to", outputFilename)
	err = ioutil.WriteFile(outputFilename, []byte(parsedDataStr), 0644)
	if err != nil {
        log.Fatal(err)
	}
}

```

Build and run:
```
$ go build

$ ./go fileToParse.rawnsloggerdata

# With a separator specified. Default is ","
$ ./go fileToParse.rawnsloggerdata " | "
```
