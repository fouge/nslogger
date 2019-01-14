# nslogger parser

Very basic parser for nslogger files. It will append text messages in a log file named after the input file.

Binary and images are not supported.

## Usage

Parsed data in plain text will be accessible in `fileToParse.rawnsloggerdata.txt`.

```
$ go build

$ ./nslogger fileToParse.rawnsloggerdata

# With a separator specified. Default is ","
$ ./nslogger fileToParse.rawnsloggerdata " | "
```
