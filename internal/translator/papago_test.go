package translator

import (
	"bytes"
	"testing"
)

var chunks = []bytes.Buffer{
	*bytes.NewBufferString("Hello, World!"),
	*bytes.NewBufferString("OwO What's this?"),
	*bytes.NewBufferString("$"),
	*bytes.NewBufferString(`1234567890!@#$%^&*()-+'"`),
}

func TestPapago(t *testing.T) {
	sequences := <-WithPapagoAsChunks(chunks, "en", "ko")
	t.Logf("%+v", sequences)
}
