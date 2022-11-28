package main

import (
	"errors"
	"fmt"
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation"
)

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}

func handleFileErr(filepath string, err error) {
	if err != nil {
		fmt.Println("Help topic validation error. Error occured in: ", filepath)
		handleErr(err)
	}
}

func notMatch(r string, msg string) validation.RuleFunc {
	return func(value interface{}) error {
		s := value.(string)
		matched, err := regexp.MatchString(r, s)
		if err != nil {
			return nil
		}

		if matched {
			return errors.New(msg)
		}
		return nil
	}
}
