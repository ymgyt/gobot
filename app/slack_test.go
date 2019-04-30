package app_test

import (
	"testing"
	"time"

	"github.com/ymgyt/gobot/app"
)

func TestDuplicationChecker_CheckDuplicateNotification(t *testing.T) {
	checker := &app.DuplicationChecker{}
	event := "https://github.com/ymgyt/gobot/pull/9"
	d := 3 * time.Second

	ok := checker.CheckDuplicateNotification(event, d)
	if !ok {
		t.Fatal("initial notification should be ok, but not ok")
	}

	ok = checker.CheckDuplicateNotification(event, d)
	if ok {
		t.Fatal("duplicate notification should not be ok, bug got ok")
	}
}
