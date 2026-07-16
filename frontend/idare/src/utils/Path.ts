export function pathJoin(...parts: any[]) {
  const separator = '/'
  parts = parts.map((part, index) => {
    part = String(part)
    if (index)
      part = part.replace(new RegExp(`^${separator}`), '')

    if (index !== parts.length - 1)
      part = part.replace(new RegExp(`${separator}$`), '')

    return part
  })

  return parts.join(separator)
}
