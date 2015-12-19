package main

import "os"

func main() {
	pgb, err := NewPgBackups()
	if err != nil {
		panic(err)
	}

	s := NewScheduler(pgb, os.Getenv("SCHEDULE"))
	err = s.Run()

	if err != nil {
		panic(err)
	}
}
