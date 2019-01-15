package nslogger

import (
	. "bytes"
	. "encoding/binary"
	"errors"
	"fmt"
	"log"
	"time"
)

/* NSLogger native binary message format:
 * Each message is a dictionary encoded in a compact format. All values are stored
 * in network order (big endian). A message is made of several "parts", which are
 * typed chunks of data, each with a specific purpose (partKey), data type (partType)
 * and data size (partSize).
 *
 *	uint32_t	totalSize		(total size for the whole message excluding this 4-byte count)
 *	uint16_t	partCount		(number of parts below)
 *  [repeat partCount times]:
 *		uint8_t		partKey		the part key
 *		uint8_t		partType	(string, binary, image, int16, int32, int64)
 *		uint32_t	partSize	(only for string, binary and image types, others are implicit)
 *		.. `partSize' data bytes
 *
 * Complete message is usually made of:
 *	- a PART_KEY_MESSAGE_TYPE (mandatory) which contains one of the LOGMSG_TYPE_* values
 *  - a PART_KEY_TIMESTAMP_S (mandatory) which is the timestamp returned by gettimeofday() (seconds from 01.01.1970 00:00)
 *	- a PART_KEY_TIMESTAMP_MS (optional) complement of the timestamp seconds, in milliseconds
 *	- a PART_KEY_TIMESTAMP_US (optional) complement of the timestamp seconds and milliseconds, in microseconds
 *	- a PART_KEY_THREAD_ID (mandatory) the ID of the user thread that produced the log entry
 *	- a PART_KEY_TAG (optional) a tag that helps categorizing and filtering logs from your application, and shows up in viewer logs
 *	- a PART_KEY_LEVEL (optional) a log level that helps filtering logs from your application (see as few or as much detail as you need)
 *	- a PART_KEY_MESSAGE which is the message text, binary data or image
 *  - a PART_KEY_MESSAGE_SEQ which is the message sequence number (message# sent by client)
 *	- a PART_KEY_FILENAME (optional) with the filename from which the log was generated
 *	- a PART_KEY_LINENUMBER (optional) the linenumber in the filename at which the log was generated
 *	- a PART_KEY_FUNCTIONNAME (optional) the function / method / selector from which the log was generated
 *  - if logging an image, PART_KEY_IMAGE_WIDTH and PART_KEY_IMAGE_HEIGHT let the desktop know the image size without having to actually decode it
 */

// Constants for the "part key" field

const PartKeyMessageType = 0
const PartKeyTimestampS = 1  // "seconds" component of timestamp
const PartKeyTimestampMs = 2 // milliseconds component of timestamp (optional, mutually exclusive with PART_KEY_TIMESTAMP_US)
const PartKeyTimestampUs = 3 // microseconds component of timestamp (optional, mutually exclusive with PART_KEY_TIMESTAMP_MS)
const PartKeyThreadId = 4
const PartKeyTag = 5
const PartKeyLevel = 6
const PartKeyMessage = 7
const PartKeyImageWidth = 8    // messages containing an image should also contain a part with the image size
const PartKeyImageHeight = 9   // (this is mainly for the desktop viewer to compute the cell size without having to immediately decode the image)
const PartKeyMessageSeq = 10   // the sequential number of this message which indicates the order in which messages are generated
const PartKeyFilename = 11     // when logging, message can contain a file name
const PartKeyLinenumber = 12   // as well as a line number
const PartKeyFunctionname = 13 // and a function or method name

// Constants for parts in LOGMSG_TYPE_CLIENTINFO

const PartKeyClientName = 20
const PartKeyClientVersion = 21
const PartKeyOsName = 22
const PartKeyOsVersion = 23
const PartKeyClientModel = 24 // For iPhone, device model (i.e 'iPhone', 'iPad', etc)
const PartKeyUniqueid = 25    // for remote device identification, part of LOGMSG_TYPE_CLIENTINFO

// Area starting at which you may define your own constants

const PartKeyUserDefined = 100

// Constants for the "partType" field

const PartTypeString = 0 // Strings are stored as UTF-8 data
const PartTypeBinary = 1 // A block of binary data
const PartTypeInt16 = 2
const PartTypeInt32 = 3
const PartTypeInt64 = 4
const PartTypeImage = 5 // An image, stored in PNG format

// Data values for the PART_KEY_MESSAGE_TYPE parts

const LogmsgTypeLog = 0        // A standard log message
const LogmsgTypeBlockstart = 1 // The start of a "block" (a group of log entries)
const LogmsgTypeBlockend = 2   // The end of the last started "block"
const LogmsgTypeClientinfo = 3 // Information about the client app
const LogmsgTypeDisconnect = 4 // Pseudo-message on the desktop side to identify client disconnects
const LogmsgTypeMark = 5       // Pseudo-message that defines a "mark" that users can place in the log flow

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

