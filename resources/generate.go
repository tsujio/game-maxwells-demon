package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

//go:generate go run generate.go "魔王魂 効果音 ワンポイント26.mp3"
//go:generate go run generate.go "魔王魂 効果音 システム19.mp3"
//go:generate go run generate.go "魔王魂 効果音 物音05.mp3"
//go:generate go run generate.go "魔王魂 効果音 物音15.mp3"
//go:generate go run generate.go "魔王魂 効果音 システム49.mp3"

func decodeAudio(filename, ext string) []byte {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	audioContext := audio.NewContext(48000)

	var stream io.Reader
	switch ext {
	case ".mp3":
		stream, err = mp3.Decode(audioContext, bytes.NewReader(data))
	default:
		log.Fatalf("Invalid ext: %s", ext)
	}
	if err != nil {
		log.Fatal(err)
	}

	audioData, err := ioutil.ReadAll(stream)
	if err != nil {
		log.Fatal(err)
	}

	return audioData
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: go run generate.go FILENAME\n")
		os.Exit(1)
	}

	filename := os.Args[1]

	ext := filename[strings.LastIndex(filename, "."):]
	var data []byte
	switch ext {
	case ".mp3":
		data = decodeAudio(filename, ext)
	default:
		log.Fatalf("Invalid ext: %s", ext)
	}

	if err := ioutil.WriteFile(filename+".dat", data, os.ModePerm); err != nil {
		log.Fatal(err)
	}
}
