// Document mirrors the backend model.Document summary shape returned by
// /api/documents. Only the fields the UI currently uses are declared.
export interface Document {
  id: number
  title: string
  url: string | null
  updatedAt: string
  isArchived: boolean
  labels: string[]
  // Permalinks of posts published from this document (e.g. Reddit). May be null
  // on summary responses; guard with ?? [].
  postUrls: string[]
}

// RedditPost is one newest post from the user's subreddit, as returned by
// GET /api/reddit/posts. url is the permalink to the Reddit post.
export interface RedditPost {
  title: string
  url: string
}

// Response mirrors model.Response — one generated answer attached to a block.
export interface Response {
  id: number
  blockId: number
  value: string
  model: string
  position: number
}

// Block mirrors model.Block. attributes are the mode's key/values; responses
// accumulate each run's output.
export interface Block {
  id: number
  documentId: number
  mode: string
  position: number
  attributes: Record<string, string>
  responses: Response[]
}

// Mode mirrors mode.Mode as returned by /api/modes (the mustache template is
// kept server-side, so it isn't included).
export interface Mode {
  name: string
  label: string
  keys: string[]
  output: string
}

// LLMModel mirrors llm.Model as returned by /api/models — one selectable model
// in the fixed, cross-provider list.
export interface LLMModel {
  id: string
  label: string
  provider: string
}

// DocumentDetail is the fully-populated document returned by
// GET /api/documents/:id (the summary plus its nested data).
export interface DocumentDetail extends Document {
  selectedModel: string
  attributes: Record<string, string>
  blocks: Block[]
}
