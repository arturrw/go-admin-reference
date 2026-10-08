// Mail apps and browsers choke on very long mailto: links, so stop adding
// addresses before the link passes this many characters.
const MAX_LINK = 1900

/**
 * Builds `mailto:?bcc=…&subject=…&body=…` with as many addresses as fit.
 * `included` says how many were put in, so the caller can tell the user.
 */
export function buildMailto(emails: string[], subject: string, body: string) {
  const tail = [subject && `subject=${encodeURIComponent(subject)}`, body && `body=${encodeURIComponent(body)}`].filter(Boolean)
  const link = (bcc: string[]) => 'mailto:?' + [bcc.length ? `bcc=${bcc.map(encodeURIComponent).join(',')}` : '', ...tail].filter(Boolean).join('&')
  let included = 0
  while (included < emails.length && link(emails.slice(0, included + 1)).length <= MAX_LINK) included++
  return { href: link(emails.slice(0, included)), included }
}
