package logging

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestInit_DefaultLevel(t *testing.T) {
	Init(0)
	if Log.GetLevel() != logrus.WarnLevel {
		t.Errorf("Init(0) level = %v, want WarnLevel", Log.GetLevel())
	}
}

func TestInit_InfoLevel(t *testing.T) {
	Init(1)
	if Log.GetLevel() != logrus.InfoLevel {
		t.Errorf("Init(1) level = %v, want InfoLevel", Log.GetLevel())
	}
	Init(0) // reset
}

func TestInit_DebugLevel(t *testing.T) {
	Init(2)
	if Log.GetLevel() != logrus.DebugLevel {
		t.Errorf("Init(2) level = %v, want DebugLevel", Log.GetLevel())
	}
	Init(0) // reset
}

func TestInit_HighVerbosity(t *testing.T) {
	Init(5)
	if Log.GetLevel() != logrus.DebugLevel {
		t.Errorf("Init(5) level = %v, want DebugLevel", Log.GetLevel())
	}
	Init(0) // reset
}

func TestSetOutput(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	Init(1)

	Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("SetOutput: expected output to contain 'test message', got %q", output)
	}

	// Reset
	Init(0)
}

func TestWithField(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	Init(1)

	WithField("key", "value").Info("field test")

	output := buf.String()
	if !strings.Contains(output, "key") || !strings.Contains(output, "value") {
		t.Errorf("WithField: expected output to contain field, got %q", output)
	}

	Init(0)
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	Init(1)

	WithFields(map[string]any{
		"method": "GET",
		"url":    "/groups",
	}).Info("multi-field test")

	output := buf.String()
	if !strings.Contains(output, "method") || !strings.Contains(output, "GET") {
		t.Errorf("WithFields: expected output to contain fields, got %q", output)
	}

	Init(0)
}

func TestDebugf_NotShownAtWarnLevel(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	Init(0) // warn level

	Debugf("should not appear")

	if buf.Len() > 0 {
		t.Errorf("Debugf at WarnLevel should produce no output, got %q", buf.String())
	}
}

func TestDebugf_ShownAtDebugLevel(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	Init(2) // debug level

	Debugf("should appear: %s", "test")

	output := buf.String()
	if !strings.Contains(output, "should appear: test") {
		t.Errorf("Debugf at DebugLevel should produce output, got %q", output)
	}

	Init(0)
}

func TestHTTPRequest(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	Init(2) // debug level

	HTTPRequest("GET", "https://api.example.com/groups", 200)

	output := buf.String()
	if !strings.Contains(output, "GET") || !strings.Contains(output, "200") {
		t.Errorf("HTTPRequest: expected method and status, got %q", output)
	}

	Init(0)
}

func TestHTTPRequestStart(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	Init(2)

	HTTPRequestStart("POST", "https://api.example.com/groups")

	output := buf.String()
	if !strings.Contains(output, "POST") {
		t.Errorf("HTTPRequestStart: expected method, got %q", output)
	}

	Init(0)
}

func TestWarnf(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	Init(0) // warn level — warnings should appear

	Warnf("warning: %s", "test")

	output := buf.String()
	if !strings.Contains(output, "warning: test") {
		t.Errorf("Warnf should produce output at WarnLevel, got %q", output)
	}

	Init(0)
}

func TestErrorf(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	Init(0)

	Errorf("error: %s", "test")

	output := buf.String()
	if !strings.Contains(output, "error: test") {
		t.Errorf("Errorf should produce output at WarnLevel, got %q", output)
	}

	Init(0)
}
