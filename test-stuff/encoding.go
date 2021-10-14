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
	buf := make([]byte, 0, 4*1024)
	for {
		n, err := r.Read(buf[:cap(buf)])
		buf = buf[:n]
		if n == 0 {
			if err == nil {
				continue
			}
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}
		nChunks++
		nBytes += int64(len(buf))
		// process buf
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}

		i, err := res.Write(buf)
		res.Reset(output)
		if err != nil {
			fmt.Println("error: " + err.Error())
			continue
		}

		fmt.Println("read " + strconv.Itoa( i ) + " bytes")
		/*
		expected := 24
		if i != expected {
			t.Errorf("Resampler 1-1 writer returned: %d , expecting: %d", i, tc.expected)
		}
		*/

	}
	log.Println("Bytes:", nBytes, "Chunks:", nChunks)
}