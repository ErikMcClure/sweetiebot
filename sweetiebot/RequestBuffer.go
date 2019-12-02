package sweetiebot

import (
	"strings"

	"github.com/erikmcclure/discordgo"
)

type sbRequestBuffer struct {
	buffer []*discordgo.MessageSend
	count  int
}

func (b *sbRequestBuffer) Append(m *discordgo.MessageSend) int {
	b.buffer = append(b.buffer, m)
	b.count += len(m.Content)
	if b.count+len(b.buffer) >= 1999 { // add one for each message in the buffer for added newlines
		return len(b.buffer)
	}
	return 0
}

func (b *sbRequestBuffer) Process() (*discordgo.MessageSend, int) {
	if len(b.buffer) < 1 {
		return nil, 0
	}

	if len(b.buffer) == 1 {
		msg := b.buffer[0]
		b.buffer = nil
		b.count = 0
		return msg, 0
	}

	count := len(b.buffer[0].Content)
	msg := make([]string, 1, len(b.buffer))
	msg[0] = b.buffer[0].Content
	i := 1

	for i < len(b.buffer) && (count+i+len(b.buffer[i].Content)) < 2000 {
		msg = append(msg, b.buffer[i].Content)
		count += len(b.buffer[i].Content)
		i++
	}

	b.count -= count
	if i >= len(b.buffer) {
		b.buffer = nil
	} else {
		b.buffer = b.buffer[i:]
	}

	return &discordgo.MessageSend{
		Content: strings.Join(msg, "\n"),
	}, len(b.buffer)
}
