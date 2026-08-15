package tty

import "os"

// SyncPromptLine resets the interactive terminal cursor so a following
// sudo Password: prompt on /dev/tty starts at column 0.
//
// Progress UI is written to stderr while sudo prompts on /dev/tty; both usually
// share one terminal, but wrapped status lines and CSI clears can leave the
// tty cursor mid-column. Writing CR+clear+NL directly to /dev/tty fixes that.
func SyncPromptLine() {
	f, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString("\r\033[2K\n")
}
