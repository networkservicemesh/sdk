package logger

import (
	"context"
	"testing"

	"github.com/networkservicemesh/sdk/pkg/tools/jaeger"
)

/*
func f() {
	//var t_ = T{7}
	var l = logrus.New()
	l.Formatter = new(logrus.JSONFormatter)
	l.Formatter = new(logrus.TextFormatter)                     //default
	l.Formatter.(*logrus.TextFormatter).DisableColors = true    // remove colors
	l.Formatter.(*logrus.TextFormatter).DisableTimestamp = true // remove timestamp from test output
	l.Level = logrus.TraceLevel
	l.Out = os.Stdout
	l.WithFields(logrus.Fields{
		"animal": "walrus",
		"number": 0,
	}).Error("a", 1)
	fmt.Println(123)
}

type st struct {
	a int
	b string
}

func g() {
	closer := jaeger.InitJaeger("xyz")
	defer closer.Close()
	c := context.Background()
	span := spanhelper.FromContext(c, "operation_operation")
	//span.logEntry()
	defer span.Finish()
	x := st{a: 1, b: "test"}
	span.LogObject("test_object", x)
	span.Span().SetTag("mykey", "myvalue")
	span.Span().LogFields(log.Int("huint", 12345))
}
*/
func log1(ctx context.Context) Logger {
	c := context.WithValue(ctx, CTXKEY_LOGGERTYPE, LOGGERTYPE_LOGRUS)
	c = context.WithValue(c, CTXKEY_LOGGERLEVEL, WARN)
	log := Log(c)
	log.Info("info1", "data")
	log.Warn("warning1", "dangerous")
	log.Error("error1", "critical")
	return log
}

func log2(ctx context.Context) Logger {
	c := context.WithValue(ctx, CTXKEY_LOGGERTYPE, LOGGERTYPE_SPAN)
	c = context.WithValue(c, CTXKEY_SPANLOGGER_OP, "O P E R A T I O N")
	c = context.WithValue(c, CTXKEY_LOGGERLEVEL, WARN)
	log := Log(c)
	log.Info("info2", "data")
	log.Warn("warning2", "dangerous")
	log.Error("error2", "critical")
	return log
}

func log3(ctx context.Context, l1, l2 Logger) {
	c := context.WithValue(ctx, CTXKEY_LOGGER, l1)
	l1 = Log(c)
	c = context.WithValue(ctx, CTXKEY_LOGGERTYPE, LOGGERTYPE_GROUP)
	c = context.WithValue(c, CTXKEY_GL_SLICE, []Logger{l1, l2})
	log := Log(c)
	log.Info("info3", "data")
	log.Warn("warning3", "dangerous")
	log.Error("error3", "critical")
}

func TestLogger() {
	closer := jaeger.InitJaeger("TEST")
	defer closer.Close()
	ctx_raw := context.Background()
	log1 := log1(ctx_raw)
	log2 := log2(ctx_raw)
	defer log2.(*spanLogger).Close()
	log3(ctx_raw, log1, log2)
}

func Test_f(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
	TestLogger()
}
