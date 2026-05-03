const HTML_TAG_PATTERN = /<[^>]*>/g;

const decodeHtmlEntities = (value: string) =>
  value
    .replaceAll("&nbsp;", " ")
    .replaceAll("&amp;", "&")
    .replaceAll("&quot;", "\"")
    .replaceAll("&#39;", "'")
    .replaceAll("&lt;", "<")
    .replaceAll("&gt;", ">");

export function sanitizeDisplayText(value: string) {
  return decodeHtmlEntities(value).replace(HTML_TAG_PATTERN, "").replace(/\s+/g, " ").trim();
}
