package main
import (
	"bufio"
	//"fmt"
	"io"
	"os"
	"sync"
	"time"
)

func main() {
	queue := struct {
		sentences []string
		position int
	} {}
	eof := false
	cond := sync.NewCond(&sync.Mutex{})
	go func() {
		r := bufio.NewReader(os.Stdin)
		sentence:=[]byte(nil)
		var previousByte byte
		for {
			b, err := r.ReadByte()
			switch err {
			case nil:
				if b == 0x03 {
					cond.L.Lock()
					queue.sentences = queue.sentences[:0]
					queue.position = 0
					cond.L.Unlock(); cond.Signal()
				} else {
				previousByteSwitch:
					switch previousByte {
					case '\n':
						sentence = append(sentence, b)
						if b == '\n' {
							cond.L.Lock()
							queue.sentences = append(queue.sentences, string(sentence))
							cond.L.Unlock(); cond.Signal()
							sentence = sentence[:0]
						}
					case '.','?','!':
						switch b {
						case '.','?','!':
						default:
							cond.L.Lock()
							queue.sentences = append(
								queue.sentences,
								string(append(sentence, '\n')),
							)
							cond.L.Unlock(); cond.Signal()
							sentence = sentence[:0]
							if b == '\n' {break previousByteSwitch}
						}
						fallthrough
					default:
						sentence = append(sentence, b)
					}
				}
				previousByte = b
			case io.EOF:
				cond.L.Lock()
				queue.sentences = append(queue.sentences, string(sentence))
				eof = true
				cond.L.Unlock(); cond.Signal()
				return
			default:
				panic(err)
			}
		}
	} ()
	cond.L.Lock()
	defer cond.L.Unlock()
	for {
		if queue.position >= len(queue.sentences) {
			if eof {return}
			queue.sentences = queue.sentences[:0]
			queue.position = 0
			cond.Wait()
		} else {
			sentence := queue.sentences[queue.position]
			queue.position++
			cond.L.Unlock()
			if _, err := os.Stdout.WriteString(sentence); err!=nil {panic(err)}
			//fmt.Printf("%#v\n", sentence)
			if queue.position < len(queue.sentences) {
				time.Sleep(time.Duration(len(sentence))*5*time.Millisecond)
			}
			cond.L.Lock()
		}
	}
}
