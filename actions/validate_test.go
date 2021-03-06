package actions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/mihaitodor/wormhole/config"
	"github.com/mihaitodor/wormhole/inventory"
	"github.com/mihaitodor/wormhole/transport"
	. "github.com/smartystreets/goconvey/convey"
)

type dummyConnection struct {
	Server *inventory.Server
}

func (*dummyConnection) Close() error { return nil }
func (*dummyConnection) Exec(context.Context, bool, transport.ExecCallbackFunc) error {
	return nil
}
func (*dummyConnection) GetAddress() string { return "" }
func (c *dummyConnection) GetHost() string  { return c.Server.Host }
func (*dummyConnection) SetError(error)     {}

func Test_Run(t *testing.T) {
	Convey("ValidateAction.Run()", t, func(c C) {
		bodyContent := "test body"
		executedRetries := 0
		returnError := false
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executedRetries++

			if returnError {
				http.Error(w, "", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(bodyContent))
			c.So(err, ShouldBeNil)
		}))

		u, err := url.Parse(ts.URL)
		So(err, ShouldBeNil)

		conn := &dummyConnection{
			Server: &inventory.Server{
				Host: u.Host,
			},
		}

		action := ValidateAction{
			Scheme:      u.Scheme,
			Retries:     1,
			Timeout:     100 * time.Millisecond,
			StatusCode:  200,
			BodyContent: bodyContent,
		}

		Convey("should be successful under normal conditions", func() {
			err := action.Run(context.Background(), conn, config.Config{})
			So(err, ShouldBeNil)
			So(executedRetries, ShouldEqual, 1)
		})

		Convey("should fail when the URL scheme is invalid", func() {
			action.Scheme = ":"
			err := action.Run(context.Background(), conn, config.Config{})
			So(err.Error(), ShouldContainSubstring, "failed to create http request")
		})

		Convey("should fail when the retries are exhausted", func() {
			returnError = true
			action.Retries = 2
			err := action.Run(context.Background(), conn, config.Config{})
			So(err.Error(), ShouldContainSubstring, "expected status 200 but got 500 instead")
			So(executedRetries, ShouldEqual, 2)
		})
	})
}
