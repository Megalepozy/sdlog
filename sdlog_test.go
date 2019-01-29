package sdlog

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"testing"

	"github.com/google/uuid"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSDLog(t *testing.T) {
	Convey("Creation of logger struct", t, func() {
		res := New()

		So(res, ShouldHaveSameTypeAs, &SDLog{})
	})

	Convey("Calling Info() with no labels", t, func() {
		expectedMsg := "The world collapsed"
		sdlogStruct := New()

		res := capturePrintedOutput("out", sdlogStruct.Info, expectedMsg)

		So(res, ShouldContainSubstring, "\"severity\":\"INFO\"")
		So(res, ShouldContainSubstring, "\"message\":\""+expectedMsg+"\"")
		So(res, ShouldContainSubstring, "logging.googleapis.com/sourceLocation")
		So(res, ShouldContainSubstring, "\"labels\":{}")
	})

	Convey("Calling Info() with added label field (string)", t, func() {
		expectedMsg := "I want a donut"
		sdlogStruct := New()

		res := capturePrintedOutput("out", sdlogStruct.Info, expectedMsg, sdlogStruct.Lbl("cat", "is cute"))

		So(res, ShouldContainSubstring, "\"severity\":\"INFO\"")
		So(res, ShouldContainSubstring, "\"message\":\""+expectedMsg+"\"")
		So(res, ShouldContainSubstring, "\"labels\":{\"cat\":\"is cute\"}")
		So(res, ShouldNotContainSubstring, "\"logTracingID\":")
	})

	Convey("Calling Info() with added logTracingID label field", t, func() {
		errTracingID := uuid.New().String()
		sdlogStruct := New()

		res := capturePrintedOutput("out", sdlogStruct.Info, "a", sdlogStruct.AddLogTracingID(errTracingID))

		So(res, ShouldContainSubstring, "\"severity\":\"INFO\"")
		So(res, ShouldContainSubstring, "\"labels\":{\"logTracingID\":\""+errTracingID+"\"")
	})

	Convey("Calling Info() with multiple added label fields (string, int, bool, logTracingID)", t, func() {
		expectedMsg := "I want a donut2"
		sdlogStruct := New()

		res := capturePrintedOutput("out", sdlogStruct.Info, expectedMsg, sdlogStruct.Lbl("cat", "is cute"),
			sdlogStruct.Lbl("# of dogs", 7), sdlogStruct.Lbl("bool", false))

		So(res, ShouldContainSubstring, "\"severity\":\"INFO\"")
		So(res, ShouldContainSubstring, "\"message\":\""+expectedMsg+"\"")
		So(res, ShouldContainSubstring, "\"cat\":\"is cute\"")
		So(res, ShouldContainSubstring, "\"# of dogs\":\"7\"")
		So(res, ShouldContainSubstring, "\"bool\":\"false\"")
		So(res, ShouldNotContainSubstring, "\"logTracingID\":")
	})

	Convey("Calling Error() with no labels", t, func() {
		expectedMsg := "The world collapsed!!"
		sdlogStruct := New()

		outID := make(chan string)
		res := capturePrintedAndReturnedOutput("err", sdlogStruct.Error, expectedMsg, outID)

		errTracingID := <-outID
		_, uuidErr := uuid.Parse(errTracingID)

		So(res, ShouldContainSubstring, "\"severity\":\"ERROR\"")
		So(res, ShouldContainSubstring, "\"message\":\""+expectedMsg+"\"")
		So(res, ShouldContainSubstring, "logging.googleapis.com/sourceLocation")
		So(res, ShouldContainSubstring, "\"labels\":{\"logTracingID\":")
		So(uuidErr, ShouldBeNil)
	})

	Convey("Calling Error() with multiple added label fields ([]struct, error)", t, func() {
		expectedMsg := "I want a donut2!!"
		sdlogStruct := New()

		s := []struct {
			i int
			b bool
		}{
			{2, true},
			{3, false},
		}

		e := fmt.Errorf("got error %s", "yey")

		outID := make(chan string)
		res := capturePrintedAndReturnedOutput("err", sdlogStruct.Error, expectedMsg, outID,
			sdlogStruct.Lbl("[]struct", s), sdlogStruct.Lbl("err", e))

		errTracingID := <-outID
		_, uuidErr := uuid.Parse(errTracingID)

		So(res, ShouldContainSubstring, "\"severity\":\"ERROR\"")
		So(res, ShouldContainSubstring, "\"message\":\""+expectedMsg+"\"")
		So(res, ShouldContainSubstring,
			"\"[]struct\":\"[]struct { i int; b bool }{struct { i int; b bool }{i:2, b:true}, struct { i int; b bool }{i:3, b:false}}\"")
		So(res, ShouldContainSubstring, "\"err\":\"got error yey\"")
		So(res, ShouldContainSubstring, "\"logTracingID\":")
		So(uuidErr, ShouldBeNil)
	})
}

func capturePrintedOutput(stream string, f func(msg string, options ...func(s *SDLog)), msg string,
	options ...func(s *SDLog)) string {
	w, formerState, outC := prepareCapturingOutput(stream)

	f(msg, options...)

	w.Close()
	out := <-outC

	returnOutputStreamToFormerState(stream, formerState)

	return out
}

func capturePrintedAndReturnedOutput(stream string, f func(msg string, options ...func(s *SDLog)) string, msg string,
	outID chan string, options ...func(s *SDLog)) string {
	w, formerState, outC := prepareCapturingOutput(stream)

	errTracingID := f(msg, options...)
	go func() {
		outID <- errTracingID
	}()

	w.Close()
	out := <-outC

	returnOutputStreamToFormerState(stream, formerState)

	return out
}

func prepareCapturingOutput(stream string) (w *os.File, old *os.File, outC chan string) {
	r, w, err := os.Pipe()
	if err != nil {
		log.Fatal(err)
	}

	if stream == "out" {
		old = os.Stdout
		os.Stdout = w
	} else {
		old = os.Stderr
		os.Stderr = w
	}

	outC = make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	return w, old, outC
}

func returnOutputStreamToFormerState(stream string, formerState *os.File) {
	if stream == "out" {
		os.Stdout = formerState
	} else {
		os.Stderr = formerState
	}
}
