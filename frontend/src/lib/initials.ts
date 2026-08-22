/**
 * The letters shown when a person has no avatar image.
 *
 * Two letters from the name when there are two words, one otherwise, and the
 * first letter of the email address as the last resort. Never more than two:
 * three letters in a 32px circle stops being legible.
 */
export function initialsOf(name: string, email = ""): string {
  const words = name.trim().split(/\s+/).filter(Boolean);
  if (words.length >= 2) {
    return (words[0][0] + words[words.length - 1][0]).toUpperCase();
  }
  if (words.length === 1) {
    return words[0].slice(0, 2).toUpperCase();
  }
  const local = email.trim();
  return local ? local.slice(0, 1).toUpperCase() : "?";
}
