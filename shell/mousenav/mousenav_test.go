// shell/mousenav/mousenav_test.go

package mousenav

import "testing"

func TestTheSideButtonsStepBackAndForward(t *testing.T) {
	for button, want := range map[uint]int{1: 0, 2: 0, 3: 0, 8: -1, 9: 1} {
		if got := stepOf(button); got != want {
			t.Errorf("stepOf(%d) = %d, want %d", button, got, want)
		}
	}
}

// Watch(win, nil) must not crash when the window has no webview.
func TestReportDropsAnErrorWithNoHandler(t *testing.T) {
	report(nil, ErrNoWebView)
	var got error
	report(func(err error) { got = err }, ErrNoWebView)
	if got != ErrNoWebView {
		t.Fatalf("report handed %v, want ErrNoWebView", got)
	}
}
