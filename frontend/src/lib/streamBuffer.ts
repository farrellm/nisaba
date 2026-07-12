// StreamBuffer accumulates streamed text deltas and notifies one subscriber
// with the full text so far. It decouples the component that receives deltas
// (BlockCard's run handler) from the component that renders them
// (StreamingPreview): deltas that arrive before the preview mounts are kept,
// and per-delta re-renders stay inside the preview.
export class StreamBuffer {
  private text = ''
  private listener: ((text: string) => void) | null = null

  push(delta: string) {
    this.text += delta
    this.listener?.(this.text)
  }

  // subscribe immediately replays the text accumulated so far, then forwards
  // each subsequent push. Returns an unsubscribe function.
  subscribe(listener: (text: string) => void): () => void {
    this.listener = listener
    if (this.text !== '') listener(this.text)
    return () => {
      if (this.listener === listener) this.listener = null
    }
  }
}
