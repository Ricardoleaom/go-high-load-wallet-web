// A minimal Server-Sent Events parser over a byte stream, for use with
// fetch() instead of EventSource: EventSource cannot send an Authorization
// header, which every load lab request needs.
export interface SSEEvent {
  event: string
  data: string
}

// parseSSE yields one event per blank-line-terminated block, buffering
// across chunk boundaries (an event's lines, or even one line's bytes, may
// arrive split across multiple reads). A block with no "data:" line (a
// bare comment, used as a keep-alive) yields nothing.
export async function* parseSSE(stream: ReadableStream<Uint8Array>): AsyncGenerator<SSEEvent> {
  const reader = stream.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  try {
    for (;;) {
      const { done, value } = await reader.read()
      if (done) {
        break
      }
      buffer += decoder.decode(value, { stream: true })
      let boundary = buffer.indexOf('\n\n')
      while (boundary !== -1) {
        const block = buffer.slice(0, boundary)
        buffer = buffer.slice(boundary + 2)
        const parsed = parseBlock(block)
        if (parsed) {
          yield parsed
        }
        boundary = buffer.indexOf('\n\n')
      }
    }
  } finally {
    reader.releaseLock()
  }
}

function parseBlock(block: string): SSEEvent | null {
  let event = 'message'
  const dataLines: string[] = []
  for (const line of block.split('\n')) {
    if (line.startsWith('event: ')) {
      event = line.slice('event: '.length)
    } else if (line.startsWith('data: ')) {
      dataLines.push(line.slice('data: '.length))
    }
  }
  if (dataLines.length === 0) {
    return null
  }
  return { event, data: dataLines.join('\n') }
}
