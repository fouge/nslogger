package nslogger

import "fmt"


type logMessage interface {
	addString(value string)
	addInt16(value int16)
	addInt32(value int32)
	addInt64(value int64)
}

type logMessageString struct {
	value     string
	separator string
}

func (t *logMessageString) String() string {
	return (t).value
}

func (t *logMessageString) addString(value string) {
	if value != "" {
		t.value += (value + t.separator)
	}
}

func (t *logMessageString) addInt16(value int16) {
	t.value += fmt.Sprintf("%v"+t.separator, value)
}

func (t *logMessageString) addInt32(value int32) {
	t.value += fmt.Sprintf("%v"+t.separator, value)
}

func (t *logMessageString) addInt64(value int64) {
	t.value += fmt.Sprintf("%v"+t.separator, value)
}
