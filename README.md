# xfyun_go_sdk
讯飞云golang SDK，暂时只支持语音转写
```go
package main

import (
	"fmt"
	"time"

	"git.hunantv.com/wujianqiang/tools/xfyun/raasr"
)

func main() {
	//ffmpeg -i 1.mp4 -vn -c:a aac -vbr 5 -ar 16000 kb.m4a
	client := raasr.New("******", "****")
	taskid, err := client.UploadAudio("/Users/Fang/Desktop/kb.m4a", "cn")
	content, err := client.GetProgress(taskid)
	fmt.Println(taskid, content, err)
	for {
		content, err := client.GetProgress(taskid)
		fmt.Println(taskid, content, err)

		content, err = client.GetResult(taskid)
		fmt.Println(taskid, content, err)
		time.Sleep(2 * time.Second)
	}
}

```
