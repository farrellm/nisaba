// Shared string helpers.

// A single module-level Collator: constructing one per comparison would
// dominate the cost of sorting large lists.
export const collator = new Intl.Collator(undefined, { sensitivity: 'base' })

// Labels are user-global and lowercase-unique, but match case-insensitively to
// stay robust to legacy data.
export const sameName = (a: string, b: string) => a.toLowerCase() === b.toLowerCase()

// stripPromptTag removes the "[WP]" writing-prompt tag (case-insensitive) from
// a Reddit post title and trims the result.
export function stripPromptTag(title: string): string {
  return title.replace(/\[wp\]/gi, '').trim()
}
