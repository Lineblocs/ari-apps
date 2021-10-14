package main

import (
	"os"
		"log"
	"io"
	"bufio"
	"fmt"
	"strconv"
	"github.com/zaf/resample"
)

func main() {
	 // open input file
    output, err := os.Create("new.wav")
    if err != nil {
        panic(err)
    }

	 // open input file
    input, err := os.Open("input.mp3")
    if err != nil {
        panic(err)
    }
	channels := 1
	res, err := resample.New(output, 41000, 8000, channels, resample.I16, resample.LowQ)
	if err != nil {
		panic( err )
	}

	nBytes, nChunks := int64(0), int64(0)
	r := bufio.NewReader(input)
	last_buf, err := r.ReadAll( r )
	if err != nil {
		fmt.Println("error: " + err.Error())
		return
	}

	i, err := res.Write(last_buf)
	res.Reset(output)
	if err != nil {
		fmt.Println("error: " + err.Error())
		return
	}

	log.Println("Bytes:", nBytes, "Chunks:", nChunks)
}