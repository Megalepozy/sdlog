package sdlog

import (
	"bytes"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSDLog(t *testing.T) {
	Convey("Creation of logger struct", t, func() {
		res := New()

		So(res, ShouldHaveSameTypeAs, &SDLog{})
	})

	Convey("Calling Info() with no labels", t, func() {
		expectedMsg := "The world collapsed"

		res := capturePrintedOutput(New().Info, expectedMsg, "out")

		So(res, ShouldContainSubstring, "\"severity\":\"INFO\"")
		So(res, ShouldContainSubstring, "\"message\":\""+expectedMsg+"\"")
		So(res, ShouldContainSubstring, "logging.googleapis.com/sourceLocation")
		So(res, ShouldContainSubstring, "\"labels\":{}")
	})

	Convey("Calling Info() with added label field (string)", t, func() {
		expectedMsg := "I want a donut"

		res := capturePrintedOutput(New().Lbl("cat", "is cute").Info, expectedMsg, "out")

		So(res, ShouldContainSubstring, "\"severity\":\"INFO\"")
		So(res, ShouldContainSubstring, "\"message\":\""+expectedMsg+"\"")
		So(res, ShouldContainSubstring, "\"labels\":{\"cat\":\"is cute\"}")
		So(res, ShouldNotContainSubstring, "\"logTracingID\":")
	})

	Convey("Calling Info() with multiple added label fields (string, int, bool)", t, func() {
		expectedMsg := "I want a donut2"

		res := capturePrintedOutput(New().Lbl("cat", "is cute").Lbl("# of dogs", 7).Lbl("bool", false).Info,
			expectedMsg, "out")

		So(res, ShouldContainSubstring, "\"severity\":\"INFO\"")
		So(res, ShouldContainSubstring, "\"message\":\""+expectedMsg+"\"")
		So(res, ShouldContainSubstring, "\"cat\":\"is cute\"")
		So(res, ShouldContainSubstring, "\"# of dogs\":\"7\"")
		So(res, ShouldContainSubstring, "\"bool\":\"false\"")
		So(res, ShouldNotContainSubstring, "\"logTracingID\":")
	})

	Convey("Calling Info() with added logTracingID label field", t, func() {
		errTracingID := uuid.New().String()

		res := capturePrintedOutput(New().AddLogTracingID(errTracingID).Info, "a", "out")

		So(res, ShouldContainSubstring, "\"severity\":\"INFO\"")
		So(res, ShouldContainSubstring, "\"labels\":{\"logTracingID\":\""+errTracingID+"\"")
	})

	Convey("Calling Error() with no labels", t, func() {
		expectedMsg := "The world collapsed!!"

		outID := make(chan string)
		res := capturePrintedAndReturnedOutput(New().Error, expectedMsg, "err", outID)

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

			s := []struct {
				i int
				b bool
			}{
				{2, true},
				{3, false},
			}

			e := fmt.Errorf("got error %s", "yey")

			outID := make(chan string)
			res := capturePrintedAndReturnedOutput(New().Lbl("[]struct", s).Lbl("err", e).Error,
				expectedMsg, "err", outID)

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

func capturePrintedOutput(f func(msg string), msg string, stream string) string {
	w, formerState, outC := prepareCapturingOutput(stream)

	f(msg)

	w.Close()
	out := <-outC

	returnOutputStreamToFormerState(stream, formerState)

	return out
}

func capturePrintedAndReturnedOutput(f func(msg string) string, msg string, stream string, outID chan string) string {
	w, formerState, outC := prepareCapturingOutput(stream)

	errTracingID := f(msg)
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
