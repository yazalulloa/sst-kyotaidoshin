package util

import (
	"strings"
)

type MultiError struct {
	Errors []error
}

func (m *MultiError) Error() string {
	if len(m.Errors) == 0 {
		return "no errors"
	}

	errorStrings := make([]string, len(m.Errors))
	for i, err := range m.Errors {
		errorStrings[i] = err.Error()
	}
	return strings.Join(errorStrings, "\n") // Join errors with newlines
}

func (m *MultiError) InsertAt(index int, err error) {
	if err != nil {
		m.Errors[index] = err
	}
}

func (m *MultiError) Add(err error) {
	if err != nil {
		m.Errors = append(m.Errors, err)
	}
}

func (m *MultiError) HasErrors() bool {
	return len(m.Errors) > 0
}
