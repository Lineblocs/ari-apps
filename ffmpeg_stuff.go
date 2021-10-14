package main
import (
	"github.com/u2takey/ffmpeg-go"
)


func main() {
	err := ffmpeg_go.Input("./test-stuff/input.mp3").Output("./test-stuff/song2.wav", ffmpeg_go.KwArgs{
			"acodec": "pcm_u8",
			"ar": "8000",
			}).OverWriteOutput().Run()

if err != nil {
	panic( err )
}
}