/** appendValue append new data part to log message */
func appendValue(b []byte, nBytes uint32, m logMessage) uint32 {
	partSize := uint32(0)
	switch partType := b[nBytes+1]; partType {
	case PartTypeInt16:
		partSize = 2
		var val int16
		err := Read(NewReader(b[2+nBytes:2+nBytes+partSize]), BigEndian, &val)
		check(err)
		m.addInt16(val)
	case PartTypeInt32:
		partSize = 4
		var val int32
		err := Read(NewReader(b[nBytes+2:nBytes+2+partSize]), BigEndian, &val)
		check(err)
		m.addInt32(val)
	case PartTypeInt64:
		partSize = 8
		var val int64
		err := Read(NewReader(b[nBytes+2:nBytes+2+partSize]), BigEndian, &val)
		check(err)
		m.addInt64(val)
	case PartTypeString:
		partSize = BigEndian.Uint32(b[nBytes+2 : nBytes+6])
		m.addString(string(b[nBytes+6 : nBytes+6+partSize]))
		partSize += 4 // Add length of partSize included in message for correct offset
	case PartTypeBinary:
		fmt.Println("PART_TYPE_BINARY, not supported")
		partSize = BigEndian.Uint32(b[nBytes+2 : nBytes+6])
		// TODO read data
		partSize += 4
	case PartTypeImage:
		fmt.Println("PART_TYPE_IMAGE, not supported")
		partSize = BigEndian.Uint32(b[nBytes+2 : nBytes+6])
		// TODO read data
		partSize += 4
	default:
		fmt.Println("Unkown part type", partType)

		err := errors.New("Unkown part type")
		check(err)
	}

	return partSize
}

func skipPart(b []byte, nBytes uint32) uint32 {
	partSize := uint32(0)

	switch partType := b[nBytes+1]; partType {
	case PartTypeInt32:
		partSize = 4
	case PartTypeInt64:
		partSize = 8
	case PartTypeString:
		partSize = BigEndian.Uint32(b[nBytes+2 : nBytes+6])
		partSize += 4 // Add length of partSize included in message for correct offset
	default:
		fmt.Println("Skipping not handled for part type", partType)
		err := errors.New("Skipping not handled for that part type")
		check(err)
	}

	return partSize
}

func readDate(b []byte, nBytes uint32) (uint32, string) {
	stringDate := ""
	partSize := uint32(0)
	switch partType := b[nBytes+1]; partType {
	case PartTypeInt32:
		partSize = 4
		var val int32
		err := Read(NewReader(b[nBytes+2:nBytes+2+partSize]), BigEndian, &val)
		check(err)
		stringDate = fmt.Sprintf("%v", val)
	case PartTypeInt64:
		partSize = 8
		var val int64
		err := Read(NewReader(b[nBytes+2:nBytes+2+partSize]), BigEndian, &val)
		check(err)
		t := time.Unix(val, 0)
		stringDate = fmt.Sprintf("%v", t)
	case PartTypeString:
		partSize = BigEndian.Uint32(b[nBytes+2 : nBytes+6])
		stringDate = string(b[nBytes+6 : nBytes+6+partSize])
		partSize += 4 // Add length of partSize included in message for correct offset
	default:
		fmt.Println("Date can't be parsed using part type:", partType)
		err := errors.New("Date can't be parsed using that part type")
		check(err)
	}

	return partSize, stringDate
}

func NsLoggerParse(b []byte, separator string) (string, error) {
	var fileSize = uint32(len(b))
	var nBytes = uint32(0)
	totalSize := BigEndian.Uint32(b[nBytes : nBytes+4])
	var res string

	for nBytes+totalSize < fileSize {
		nBytes += 4
		partCount := BigEndian.Uint16(b[nBytes : nBytes+2])
		nBytes += 2
		// Create new empty line
		m := logMessageString{"", separator}

		for partCount > 0 {
			usedData := uint32(0)

			formatedValue := ""

			key := b[nBytes]
			switch key {
			case PartKeyMessageType:
			case PartKeyTimestampS:
				usedData, formatedValue = readDate(b, nBytes)
			case PartKeyTimestampMs:
			case PartKeyTimestampUs:
				//usedData = skipPart(b, nBytes)
			case PartKeyThreadId:
			case PartKeyTag:
			case PartKeyLevel:
			case PartKeyMessage:
			case PartKeyImageWidth:
			case PartKeyImageHeight:
			case PartKeyMessageSeq:
				// Skip PartKeyMessageSeq as it comes before date and thus shift date column from line to line
				usedData = skipPart(b, nBytes)
			case PartKeyFilename:
			case PartKeyLinenumber:
			case PartKeyFunctionname:
			case PartKeyClientName:
			case PartKeyClientVersion:
			case PartKeyOsName:
			case PartKeyOsVersion:
			case PartKeyClientModel:
			case PartKeyUniqueid:
			default:
				return res, errors.New("Unkown part key")
			}

			if usedData != 0 {
				m.addString(formatedValue)
			} else {
				usedData = appendValue(b, nBytes, &m)
			}

			partCount--
			nBytes += (2 + usedData)
		}

		res += (m.String() + "\n")

		// nBytes = nBytes + totalSize
		totalSize = BigEndian.Uint32(b[nBytes : nBytes+4])
	}

	return res, nil
}

