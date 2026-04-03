package utils

import "log"

func LogError(name string, err error) {
	log.Printf("\n==[ERROR][%s]: %v==", name, err)
}

func LogInfo(name string, msg string) {
	log.Printf("\n==[INFO][%s]: %v==", name, msg)
}