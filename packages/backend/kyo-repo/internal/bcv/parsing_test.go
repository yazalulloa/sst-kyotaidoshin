package bcv

import (
	"fmt"
	"testing"
)

func TestToASCII(t *testing.T) {
	str := "😀 was here, what the fuck is this, 20250232"
	ascii := toASCII(str)
	fmt.Println(ascii)
}
