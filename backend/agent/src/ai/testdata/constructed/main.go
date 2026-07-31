// The control program for AI-06.3 item 2. It MUST build.
//
// Its only job is to make the failure of ../handrolled mean something. A test
// that asserts "this program does not compile" passes just as happily when the
// import path is misspelled, when the module does not resolve, when the
// toolchain is missing, or when package ai does not build at all. This program
// is written against the same package, through the same import path, from the
// same directory, using only the exported constructor — so if it builds and the
// other one does not, the difference is the seal and nothing else.
//
// It is also the readable half of the contract stated as a program rather than
// as a test: this is every line an adapter needs, and there is no other way in.
package main

import (
	"fmt"

	"github.com/cachicamas/backend/agent/src/ai"
)

func main() {
	part, err := ai.NewText("hello")
	if err != nil {
		panic(err)
	}

	msg, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		panic(err)
	}

	for _, got := range msg.Content() {
		switch got.Kind() {
		case ai.PartKindText:
			text, ok := got.Text()
			fmt.Println(len(text), ok)
		default:
			fmt.Println("unhandled kind", got.Kind())
		}
	}
}
